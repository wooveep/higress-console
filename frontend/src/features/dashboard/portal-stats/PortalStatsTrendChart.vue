<script setup lang="ts">
import { computed } from 'vue';
import type { PortalUsageTrendPoint } from '@/interfaces/portal-stats';
import { formatChartTimeLabel, formatDateTimeDisplay } from '@/utils/time';

const props = defineProps<{
  points: PortalUsageTrendPoint[];
  rangeMs?: number;
}>();

const maxTokens = computed(() => {
  return props.points.reduce((currentMax, item) => Math.max(currentMax, item.totalTokens || 0), 0) || 1;
});

const chartBars = computed(() => {
  return props.points.map((item) => ({
    label: formatBucketLabel(item.bucketLabel, props.rangeMs),
    tooltipLabel: formatBucketTooltip(item.bucketLabel),
    height: Math.max(8, Math.round(((item.totalTokens || 0) / maxTokens.value) * 160)),
    value: item.totalTokens || 0,
    requests: item.requestCount || 0,
    costMicroYuan: item.costMicroYuan || 0,
  }));
});

function parseBucketTimestamp(value: string) {
  const normalized = value.includes('T') ? value : value.replace(' ', 'T');
  const parsed = Date.parse(normalized);
  return Number.isNaN(parsed) ? null : parsed;
}

function formatBucketLabel(value: string, rangeMs?: number) {
  const timestamp = parseBucketTimestamp(value);
  if (timestamp === null) {
    return value;
  }
  return formatChartTimeLabel(timestamp, rangeMs);
}

function formatBucketTooltip(value: string) {
  const timestamp = parseBucketTimestamp(value);
  if (timestamp === null) {
    return value;
  }
  return formatDateTimeDisplay(timestamp);
}
</script>

<template>
  <div class="portal-stats-trend">
    <div v-if="!chartBars.length" class="portal-stats-trend__empty">暂无趋势数据</div>

    <div v-else class="portal-stats-trend__bars">
      <div v-for="(item, index) in chartBars" :key="`${item.tooltipLabel}-${index}`" class="portal-stats-trend__bar">
        <div class="portal-stats-trend__tooltip">
          <strong>{{ item.tooltipLabel }}</strong>
          <span>Token: {{ item.value }}</span>
          <span>请求数: {{ item.requests }}</span>
          <span>费用(μ¥): {{ item.costMicroYuan }}</span>
        </div>
        <div class="portal-stats-trend__column" :style="{ height: `${item.height}px` }" />
        <div class="portal-stats-trend__label">{{ item.label }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.portal-stats-trend {
  padding: 16px;
  border: 1px solid #dce4f0;
  border-radius: 18px;
  background:
    linear-gradient(180deg, rgba(243, 248, 255, 0.92) 0%, rgba(255, 255, 255, 0.98) 100%),
    radial-gradient(circle at top left, rgba(34, 139, 230, 0.12), transparent 42%);
}

.portal-stats-trend__empty {
  color: #6b7280;
  text-align: center;
}

.portal-stats-trend__bars {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(72px, 1fr));
  gap: 12px;
  align-items: end;
}

.portal-stats-trend__bar {
  position: relative;
  display: grid;
  gap: 8px;
  align-items: end;
}

.portal-stats-trend__tooltip {
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.18s ease;
  position: absolute;
  left: 50%;
  bottom: calc(100% + 12px);
  transform: translateX(-50%);
  min-width: 140px;
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(12, 24, 41, 0.92);
  color: #fff;
  font-size: 12px;
  display: grid;
  gap: 4px;
  z-index: 1;
}

.portal-stats-trend__bar:hover .portal-stats-trend__tooltip {
  opacity: 1;
}

.portal-stats-trend__column {
  border-radius: 16px 16px 6px 6px;
  background: linear-gradient(180deg, #1d4ed8 0%, #38bdf8 100%);
  min-height: 8px;
}

.portal-stats-trend__label {
  font-size: 12px;
  color: #526076;
  line-height: 1.4;
  word-break: break-word;
}
</style>
