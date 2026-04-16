<script setup lang="ts">
import { computed, onBeforeUnmount, shallowRef, watch } from 'vue';
import { ReloadOutlined } from '@ant-design/icons-vue';
import NativeDashboardPanelCard from '@/features/dashboard/NativeDashboardPanelCard.vue';
import {
  panelHasData,
  RANGE_OPTIONS,
  REFRESH_OPTIONS,
  resolveDashboardTimeWindow,
  syncFixedDashboardTimeRange,
  type DashboardTimeRangeState,
} from '@/features/dashboard/dashboard-native';
import { DashboardType, type NativeDashboardData } from '@/interfaces/dashboard';
import { getNativeDashboard } from '@/services/dashboard';
import {
  formatDateTimeDisplay,
  getNowDateTimeLocalInputValue,
} from '@/utils/time';
import { useI18n } from 'vue-i18n';

const props = defineProps<{
  type: DashboardType;
  timeRange: DashboardTimeRangeState;
}>();

const emit = defineEmits<{
  (event: 'update:timeRange', value: DashboardTimeRangeState): void;
  (event: 'windowChange', value: { from: number; to: number; valid: boolean }): void;
}>();

const { t } = useI18n();

const loading = shallowRef(false);
const errorMessage = shallowRef('');
const lastUpdated = shallowRef('');
const data = shallowRef<NativeDashboardData | null>(null);
const activeRows = shallowRef<string[]>([]);

let refreshTimer: number | null = null;

const hasAnyData = computed(() => data.value?.rows.some((row) => row.panels.some((panel) => panelHasData(panel))) ?? false);
const isFixedMode = computed(() => props.timeRange.endTimeMode === 'fixed');
const canAutoRefresh = computed(() => props.timeRange.endTimeMode === 'now');
const rangeLabel = computed(() => t(isFixedMode.value ? 'dashboard.native.quickRange' : 'dashboard.native.range'));
const rangeHint = computed(() => (
  isFixedMode.value
    ? t('dashboard.native.quickRangeHint')
    : ''
));
const chartRangeMs = computed(() => props.timeRange.endTimeMode === 'fixed'
  ? Math.max(0, resolveDashboardTimeWindow(props.timeRange).to - resolveDashboardTimeWindow(props.timeRange).from)
  : props.timeRange.rangeMs);
const rangeMsModel = computed({
  get: () => props.timeRange.rangeMs,
  set: (value: number) => {
    const nextState = props.timeRange.endTimeMode === 'fixed'
      ? syncFixedDashboardTimeRange({ ...props.timeRange, rangeMs: value })
      : { ...props.timeRange, rangeMs: value };
    emit('update:timeRange', nextState);
  },
});
const refreshMsModel = computed({
  get: () => props.timeRange.refreshMs,
  set: (value: number) => emit('update:timeRange', { ...props.timeRange, refreshMs: value }),
});
const endTimeModeModel = computed({
  get: () => props.timeRange.endTimeMode,
  set: (value: 'now' | 'fixed') => {
    if (value === 'fixed') {
      emit('update:timeRange', syncFixedDashboardTimeRange({ ...props.timeRange, endTimeMode: value }, Date.now()));
      return;
    }
    const now = Date.now();
    emit('update:timeRange', {
      ...props.timeRange,
      endTimeMode: value,
      fixedEndTime: getNowDateTimeLocalInputValue(),
      fixedStartTime: formatDateTimeDisplay(now - props.timeRange.rangeMs).replace(' ', 'T').slice(0, 16),
    });
  },
});
const fixedStartTimeModel = computed({
  get: () => props.timeRange.fixedStartTime,
  set: (value: string) => emit('update:timeRange', { ...props.timeRange, fixedStartTime: value }),
});
const fixedEndTimeModel = computed({
  get: () => props.timeRange.fixedEndTime,
  set: (value: string) => emit('update:timeRange', { ...props.timeRange, fixedEndTime: value }),
});

async function load() {
  const effectiveWindow = resolveDashboardTimeWindow(props.timeRange);
  emit('windowChange', effectiveWindow);
  if (!effectiveWindow.valid) {
    data.value = null;
    errorMessage.value = t('dashboard.native.invalidTimeRange');
    return;
  }
  loading.value = true;
  errorMessage.value = '';
  try {
    const result = await getNativeDashboard(props.type, {
      from: effectiveWindow.from,
      to: effectiveWindow.to,
    });
    data.value = result;
    lastUpdated.value = formatDateTimeDisplay(Date.now());
    if (!activeRows.value.length) {
      activeRows.value = result.rows.filter((row) => !row.collapsed).map((row) => row.title);
    }
  } catch (error: any) {
    data.value = null;
    errorMessage.value = String(error?.response?.data?.message || error?.message || t('dashboard.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function setupAutoRefresh() {
  if (refreshTimer) {
    window.clearInterval(refreshTimer);
    refreshTimer = null;
  }
  if (canAutoRefresh.value && props.timeRange.refreshMs > 0) {
    refreshTimer = window.setInterval(() => {
      void load();
    }, props.timeRange.refreshMs);
  }
}

function translateText(group: 'titles' | 'series' | 'columns' | 'values', value?: string) {
  if (!value) {
    return value || '';
  }
  const key = `dashboard.native.${group}.${value}`;
  const translated = t(key);
  return translated === key ? value : translated;
}

watch(() => props.type, () => {
  activeRows.value = [];
  void load();
}, { immediate: true });

watch(
  () => [
    props.timeRange.rangeMs,
    props.timeRange.endTimeMode,
    props.timeRange.fixedStartTime,
    props.timeRange.fixedEndTime,
  ],
  () => {
    void load();
  },
);

watch([() => props.timeRange.refreshMs, canAutoRefresh], setupAutoRefresh, { immediate: true });

onBeforeUnmount(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer);
  }
});
</script>

<template>
  <div class="native-dashboard">
    <div class="native-dashboard__toolbar">
      <div class="native-dashboard__controls">
        <div class="native-dashboard__control">
          <span class="native-dashboard__label">{{ t('dashboard.native.endTimeMode') }}</span>
          <a-select
            v-model:value="endTimeModeModel"
            :options="[
              { value: 'now', label: t('dashboard.native.endTimeModes.now') },
              { value: 'fixed', label: t('dashboard.native.endTimeModes.fixed') },
            ]"
          />
        </div>
        <div v-if="timeRange.endTimeMode === 'fixed'" class="native-dashboard__control">
          <span class="native-dashboard__label">{{ t('dashboard.native.startTime') }}</span>
          <input
            v-model="fixedStartTimeModel"
            class="native-dashboard__datetime-input"
            type="datetime-local"
          />
        </div>
        <div v-if="timeRange.endTimeMode === 'fixed'" class="native-dashboard__control">
          <span class="native-dashboard__label">{{ t('dashboard.native.endTime') }}</span>
          <input
            v-model="fixedEndTimeModel"
            class="native-dashboard__datetime-input"
            type="datetime-local"
          />
        </div>
        <div class="native-dashboard__control">
          <span class="native-dashboard__label">{{ rangeLabel }}</span>
          <a-select v-model:value="rangeMsModel" :options="RANGE_OPTIONS.map((option) => ({ value: option, label: t(`dashboard.native.rangeOptions.${option}`) }))" />
          <span v-if="rangeHint" class="native-dashboard__hint">{{ rangeHint }}</span>
        </div>
        <div class="native-dashboard__control">
          <span class="native-dashboard__label">{{ t('dashboard.native.refreshEvery') }}</span>
          <a-select
            v-model:value="refreshMsModel"
            :disabled="!canAutoRefresh"
            :options="REFRESH_OPTIONS.map((option) => ({ value: option, label: t(`dashboard.native.refreshOptions.${option}`) }))"
          />
        </div>
      </div>
      <div class="native-dashboard__actions">
        <span class="native-dashboard__updated" v-if="lastUpdated">
          {{ t('dashboard.native.lastUpdated', { time: lastUpdated }) }}
        </span>
        <a-button @click="load">
          <template #icon>
            <ReloadOutlined />
          </template>
          {{ t('dashboard.native.refresh') }}
        </a-button>
      </div>
    </div>

    <a-skeleton v-if="loading && !data" active />

    <a-alert
      v-else-if="errorMessage"
      type="warning"
      show-icon
      :message="errorMessage"
    />

    <a-empty v-else-if="!hasAnyData" :description="t('dashboard.native.noData')" />

    <a-collapse
      v-else
      class="native-dashboard__collapse"
      v-model:active-key="activeRows"
    >
      <a-collapse-panel
        v-for="row in data?.rows || []"
        :key="row.title"
        :header="t(`dashboard.native.rows.${row.title}`) === `dashboard.native.rows.${row.title}` ? row.title : t(`dashboard.native.rows.${row.title}`)"
      >
        <div class="native-dashboard__grid">
          <div
            v-for="panel in row.panels"
            :key="panel.id"
            class="native-dashboard__panel-cell"
            :style="{ gridColumn: `${panel.gridPos.x + 1} / span ${Math.max(1, panel.gridPos.w)}` }"
          >
            <NativeDashboardPanelCard
              :panel="panel"
              :range-ms="chartRangeMs"
              :from="data?.from"
              :to="data?.to"
              :translate-text="translateText"
            />
          </div>
        </div>
      </a-collapse-panel>
    </a-collapse>
  </div>
</template>

<style scoped>
.native-dashboard {
  display: grid;
  gap: 16px;
}

.native-dashboard__toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.native-dashboard__controls,
.native-dashboard__actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.native-dashboard__control {
  display: grid;
  gap: 6px;
  min-width: 180px;
}

.native-dashboard__label,
.native-dashboard__updated {
  color: var(--portal-text-soft);
  font-size: 12px;
}

.native-dashboard__hint {
  color: var(--portal-text-soft);
  font-size: 11px;
  line-height: 1.4;
}

.native-dashboard__datetime-input {
  min-height: 32px;
  border-radius: 8px;
  border: 1px solid var(--portal-border);
  padding: 6px 10px;
  color: var(--portal-text);
  background: var(--portal-surface-strong);
}

.native-dashboard__collapse :deep(.ant-collapse-item) {
  border-radius: 18px;
  border: 1px solid var(--portal-border);
  overflow: hidden;
  background: rgba(255, 255, 255, 0.9);
}

.native-dashboard__grid {
  display: grid;
  grid-template-columns: repeat(24, minmax(0, 1fr));
  gap: 14px;
}

.native-dashboard__panel-cell {
  min-width: 0;
}

@media (max-width: 1023px) {
  .native-dashboard__grid {
    grid-template-columns: repeat(12, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .native-dashboard__grid {
    grid-template-columns: 1fr;
  }

  .native-dashboard__panel-cell {
    grid-column: 1 / -1 !important;
  }
}
</style>
