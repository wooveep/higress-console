<script setup lang="ts">
import { computed } from 'vue';
import NativeDashboardLineChart from '@/features/dashboard/NativeDashboardLineChart.vue';
import { formatTableValue, formatValue, panelHasData } from '@/features/dashboard/dashboard-native';
import type { NativeDashboardPanel } from '@/interfaces/dashboard';
import { formatDateTimeDisplay } from '@/utils/time';

const props = defineProps<{
  panel: NativeDashboardPanel;
  rangeMs: number;
  translateText: (group: 'titles' | 'series' | 'columns' | 'values', value?: string) => string;
}>();

const cardHeight = computed(() => Math.max(props.panel.type === 'stat' ? 184 : 252, props.panel.gridPos.h * 34));
const tableColumns = computed(() => (props.panel.table?.columns || []).map((column) => ({
  title: props.translateText('columns', column.title || column.key),
  dataIndex: column.key,
  key: column.key,
})));
const hasContent = computed(() => panelHasData(props.panel));

function formatBodyCellValue(columnKey: string, value: string | number | null) {
  if (typeof value === 'string' && columnKey === 'requestStatus') {
    return props.translateText('values', value);
  }
  if (typeof value === 'string' && /At$/.test(columnKey)) {
    return formatDateTimeDisplay(value);
  }
  if (typeof value === 'number' && /(duration|latency|rt)/i.test(columnKey)) {
    return formatValue(value, 'ms');
  }
  return formatTableValue(value);
}
</script>

<template>
  <a-card
    class="native-dashboard-panel"
    :title="translateText('titles', panel.title)"
    :bordered="false"
    :body-style="{ height: `${cardHeight - 58}px` }"
  >
    <div class="native-dashboard-panel__body" :style="{ minHeight: `${cardHeight - 58}px` }">
      <a-alert
        v-if="panel.error"
        class="native-dashboard-panel__alert"
        type="warning"
        show-icon
        :message="panel.error"
      />

      <div v-if="panel.type === 'stat'" class="native-dashboard-panel__stat">
        <template v-if="hasContent">
          <a-statistic :value="formatValue(panel.stat?.value, panel.unit)" />
        </template>
        <a-empty v-else />
      </div>

      <div v-else-if="panel.type === 'timeseries'" class="native-dashboard-panel__chart">
        <NativeDashboardLineChart v-if="hasContent" :series="panel.series || []" :range-ms="rangeMs" :unit="panel.unit" />
        <a-empty v-else />
      </div>

      <div v-else class="native-dashboard-panel__table">
        <a-table
          v-if="hasContent"
          size="small"
          :pagination="false"
          :row-key="(_, index) => `${panel.id}-${index}`"
          :scroll="{ x: 'max-content' }"
          :columns="tableColumns"
          :data-source="panel.table?.rows || []"
        >
          <template #bodyCell="{ column, text }">
            <span :data-column="column.key">{{ formatBodyCellValue(String(column.key || ''), text as string | number | null) }}</span>
          </template>
        </a-table>
        <a-empty v-else />
      </div>
    </div>
  </a-card>
</template>

<style scoped>
.native-dashboard-panel {
  height: 100%;
  border: 1px solid var(--portal-border);
  border-radius: 18px;
  box-shadow: none;
  background: var(--portal-surface-strong);
}

.native-dashboard-panel__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.native-dashboard-panel__alert {
  margin-bottom: 2px;
}

.native-dashboard-panel__stat,
.native-dashboard-panel__chart,
.native-dashboard-panel__table {
  flex: 1;
  min-height: 0;
}

.native-dashboard-panel__stat {
  display: flex;
  align-items: center;
  justify-content: center;
}

.native-dashboard-panel__table :deep(.ant-table-wrapper),
.native-dashboard-panel__table :deep(.ant-spin-nested-loading),
.native-dashboard-panel__table :deep(.ant-spin-container) {
  height: 100%;
}
</style>
