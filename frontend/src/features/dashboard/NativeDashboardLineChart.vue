<script setup lang="ts">
import { computed } from 'vue';
import { formatValue } from '@/features/dashboard/dashboard-native';
import type { NativeDashboardSeries } from '@/interfaces/dashboard';
import { formatChartTimeLabel } from '@/utils/time';

const props = defineProps<{
  series: NativeDashboardSeries[];
  rangeMs: number;
  from?: number;
  to?: number;
  unit: string;
}>();

const VIEWBOX_WIDTH = 720;
const VIEWBOX_HEIGHT = 220;
const PADDING = {
  top: 18,
  right: 84,
  bottom: 38,
  left: 24,
};

interface ChartPoint {
  timeValue: number;
  value: number;
}

interface ChartSegment {
  key: string;
  path: string;
}

interface ChartSeriesGroup {
  series: string;
  color: string;
  points: ChartPoint[];
  segments: ChartSegment[];
}

type ScaleMode = 'linear' | 'log';

const rawSeries = computed(() => props.series.map((series, index) => ({
  name: series.name,
  color: SERIES_COLORS[index % SERIES_COLORS.length],
  points: series.points
    .map((point) => ({
      timeValue: point.time,
      value: point.value,
    }))
    .filter((point) => Number.isFinite(point.timeValue) && Number.isFinite(point.value))
    .sort((left, right) => left.timeValue - right.timeValue),
})));

const xDomain = computed(() => {
  if (
    typeof props.from === 'number'
    && Number.isFinite(props.from)
    && typeof props.to === 'number'
    && Number.isFinite(props.to)
    && props.to > props.from
  ) {
    return { min: props.from, max: props.to };
  }

  const timestamps = rawSeries.value.flatMap((series) => series.points.map((point) => point.timeValue));
  if (!timestamps.length) {
    return { min: 0, max: 1 };
  }

  return {
    min: Math.min(...timestamps),
    max: Math.max(...timestamps),
  };
});

const stepMs = computed(() => inferStepMs(rawSeries.value, xDomain.value.min, xDomain.value.max));

const scaleMode = computed<ScaleMode>(() => {
  if (props.unit !== 'ms') {
    return 'linear';
  }
  const positiveValues = rawSeries.value
    .flatMap((series) => series.points.map((point) => point.value))
    .filter((value) => value > 0);
  if (positiveValues.length < 2) {
    return 'linear';
  }
  const min = Math.min(...positiveValues);
  const max = Math.max(...positiveValues);
  return max / Math.max(min, 1) >= 20 ? 'log' : 'linear';
});

const seriesGroups = computed<ChartSeriesGroup[]>(() => rawSeries.value.map((series) => {
  const points = props.unit === 'reqps'
    ? fillMissingBuckets(series.points, xDomain.value.min, xDomain.value.max, stepMs.value)
    : series.points;
  return {
    series: series.name,
    color: series.color,
    points,
    segments: buildSegments(series.name, points, stepMs.value, props.unit),
  };
}));

const yDomain = computed(() => {
  const values = seriesGroups.value.flatMap((series) => series.points.map((point) => point.value));
  if (!values.length) {
    return { min: 0, max: 1 };
  }

  if (scaleMode.value === 'log') {
    const positiveValues = values.filter((value) => value > 0);
    if (!positiveValues.length) {
      return { min: 0.1, max: 1 };
    }
    const min = Math.min(...positiveValues);
    const max = Math.max(...positiveValues);
    return {
      min: Math.max(0.1, min / 1.4),
      max: max * 1.15,
    };
  }

  const min = props.unit === 'reqps' ? 0 : Math.min(...values);
  const max = Math.max(...values);
  const span = max - min;
  const basePadding = span > 0
    ? span * 0.12
    : Math.max(Math.abs(max || min) * 0.15, Math.abs(max || min) < 1 ? 0.1 : 1);
  const paddedMin = min - basePadding;
  const paddedMax = max + basePadding;
  const shouldClampToZero = props.unit === 'reqps' || (min >= 0 && min <= basePadding);
  return {
    min: shouldClampToZero ? 0 : paddedMin,
    max: paddedMax,
  };
});

const xTicks = computed(() => {
  const min = xDomain.value.min;
  const max = xDomain.value.max;
  if (max <= min) {
    return [];
  }
  const tickCount: number = props.rangeMs <= 60 * 60 * 1000 ? 6 : 5;
  return Array.from({ length: tickCount }, (_, index) => {
    const timeValue = tickCount === 1
      ? min
      : min + ((max - min) * index) / (tickCount - 1);
    return {
      x: toX(timeValue),
      label: formatChartTimeLabel(timeValue, props.rangeMs),
    };
  });
});

const yTicks = computed(() => {
  if (scaleMode.value === 'log') {
    return buildLogTicks(yDomain.value.min, yDomain.value.max).map((value) => ({
      y: toY(value),
      label: formatValue(value, props.unit),
    }));
  }

  const min = yDomain.value.min;
  const max = yDomain.value.max;
  const tickCount: number = 4;
  return Array.from({ length: tickCount }, (_, index) => {
    const ratio = tickCount === 1 ? 0 : index / (tickCount - 1);
    const value = max - ((max - min) * ratio);
    return {
      y: toY(value),
      label: formatValue(value, props.unit),
    };
  });
});

function toX(value: number) {
  const width = VIEWBOX_WIDTH - PADDING.left - PADDING.right;
  const span = Math.max(1, xDomain.value.max - xDomain.value.min);
  return PADDING.left + ((value - xDomain.value.min) / span) * width;
}

function toY(value: number) {
  const height = VIEWBOX_HEIGHT - PADDING.top - PADDING.bottom;
  if (scaleMode.value === 'log') {
    const min = Math.max(yDomain.value.min, 0.1);
    const max = Math.max(yDomain.value.max, min * 1.01);
    const span = Math.max(0.001, Math.log10(max) - Math.log10(min));
    const normalized = (Math.log10(Math.max(value, min)) - Math.log10(min)) / span;
    return PADDING.top + (1 - normalized) * height;
  }

  const span = Math.max(1, yDomain.value.max - yDomain.value.min);
  return PADDING.top + (1 - (value - yDomain.value.min) / span) * height;
}

function shouldRenderMarkers(points: ChartPoint[]) {
  return props.unit === 'ms' || points.length <= 96;
}

function inferStepMs(
  seriesList: Array<{ points: ChartPoint[] }>,
  from: number,
  to: number,
) {
  const deltas: number[] = [];
  seriesList.forEach((series) => {
    for (let index = 1; index < series.points.length; index += 1) {
      const delta = series.points[index]!.timeValue - series.points[index - 1]!.timeValue;
      if (delta > 0) {
        deltas.push(delta);
      }
    }
  });
  if (deltas.length) {
    deltas.sort((left, right) => left - right);
    return deltas[Math.floor(deltas.length / 2)] || 60_000;
  }

  const window = Math.max(to - from, 60_000);
  return Math.max(60_000, Math.round(window / 60));
}

function fillMissingBuckets(points: ChartPoint[], from: number, to: number, step: number) {
  if (!points.length || step <= 0 || to <= from) {
    return points;
  }

  let start = points[0]!.timeValue;
  while (start-step >= from) {
    start -= step;
  }

  let end = points[points.length - 1]!.timeValue;
  while (end + step <= to) {
    end += step;
  }

  const valueByBucket = new Map<number, number>();
  points.forEach((point) => {
    valueByBucket.set(point.timeValue, point.value);
  });

  const items: ChartPoint[] = [];
  for (let timeValue = start; timeValue <= end; timeValue += step) {
    if (timeValue < from || timeValue > to) {
      continue;
    }
    items.push({
      timeValue,
      value: valueByBucket.get(timeValue) ?? 0,
    });
  }
  return items;
}

function buildSegments(series: string, points: ChartPoint[], step: number, unit: string) {
  if (!points.length) {
    return [];
  }

  const segments: ChartSegment[] = [];
  let current: ChartPoint[] = [points[0]!];
  const maxGap = Math.max(step * (unit === 'reqps' ? 2 : 1.75), 60_000);

  for (let index = 1; index < points.length; index += 1) {
    const point = points[index]!;
    const previous = points[index - 1]!;
    if (point.timeValue - previous.timeValue > maxGap) {
      segments.push({
        key: `${series}-${segments.length}`,
        path: current.map((item) => `${toX(item.timeValue)},${toY(item.value)}`).join(' '),
      });
      current = [point];
      continue;
    }
    current.push(point);
  }

  if (current.length) {
    segments.push({
      key: `${series}-${segments.length}`,
      path: current.map((item) => `${toX(item.timeValue)},${toY(item.value)}`).join(' '),
    });
  }

  return segments.filter((segment) => segment.path.length > 0);
}

function buildLogTicks(min: number, max: number) {
  const baseTicks: number[] = [];
  const safeMin = Math.max(min, 0.1);
  const safeMax = Math.max(max, safeMin * 1.01);
  const minExponent = Math.floor(Math.log10(safeMin));
  const maxExponent = Math.ceil(Math.log10(safeMax));
  const multipliers = [1, 2, 5];

  for (let exponent = minExponent; exponent <= maxExponent; exponent += 1) {
    multipliers.forEach((multiplier) => {
      const value = multiplier * (10 ** exponent);
      if (value >= safeMin && value <= safeMax) {
        baseTicks.push(value);
      }
    });
  }

  const unique = Array.from(new Set(baseTicks)).sort((left, right) => right - left);
  if (unique.length <= 5) {
    return unique;
  }

  const items: number[] = [];
  const lastIndex = unique.length - 1;
  for (let index = 0; index < 5; index += 1) {
    const position = Math.round((lastIndex * index) / 4);
    items.push(unique[position]!);
  }
  return Array.from(new Set(items)).sort((left, right) => right - left);
}

const SERIES_COLORS = ['#1890ff', '#13c2c2', '#52c41a', '#fa8c16', '#722ed1', '#eb2f96'];
</script>

<template>
  <div class="native-line-chart">
    <div class="native-line-chart__legend">
      <span v-for="item in seriesGroups" :key="item.series" class="native-line-chart__legend-item">
        <span class="native-line-chart__legend-dot" :style="{ backgroundColor: item.color }" />
        <span class="native-line-chart__legend-text">{{ item.series }}</span>
      </span>
    </div>

    <svg
      class="native-line-chart__svg"
      :viewBox="`0 0 ${VIEWBOX_WIDTH} ${VIEWBOX_HEIGHT}`"
      preserveAspectRatio="none"
      role="img"
      aria-label="Native dashboard line chart"
    >
      <line
        v-for="tick in yTicks"
        :key="tick.y"
        class="native-line-chart__grid"
        :x1="PADDING.left"
        :x2="VIEWBOX_WIDTH - PADDING.right"
        :y1="tick.y"
        :y2="tick.y"
      />

      <template v-for="item in seriesGroups" :key="item.series">
        <polyline
          v-for="segment in item.segments"
          :key="segment.key"
          :points="segment.path"
          :stroke="item.color"
          class="native-line-chart__polyline"
        />
        <circle
          v-for="point in shouldRenderMarkers(item.points) ? item.points : []"
          :key="`${item.series}-${point.timeValue}`"
          class="native-line-chart__point"
          :cx="toX(point.timeValue)"
          :cy="toY(point.value)"
          :fill="item.color"
          r="2.6"
        />
      </template>

      <g v-for="tick in xTicks" :key="tick.x">
        <text class="native-line-chart__axis native-line-chart__axis--x" :x="tick.x" :y="VIEWBOX_HEIGHT - 8">
          {{ tick.label }}
        </text>
      </g>

      <g v-for="tick in yTicks" :key="`${tick.y}-${tick.label}`">
        <text class="native-line-chart__axis native-line-chart__axis--y" :x="VIEWBOX_WIDTH - 8" :y="tick.y - 6">
          {{ tick.label }}
        </text>
      </g>
    </svg>
  </div>
</template>

<style scoped>
.native-line-chart {
  display: grid;
  gap: 10px;
  height: 100%;
}

.native-line-chart__legend {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
}

.native-line-chart__legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.native-line-chart__legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
}

.native-line-chart__legend-text {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.native-line-chart__svg {
  width: 100%;
  height: 100%;
  min-height: 150px;
}

.native-line-chart__grid {
  stroke: rgba(16, 35, 63, 0.08);
  stroke-width: 1;
}

.native-line-chart__polyline {
  fill: none;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.native-line-chart__point {
  stroke: rgba(255, 255, 255, 0.92);
  stroke-width: 1;
}

.native-line-chart__axis {
  fill: var(--portal-text-muted);
  font-size: 11px;
}

.native-line-chart__axis--x {
  text-anchor: middle;
}

.native-line-chart__axis--y {
  text-anchor: end;
}
</style>
