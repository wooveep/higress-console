import {
  formatChartTimeLabel,
  formatDateTimeDisplay,
  formatDateTimeLocalInputValue,
  normalizeTimestamp,
} from '@/utils/time';
import type { NativeDashboardPanel, NativeDashboardSeries } from '@/interfaces/dashboard';

export const DEFAULT_RANGE_MS = 60 * 60 * 1000;
export const RANGE_OPTIONS = [5 * 60 * 1000, 15 * 60 * 1000, 60 * 60 * 1000, 6 * 60 * 60 * 1000, 24 * 60 * 60 * 1000, 7 * 24 * 60 * 60 * 1000];
export const REFRESH_OPTIONS = [0, 15 * 1000, 30 * 1000, 60 * 1000];
export const CHART_COLORS = ['#1890ff', '#13c2c2', '#52c41a', '#fa8c16', '#722ed1', '#eb2f96'];

export interface DashboardTimeRangeState {
  rangeMs: number;
  refreshMs: number;
  endTimeMode: 'now' | 'fixed';
  fixedStartTime: string;
  fixedEndTime: string;
}

export interface DashboardEffectiveWindow {
  from: number;
  to: number;
  valid: boolean;
}

export interface DashboardLinePoint {
  timeValue: number;
  timeLabel: string;
  timeTooltip: string;
  series: string;
  value: number;
  color: string;
}

export function buildLineData(seriesList: NativeDashboardSeries[] = [], rangeMs?: number): DashboardLinePoint[] {
  const data: DashboardLinePoint[] = [];

  seriesList.forEach((series, seriesIndex) => {
    const color = CHART_COLORS[seriesIndex % CHART_COLORS.length];
    series.points.forEach((point) => {
      const timeValue = normalizeTimestamp(point.time);
      if (timeValue === null) {
        return;
      }
      data.push({
        timeValue,
        timeLabel: formatChartTimeLabel(timeValue, rangeMs),
        timeTooltip: formatDateTimeDisplay(timeValue),
        series: series.name,
        value: point.value,
        color,
      });
    });
  });

  data.sort((left, right) => {
    if (left.timeValue !== right.timeValue) {
      return left.timeValue - right.timeValue;
    }
    return left.series.localeCompare(right.series);
  });

  return data;
}

export function createDashboardTimeRangeState(initialNow = Date.now()): DashboardTimeRangeState {
  return {
    rangeMs: DEFAULT_RANGE_MS,
    refreshMs: 30 * 1000,
    endTimeMode: 'now',
    fixedStartTime: formatDateTimeLocalInputValue(initialNow - DEFAULT_RANGE_MS),
    fixedEndTime: formatDateTimeLocalInputValue(initialNow),
  };
}

export function resolveDashboardTimeWindow(state: DashboardTimeRangeState, now = Date.now()): DashboardEffectiveWindow {
  if (state.endTimeMode === 'now') {
    return {
      from: now - state.rangeMs,
      to: now,
      valid: true,
    };
  }

  const from = normalizeTimestamp(state.fixedStartTime);
  const to = normalizeTimestamp(state.fixedEndTime);
  if (from === null || to === null || from >= to) {
    return {
      from: 0,
      to: 0,
      valid: false,
    };
  }
  return { from, to, valid: true };
}

export function syncFixedDashboardTimeRange(state: DashboardTimeRangeState, baseTo?: number): DashboardTimeRangeState {
  const endTimestamp = baseTo ?? normalizeTimestamp(state.fixedEndTime) ?? Date.now();
  return {
    ...state,
    fixedEndTime: formatDateTimeLocalInputValue(endTimestamp),
    fixedStartTime: formatDateTimeLocalInputValue(endTimestamp - state.rangeMs),
  };
}

export function formatValue(value: number | null | undefined, unit: string) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return '-';
  }

  switch (unit) {
    case 'currencyCny':
      return new Intl.NumberFormat(undefined, {
        style: 'currency',
        currency: 'CNY',
        maximumFractionDigits: value >= 100 ? 0 : 2,
      }).format(value);
    case 'percentunit':
      return `${formatNumber(value * 100)}%`;
    case 'percent':
      return `${formatNumber(value)}%`;
    case 'reqps':
      return `${formatNumber(value)} req/s`;
    case 'Bps':
      return `${formatBytes(value)}/s`;
    case 'bytes':
      return formatBytes(value);
    case 'dtdurationms':
    case 'ms':
      return formatDuration(value);
    case 'ops':
      return `${formatNumber(value)} ops`;
    case 'short':
      return formatCompactNumber(value);
    default:
      return formatNumber(value);
  }
}

export function formatTableValue(value: string | number | null) {
  if (value === null || value === undefined) {
    return '-';
  }
  if (typeof value === 'number') {
    return formatNumber(value);
  }
  return value;
}

export function panelHasData(panel: NativeDashboardPanel) {
  if (panel.error) {
    return true;
  }
  if (panel.type === 'stat') {
    return panel.stat?.value !== null && panel.stat?.value !== undefined;
  }
  if (panel.type === 'timeseries') {
    return !!panel.series?.some((series) => series.points.length > 0);
  }
  if (panel.type === 'table') {
    return !!panel.table?.rows?.length;
  }
  return false;
}

function formatNumber(value: number) {
  if (Math.abs(value) >= 1000) {
    return formatCompactNumber(value);
  }
  if (Math.abs(value) >= 100) {
    return value.toFixed(1);
  }
  if (Math.abs(value) >= 10) {
    return value.toFixed(2);
  }
  return value.toFixed(3).replace(/\.?0+$/, '');
}

function formatCompactNumber(value: number) {
  return new Intl.NumberFormat(undefined, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value);
}

function formatBytes(value: number) {
  if (value === 0) {
    return '0 B';
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let current = value;
  let unitIndex = 0;
  while (current >= 1024 && unitIndex < units.length - 1) {
    current /= 1024;
    unitIndex += 1;
  }
  return `${formatNumber(current)} ${units[unitIndex]}`;
}

function formatDuration(value: number) {
  if (value >= 1000) {
    return `${formatNumber(value / 1000)} s`;
  }
  return `${formatNumber(value)} ms`;
}
