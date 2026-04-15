<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import PortalUnavailableState from '@/components/common/PortalUnavailableState.vue';
import { usePortalAvailability } from '@/composables/usePortalAvailability';
import PortalStatsTrendChart from './portal-stats/PortalStatsTrendChart.vue';
import type {
  PortalDepartmentBillRecord,
  PortalStatsSelectOption,
  PortalUsageEventRecord,
  PortalUsageEventFilterOptions,
  PortalUsageStatRecord,
  PortalUsageTrendPoint,
} from '@/interfaces/portal-stats';
import {
  getPortalDepartmentBills,
  getPortalUsageEventOptions,
  getPortalUsageEvents,
  getPortalUsageStats,
  getPortalUsageTrend,
} from '@/services/portal-stats';
import { formatDateTimeDisplay } from '@/utils/time';

const props = defineProps<{
  from?: number;
  to?: number;
}>();

const { portalUnavailable } = usePortalAvailability();

const activeTab = ref('usage');
const loading = ref(false);
const usageRows = ref<PortalUsageStatRecord[]>([]);
const usageTrendRows = ref<PortalUsageTrendPoint[]>([]);
const usageEventRows = ref<PortalUsageEventRecord[]>([]);
const departmentBillRows = ref<PortalDepartmentBillRecord[]>([]);
const usageEventFilterOptions = ref<PortalUsageEventFilterOptions>({
  consumers: [],
  departments: [],
  apiKeys: [],
  models: [],
  routes: [],
  requestStatuses: [],
  usageStatuses: [],
});
const usageTrendBucket = ref<'auto' | '5m' | 'hour' | 'day'>('auto');

const usageEventsQuery = reactive({
  consumerNames: [] as string[],
  departmentIds: [] as string[],
  apiKeyIds: [] as string[],
  modelIds: [] as string[],
  routeNames: [] as string[],
  requestStatuses: [] as string[],
  usageStatuses: [] as string[],
  includeChildren: true,
  pageNum: 1,
  pageSize: 50,
});

const departmentBillsQuery = reactive({
  departmentIds: [] as string[],
  includeChildren: true,
});

const statsTimeRange = computed(() => ({
  from: props.from,
  to: props.to,
}));

const usageTrendQuery = computed(() => ({
  ...statsTimeRange.value,
  bucket: usageTrendBucket.value === 'auto' ? undefined : usageTrendBucket.value,
}));

const usageEventsParams = computed(() => ({
  ...statsTimeRange.value,
  ...usageEventsQuery,
}));

const departmentBillsParams = computed(() => ({
  ...statsTimeRange.value,
  ...departmentBillsQuery,
}));

const trendRangeMs = computed(() => {
  if (!props.from || !props.to || props.to <= props.from) {
    return undefined;
  }
  return props.to - props.from;
});

const departmentOptions = computed(() => usageEventFilterOptions.value.departments);

async function loadUsage() {
  const [usage, trend] = await Promise.all([
    getPortalUsageStats(statsTimeRange.value).catch(() => []),
    getPortalUsageTrend(usageTrendQuery.value).catch(() => []),
  ]);
  usageRows.value = usage;
  usageTrendRows.value = trend;
}

async function loadUsageEvents() {
  usageEventRows.value = await getPortalUsageEvents(usageEventsParams.value).catch(() => []);
}

async function loadDepartmentBills() {
  departmentBillRows.value = await getPortalDepartmentBills(departmentBillsParams.value).catch(() => []);
}

async function loadUsageEventFilterOptions() {
  usageEventFilterOptions.value = await getPortalUsageEventOptions({
    ...statsTimeRange.value,
    consumerNames: usageEventsQuery.consumerNames,
    departmentIds: usageEventsQuery.departmentIds,
    includeChildren: usageEventsQuery.includeChildren,
    apiKeyIds: usageEventsQuery.apiKeyIds,
    modelIds: usageEventsQuery.modelIds,
    routeNames: usageEventsQuery.routeNames,
    requestStatuses: usageEventsQuery.requestStatuses,
    usageStatuses: usageEventsQuery.usageStatuses,
  }).catch(() => ({
    consumers: [],
    departments: [],
    apiKeys: [],
    models: [],
    routes: [],
    requestStatuses: [],
    usageStatuses: [],
  }));
}

async function loadActiveTab() {
  if (portalUnavailable.value) {
    usageRows.value = [];
    usageTrendRows.value = [];
    usageEventRows.value = [];
    departmentBillRows.value = [];
    usageEventFilterOptions.value = {
      consumers: [],
      departments: [],
      apiKeys: [],
      models: [],
      routes: [],
      requestStatuses: [],
      usageStatuses: [],
    };
    return;
  }
  loading.value = true;
  try {
    if (activeTab.value === 'usage') {
      await loadUsage();
      return;
    }
    if (activeTab.value === 'usage-events') {
      await Promise.all([loadUsageEventFilterOptions(), loadUsageEvents()]);
      return;
    }
    await Promise.all([loadUsageEventFilterOptions(), loadDepartmentBills()]);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void loadActiveTab();
});

watch(activeTab, () => {
  void loadActiveTab();
});

watch(usageTrendBucket, () => {
  if (activeTab.value === 'usage') {
    void loadUsage();
  }
});

watch(
  () => [props.from, props.to],
  () => {
    void loadActiveTab();
  },
);

watch(
  () => ({
    consumerNames: usageEventsQuery.consumerNames.join(','),
    departmentIds: usageEventsQuery.departmentIds.join(','),
    apiKeyIds: usageEventsQuery.apiKeyIds.join(','),
    modelIds: usageEventsQuery.modelIds.join(','),
    routeNames: usageEventsQuery.routeNames.join(','),
    requestStatuses: usageEventsQuery.requestStatuses.join(','),
    usageStatuses: usageEventsQuery.usageStatuses.join(','),
    includeChildren: usageEventsQuery.includeChildren,
  }),
  () => {
    if (activeTab.value === 'usage-events') {
      void loadUsageEventFilterOptions();
    }
  },
);

const usageSummary = computed(() => {
  return usageRows.value.reduce(
    (summary, item) => {
      summary.requests += item.requestCount || 0;
      summary.tokens += item.totalTokens || 0;
      return summary;
    },
    { requests: 0, tokens: 0 },
  );
});

function downloadCsv(filename: string, rows: Record<string, unknown>[]) {
  if (!rows.length) {
    return;
  }
  const headers = Object.keys(rows[0]);
  const content = [
    headers.join(','),
    ...rows.map((row) =>
      headers
        .map((header) => {
          const value = row[header] ?? '';
          const normalized = String(value).replace(/"/g, '""');
          return `"${normalized}"`;
        })
        .join(','),
    ),
  ].join('\n');

  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' });
  const link = document.createElement('a');
  link.href = URL.createObjectURL(blob);
  link.download = filename;
  link.click();
  URL.revokeObjectURL(link.href);
}

function exportUsage() {
  downloadCsv(
    `portal-usage-${usageTrendBucket.value}.csv`,
    usageRows.value.map((item) => ({
      consumerName: item.consumerName,
      modelName: item.modelName,
      requestCount: item.requestCount,
      totalTokens: item.totalTokens,
      inputTokens: item.inputTokens,
      outputTokens: item.outputTokens,
    })),
  );
}

function exportUsageEvents() {
  downloadCsv(
    'portal-usage-events.csv',
    usageEventRows.value.map((item) => ({
      occurredAt: item.occurredAt,
      consumerName: item.consumerName,
      departmentPath: item.departmentPath,
      apiKeyId: item.apiKeyId,
      modelId: item.modelId,
      routeName: item.routeName,
      requestPath: item.requestPath,
      requestStatus: item.requestStatus,
      usageStatus: item.usageStatus,
      httpStatus: item.httpStatus,
      errorCode: item.errorCode,
      serviceDurationMs: item.serviceDurationMs,
      requestCount: item.requestCount,
      totalTokens: item.totalTokens,
      costMicroYuan: item.costMicroYuan,
    })),
  );
}

function exportDepartmentBills() {
  downloadCsv(
    'portal-department-bills.csv',
    departmentBillRows.value.map((item) => ({
      departmentId: item.departmentId,
      departmentName: item.departmentName,
      departmentPath: item.departmentPath,
      requestCount: item.requestCount,
      totalTokens: item.totalTokens,
      totalCost: item.totalCost,
      activeConsumers: item.activeConsumers,
    })),
  );
}

function filterSelectOption(input: string, option?: PortalStatsSelectOption) {
  const normalized = input.trim().toLowerCase();
  if (!normalized) {
    return true;
  }
  return `${option?.label || ''} ${option?.value || ''}`.toLowerCase().includes(normalized);
}
</script>

<template>
  <div class="portal-stats-panel">
    <PortalUnavailableState v-if="portalUnavailable" />

    <template v-else>
      <div class="portal-stats-panel__actions">
        <a-button @click="loadActiveTab">刷新</a-button>
      </div>

      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="usage" tab="用量概览">
          <div class="portal-stats-panel__summary">
            <div class="portal-stats-panel__summary-card">
              <span>总请求数</span>
              <strong>{{ usageSummary.requests }}</strong>
            </div>
            <div class="portal-stats-panel__summary-card">
              <span>总 Token</span>
              <strong>{{ usageSummary.tokens }}</strong>
            </div>
            <div class="portal-stats-panel__summary-actions">
              <a-radio-group v-model:value="usageTrendBucket" size="small">
                <a-radio-button value="auto">自动</a-radio-button>
                <a-radio-button value="5m">5 分钟</a-radio-button>
                <a-radio-button value="day">按天</a-radio-button>
                <a-radio-button value="hour">按小时</a-radio-button>
              </a-radio-group>
              <a-button @click="exportUsage">导出 CSV</a-button>
            </div>
          </div>

          <PortalStatsTrendChart :points="usageTrendRows" :range-ms="trendRangeMs" />

          <a-table :data-source="usageRows" :loading="loading" :row-key="(record) => `${record.consumerName}-${record.modelName}`" :scroll="{ x: 1180 }" size="small">
            <a-table-column key="consumerName" data-index="consumerName" title="用户" width="180" />
            <a-table-column key="modelName" data-index="modelName" title="模型" width="180" />
            <a-table-column key="requestCount" data-index="requestCount" title="请求数" width="120" />
            <a-table-column key="inputTokens" data-index="inputTokens" title="输入 Token" width="120" />
            <a-table-column key="outputTokens" data-index="outputTokens" title="输出 Token" width="120" />
            <a-table-column key="totalTokens" data-index="totalTokens" title="总 Token" width="120" />
            <a-table-column key="cacheReadInputTokens" data-index="cacheReadInputTokens" title="缓存读取 Token" width="120" />
            <a-table-column key="inputImageCount" data-index="inputImageCount" title="输入图片数" width="120" />
            <a-table-column key="outputImageCount" data-index="outputImageCount" title="输出图片数" width="120" />
          </a-table>
        </a-tab-pane>

        <a-tab-pane key="usage-events" tab="使用记录">
          <div class="portal-stats-panel__toolbar">
            <a-form layout="inline" class="portal-stats-panel__filters">
              <a-form-item label="用户">
                <a-select
                  v-model:value="usageEventsQuery.consumerNames"
                  mode="multiple"
                  allow-clear
                  show-search
                  :filter-option="filterSelectOption"
                  :options="usageEventFilterOptions.consumers"
                  style="width: 220px"
                />
              </a-form-item>
              <a-form-item label="部门">
                <a-select
                  v-model:value="usageEventsQuery.departmentIds"
                  mode="multiple"
                  allow-clear
                  show-search
                  :filter-option="filterSelectOption"
                  :options="usageEventFilterOptions.departments"
                  style="width: 220px"
                />
              </a-form-item>
              <a-form-item label="API Key">
                <a-select
                  v-model:value="usageEventsQuery.apiKeyIds"
                  mode="multiple"
                  allow-clear
                  show-search
                  :filter-option="filterSelectOption"
                  :options="usageEventFilterOptions.apiKeys"
                  style="width: 220px"
                />
              </a-form-item>
              <a-form-item label="模型">
                <a-select
                  v-model:value="usageEventsQuery.modelIds"
                  mode="multiple"
                  allow-clear
                  show-search
                  :filter-option="filterSelectOption"
                  :options="usageEventFilterOptions.models"
                  style="width: 220px"
                />
              </a-form-item>
              <a-form-item label="路由">
                <a-select
                  v-model:value="usageEventsQuery.routeNames"
                  mode="multiple"
                  allow-clear
                  show-search
                  :filter-option="filterSelectOption"
                  :options="usageEventFilterOptions.routes"
                  style="width: 220px"
                />
              </a-form-item>
              <a-form-item label="请求状态">
                <a-select
                  v-model:value="usageEventsQuery.requestStatuses"
                  mode="multiple"
                  allow-clear
                  show-search
                  :filter-option="filterSelectOption"
                  :options="usageEventFilterOptions.requestStatuses"
                  style="width: 180px"
                />
              </a-form-item>
              <a-form-item label="计费状态">
                <a-select
                  v-model:value="usageEventsQuery.usageStatuses"
                  mode="multiple"
                  allow-clear
                  show-search
                  :filter-option="filterSelectOption"
                  :options="usageEventFilterOptions.usageStatuses"
                  style="width: 180px"
                />
              </a-form-item>
              <a-form-item label="包含子部门"><a-switch v-model:checked="usageEventsQuery.includeChildren" /></a-form-item>
              <a-form-item><a-button type="primary" @click="loadActiveTab">查询</a-button></a-form-item>
            </a-form>
            <a-button @click="exportUsageEvents">导出 CSV</a-button>
          </div>

          <a-table :data-source="usageEventRows" :loading="loading" row-key="eventId" :scroll="{ x: 1800 }" size="small">
            <a-table-column key="occurredAt" title="发生时间" width="180">
              <template #default="{ record }">{{ formatDateTimeDisplay(record.occurredAt) }}</template>
            </a-table-column>
            <a-table-column key="consumerName" data-index="consumerName" title="用户" width="160" />
            <a-table-column key="departmentPath" data-index="departmentPath" title="部门路径" width="220" />
            <a-table-column key="apiKeyId" data-index="apiKeyId" title="API Key" width="180" />
            <a-table-column key="modelId" data-index="modelId" title="模型" width="180" />
            <a-table-column key="routeName" data-index="routeName" title="路由" width="180" />
            <a-table-column key="requestStatus" data-index="requestStatus" title="请求状态" width="140" />
            <a-table-column key="usageStatus" data-index="usageStatus" title="计费状态" width="140" />
            <a-table-column key="requestPath" data-index="requestPath" title="请求路径" width="220" />
            <a-table-column key="httpStatus" data-index="httpStatus" title="HTTP 状态码" width="120" />
            <a-table-column key="errorCode" data-index="errorCode" title="错误码" width="180" />
            <a-table-column key="serviceDurationMs" data-index="serviceDurationMs" title="服务时长(ms)" width="140" />
            <a-table-column key="requestCount" data-index="requestCount" title="请求数" width="110" />
            <a-table-column key="totalTokens" data-index="totalTokens" title="总 Token" width="120" />
            <a-table-column key="costMicroYuan" data-index="costMicroYuan" title="费用(μ¥)" width="120" />
          </a-table>
        </a-tab-pane>

        <a-tab-pane key="department-bills" tab="部门账单">
          <div class="portal-stats-panel__toolbar">
            <a-form layout="inline" class="portal-stats-panel__filters">
              <a-form-item label="部门">
                <a-select
                  v-model:value="departmentBillsQuery.departmentIds"
                  mode="multiple"
                  allow-clear
                  show-search
                  :filter-option="filterSelectOption"
                  :options="departmentOptions"
                  style="width: 260px"
                />
              </a-form-item>
              <a-form-item label="包含子部门"><a-switch v-model:checked="departmentBillsQuery.includeChildren" /></a-form-item>
              <a-form-item><a-button type="primary" @click="loadActiveTab">查询</a-button></a-form-item>
            </a-form>
            <a-button @click="exportDepartmentBills">导出 CSV</a-button>
          </div>

          <a-table :data-source="departmentBillRows" :loading="loading" row-key="departmentId" :scroll="{ x: 980 }" size="small">
            <a-table-column key="departmentId" data-index="departmentId" title="部门 ID" width="180" />
            <a-table-column key="departmentName" data-index="departmentName" title="部门" width="180" />
            <a-table-column key="departmentPath" data-index="departmentPath" title="部门路径" width="240" />
            <a-table-column key="requestCount" data-index="requestCount" title="请求数" width="120" />
            <a-table-column key="totalTokens" data-index="totalTokens" title="总 Token" width="140" />
            <a-table-column key="totalCost" data-index="totalCost" title="总费用" width="120" />
            <a-table-column key="activeConsumers" data-index="activeConsumers" title="活跃用户数" width="140" />
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </template>
  </div>
</template>

<style scoped>
.portal-stats-panel {
  display: grid;
  gap: 16px;
}

.portal-stats-panel__actions {
  display: flex;
  justify-content: flex-end;
}

.portal-stats-panel__summary {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 16px;
}

.portal-stats-panel__summary-card {
  min-width: 160px;
  padding: 14px 16px;
  border-radius: 16px;
  background: linear-gradient(160deg, #edf5ff 0%, #ffffff 100%);
  border: 1px solid #dce4f0;
  display: grid;
  gap: 4px;
}

.portal-stats-panel__summary-card span {
  color: #5f6f85;
  font-size: 12px;
}

.portal-stats-panel__summary-card strong {
  font-size: 24px;
  color: #10233c;
}

.portal-stats-panel__summary-actions {
  margin-left: auto;
  display: flex;
  gap: 12px;
  align-items: center;
}

.portal-stats-panel__toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: space-between;
  margin-bottom: 16px;
}

.portal-stats-panel__filters {
  gap: 8px 0;
}
</style>
