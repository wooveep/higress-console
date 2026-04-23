<script setup lang="ts">
import { computed, defineAsyncComponent, shallowRef, watch } from 'vue';
import { useRoute } from 'vue-router';
import { DashboardType, type DashboardInfo } from '@/interfaces/dashboard';
import PageSection from '@/components/common/PageSection.vue';
import { getDashboardInfo } from '@/services/dashboard';
import { useI18n } from 'vue-i18n';
import { usePortalAvailability } from '@/composables/usePortalAvailability';
import {
  createDashboardTimeRangeState,
  resolveDashboardTimeWindow,
  type DashboardTimeRangeState,
} from '@/features/dashboard/dashboard-native';

const NativeDashboardView = defineAsyncComponent(() => import('@/features/dashboard/NativeDashboardView.vue'));
const PortalStatsPanel = defineAsyncComponent(() => import('@/features/dashboard/PortalStatsPanel.vue'));

const route = useRoute();
const { t } = useI18n();
const { portalUnavailable } = usePortalAvailability();

const loading = shallowRef(false);
const dashboardInfo = shallowRef<DashboardInfo | null>(null);
const errorMessage = shallowRef('');
const timeRange = shallowRef<DashboardTimeRangeState>(createDashboardTimeRangeState());
const resolvedWindow = shallowRef(resolveDashboardTimeWindow(timeRange.value));

const dashboardType = computed(() => route.meta.dashboardType === 'AI' ? DashboardType.AI : DashboardType.MAIN);
const supportsNative = computed(() => Boolean(dashboardInfo.value?.builtIn));

async function load() {
  loading.value = true;
  errorMessage.value = '';
  try {
    dashboardInfo.value = await getDashboardInfo(dashboardType.value).catch(() => null);
    if (!dashboardInfo.value) {
      errorMessage.value = t('dashboard.loadFailed');
    }
  } finally {
    loading.value = false;
  }
}

watch(dashboardType, () => {
  void load();
}, { immediate: true });

function handleTimeRangeChange(next: DashboardTimeRangeState) {
  timeRange.value = next;
}

function handleWindowChange(next: { from: number; to: number; valid: boolean }) {
  resolvedWindow.value = next;
}
</script>

<template>
  <div class="dashboard-page">
    <PageSection title="监控视图">
      <a-skeleton v-if="loading" active />

      <a-alert
        v-else-if="errorMessage"
        type="warning"
        show-icon
        :message="errorMessage"
      />

      <NativeDashboardView
        v-else-if="supportsNative"
        :type="dashboardType"
        :time-range="timeRange"
        @update:time-range="handleTimeRangeChange"
        @window-change="handleWindowChange"
      />

      <div v-else class="dashboard-page__empty">
        <a-empty :description="t('dashboard.noBuiltInDashboard')" />
      </div>
    </PageSection>

    <PageSection v-if="dashboardType === DashboardType.AI" title="AI 用量统计">
      <PortalStatsPanel
        :key="String(portalUnavailable)"
        :from="resolvedWindow.valid ? resolvedWindow.from : undefined"
        :to="resolvedWindow.valid ? resolvedWindow.to : undefined"
      />
    </PageSection>
  </div>
</template>

<style scoped>
.dashboard-page {
  display: grid;
  gap: 18px;
}

.dashboard-page__empty {
  display: grid;
  justify-items: center;
  gap: 12px;
  padding: 24px 0 8px;
}
</style>
