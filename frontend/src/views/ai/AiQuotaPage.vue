<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, shallowRef, watch } from 'vue';
import type { AiQuotaConsumerQuota, AiQuotaRouteSummary, AiQuotaScheduleRule, AiQuotaUserPolicy } from '@/interfaces/ai-quota';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import { showConfirm, showSuccess } from '@/lib/feedback';
import {
  deltaAiQuota,
  deleteAiQuotaScheduleRule,
  getAiQuotaConsumers,
  getAiQuotaRoutes,
  getAiQuotaScheduleRules,
  getAiQuotaUserPolicy,
  refreshAiQuota,
  saveAiQuotaScheduleRule,
  saveAiQuotaUserPolicy,
} from '@/services/ai-quota';
import {
  BUILTIN_ADMIN_CONSUMER,
  buildPolicyPayload,
  buildSchedulePayload,
  toDisplayAmount,
  toFormAmount,
  toStoredAmount,
  type AiQuotaPolicyFormValues,
  type AiQuotaScheduleFormValues,
} from '@/features/ai-quota/ai-quota-utils';
import { useI18n } from 'vue-i18n';

const AiQuotaPolicyDrawer = defineAsyncComponent(() => import('@/features/ai-quota/AiQuotaPolicyDrawer.vue'));
const AiQuotaScheduleDrawer = defineAsyncComponent(() => import('@/features/ai-quota/AiQuotaScheduleDrawer.vue'));

const { t } = useI18n();

const routes = shallowRef<AiQuotaRouteSummary[]>([]);
const consumers = shallowRef<AiQuotaConsumerQuota[]>([]);
const selectedRouteName = shallowRef('');
const search = shallowRef('');
const loadingRoutes = shallowRef(false);
const loadingConsumers = shallowRef(false);

const quotaModalOpen = shallowRef(false);
const quotaModalAction = shallowRef<'refresh' | 'delta'>('refresh');
const quotaModalValue = shallowRef(0);
const quotaSubmitting = shallowRef(false);
const currentConsumer = shallowRef<AiQuotaConsumerQuota | null>(null);

const policyOpen = shallowRef(false);
const policyLoading = shallowRef(false);
const policySubmitting = shallowRef(false);
const policyConsumer = shallowRef<AiQuotaConsumerQuota | null>(null);
const policyInitialValues = shallowRef<Partial<AiQuotaUserPolicy>>({});

const scheduleOpen = shallowRef(false);
const scheduleLoading = shallowRef(false);
const scheduleSaving = shallowRef(false);
const scheduleConsumer = shallowRef<AiQuotaConsumerQuota | null>(null);
const scheduleRules = shallowRef<AiQuotaScheduleRule[]>([]);
const editingScheduleRule = shallowRef<AiQuotaScheduleRule | null>(null);
const scheduleResetCounter = shallowRef(0);

const activeRoute = computed(() => routes.value.find((item) => item.routeName === selectedRouteName.value) || null);
const filteredConsumers = computed(() => consumers.value.filter((item) => {
  if (item.consumerName === BUILTIN_ADMIN_CONSUMER) {
    return false;
  }
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return item.consumerName.toLowerCase().includes(keyword);
}));

function formatRouteDomainDisplay(route: Pick<AiQuotaRouteSummary, 'domains' | 'path'>) {
  const domains = (route.domains || []).map((item) => String(item || '').trim()).filter(Boolean);
  if (domains.length > 0) {
    return domains.join(', ');
  }
  const internalPath = String(route.path || '').trim();
  if (internalPath) {
    return `内部路由 · ${internalPath}`;
  }
  return '内部路由';
}

const summaryItems = computed(() => {
  if (!activeRoute.value) {
    return [];
  }
  return [
    { label: t('aiQuota.summary.route'), value: activeRoute.value.routeName },
    { label: t('aiQuota.summary.path'), value: activeRoute.value.path || '-' },
    { label: t('aiQuota.summary.domains'), value: formatRouteDomainDisplay(activeRoute.value) },
    { label: t('aiQuota.summary.balanceKeyPrefix'), value: activeRoute.value.redisKeyPrefix || '-' },
    { label: t('aiQuota.summary.adminConsumer'), value: activeRoute.value.adminConsumer || '-' },
    { label: t('aiQuota.summary.adminPath'), value: activeRoute.value.adminPath || '-' },
    { label: t('aiQuota.summary.quotaUnit'), value: 'RMB' },
    { label: t('aiQuota.summary.scheduleRuleCount'), value: String(activeRoute.value.scheduleRuleCount ?? 0) },
  ];
});

async function loadRoutes() {
  loadingRoutes.value = true;
  try {
    const result = await getAiQuotaRoutes().catch(() => []);
    routes.value = result.filter((item) => item.quotaUnit === 'amount');
    if (!routes.value.length) {
      selectedRouteName.value = '';
      consumers.value = [];
      return;
    }
    if (!selectedRouteName.value || !routes.value.some((item) => item.routeName === selectedRouteName.value)) {
      selectedRouteName.value = routes.value[0].routeName;
    }
  } finally {
    loadingRoutes.value = false;
  }
}

async function loadConsumers() {
  if (!selectedRouteName.value) {
    consumers.value = [];
    return;
  }
  loadingConsumers.value = true;
  try {
    consumers.value = await getAiQuotaConsumers(selectedRouteName.value).catch(() => []);
  } finally {
    loadingConsumers.value = false;
  }
}

async function refreshAll() {
  await loadRoutes();
  await loadConsumers();
}

function openQuotaModal(record: AiQuotaConsumerQuota, action: 'refresh' | 'delta') {
  currentConsumer.value = record;
  quotaModalAction.value = action;
  quotaModalValue.value = action === 'refresh' ? toFormAmount(record.quota) : 0;
  quotaModalOpen.value = true;
}

function closeQuotaModal() {
  quotaModalOpen.value = false;
  currentConsumer.value = null;
  quotaModalValue.value = 0;
}

async function submitQuotaModal() {
  if (!selectedRouteName.value || !currentConsumer.value) {
    return;
  }
  quotaSubmitting.value = true;
  try {
    if (quotaModalAction.value === 'refresh') {
      await refreshAiQuota(selectedRouteName.value, currentConsumer.value.consumerName, toStoredAmount(quotaModalValue.value));
      showSuccess(t('aiQuota.messages.refreshBalanceSuccess'));
    } else {
      await deltaAiQuota(selectedRouteName.value, currentConsumer.value.consumerName, toStoredAmount(quotaModalValue.value));
      showSuccess(t('aiQuota.messages.deltaBalanceSuccess'));
    }
    closeQuotaModal();
    await loadConsumers();
  } finally {
    quotaSubmitting.value = false;
  }
}

async function openPolicyDrawer(record: AiQuotaConsumerQuota) {
  if (!selectedRouteName.value) {
    return;
  }
  policyConsumer.value = record;
  policyOpen.value = true;
  await reloadPolicy();
}

async function reloadPolicy() {
  if (!selectedRouteName.value || !policyConsumer.value) {
    return;
  }
  policyLoading.value = true;
  try {
    policyInitialValues.value = await getAiQuotaUserPolicy(selectedRouteName.value, policyConsumer.value.consumerName).catch(() => ({}));
  } finally {
    policyLoading.value = false;
  }
}

async function submitPolicy(values: AiQuotaPolicyFormValues) {
  if (!selectedRouteName.value || !policyConsumer.value) {
    return;
  }
  policySubmitting.value = true;
  try {
    const saved = await saveAiQuotaUserPolicy(
      selectedRouteName.value,
      policyConsumer.value.consumerName,
      buildPolicyPayload(values),
    );
    policyInitialValues.value = saved;
    showSuccess(t('aiQuota.messages.policySaved'));
  } finally {
    policySubmitting.value = false;
  }
}

function closePolicyDrawer() {
  policyOpen.value = false;
  policyConsumer.value = null;
  policyInitialValues.value = {};
}

async function openScheduleDrawer(record: AiQuotaConsumerQuota) {
  if (!selectedRouteName.value) {
    return;
  }
  scheduleConsumer.value = record;
  editingScheduleRule.value = null;
  scheduleOpen.value = true;
  scheduleResetCounter.value += 1;
  await reloadSchedules();
}

async function reloadSchedules() {
  if (!selectedRouteName.value || !scheduleConsumer.value) {
    return;
  }
  scheduleLoading.value = true;
  try {
    scheduleRules.value = await getAiQuotaScheduleRules(selectedRouteName.value, scheduleConsumer.value.consumerName).catch(() => []);
  } finally {
    scheduleLoading.value = false;
  }
}

async function submitSchedule(values: AiQuotaScheduleFormValues) {
  if (!selectedRouteName.value || !scheduleConsumer.value) {
    return;
  }
  scheduleSaving.value = true;
  try {
    await saveAiQuotaScheduleRule(selectedRouteName.value, {
      id: editingScheduleRule.value?.id,
      consumerName: scheduleConsumer.value.consumerName,
      ...buildSchedulePayload(values),
    });
    editingScheduleRule.value = null;
    scheduleResetCounter.value += 1;
    showSuccess(t('aiQuota.messages.scheduleSaved'));
    await reloadSchedules();
    await loadRoutes();
  } finally {
    scheduleSaving.value = false;
  }
}

function editSchedule(rule: AiQuotaScheduleRule) {
  editingScheduleRule.value = rule;
  scheduleResetCounter.value += 1;
}

function resetSchedule() {
  editingScheduleRule.value = null;
  scheduleResetCounter.value += 1;
}

function removeSchedule(rule: AiQuotaScheduleRule) {
  if (!selectedRouteName.value) {
    return;
  }
  showConfirm({
    title: t('misc.delete'),
    content: `${t('misc.delete')} ${rule.id} ?`,
    okText: t('misc.confirm'),
    cancelText: t('misc.cancel'),
    async onOk() {
      await deleteAiQuotaScheduleRule(selectedRouteName.value, rule.id);
      if (editingScheduleRule.value?.id === rule.id) {
        resetSchedule();
      }
      showSuccess(t('aiQuota.messages.scheduleDeleted'));
      await reloadSchedules();
      await loadRoutes();
    },
  });
}

function closeScheduleDrawer() {
  scheduleOpen.value = false;
  scheduleConsumer.value = null;
  editingScheduleRule.value = null;
  scheduleRules.value = [];
}

watch(selectedRouteName, () => {
  if (selectedRouteName.value) {
    void loadConsumers();
  }
});

onMounted(() => {
  void loadRoutes();
});
</script>

<template>
  <div class="ai-quota-page">
    <PageSection :title="t('menu.aiQuotaManagement')">
      <ListToolbar
        v-model:search="search"
        :search-placeholder="t('aiQuota.searchPlaceholder')"
        @refresh="refreshAll"
      >
        <template #left>
          <a-select
            v-model:value="selectedRouteName"
            class="ai-quota-page__route-select"
            :placeholder="t('aiQuota.route')"
            :loading="loadingRoutes"
          >
            <a-select-option v-for="item in routes" :key="item.routeName" :value="item.routeName">
              {{ item.routeName }}
            </a-select-option>
          </a-select>
        </template>
      </ListToolbar>

      <a-empty v-if="!routes.length && !loadingRoutes" :description="t('aiQuota.amountOnlyEmpty')" />

      <template v-else>
        <div v-if="summaryItems.length" class="ai-quota-page__summary">
          <article v-for="item in summaryItems" :key="item.label" class="ai-quota-page__summary-card">
            <span>{{ item.label }}</span>
            <strong :title="item.value">{{ item.value }}</strong>
          </article>
        </div>

        <a-table
          row-key="consumerName"
          size="middle"
          :loading="loadingConsumers"
          :scroll="{ x: 980 }"
          :data-source="filteredConsumers"
          :pagination="{ showSizeChanger: true, showTotal: (total) => `${t('misc.total')} ${total}` }"
        >
          <a-table-column key="consumerName" data-index="consumerName" :title="t('aiQuota.columns.consumer')" />
          <a-table-column key="quota" :title="t('aiQuota.columns.balance')">
            <template #default="{ record }">{{ toDisplayAmount(record.quota) }}</template>
          </a-table-column>
          <a-table-column key="actions" :title="t('aiQuota.columns.actions')" width="320">
            <template #default="{ record }">
              <a-button type="link" size="small" @click="openQuotaModal(record, 'refresh')">
                {{ t('aiQuota.actions.refreshBalance') }}
              </a-button>
              <a-button type="link" size="small" @click="openQuotaModal(record, 'delta')">
                {{ t('aiQuota.actions.deltaBalance') }}
              </a-button>
              <a-button type="link" size="small" @click="openPolicyDrawer(record)">
                {{ t('aiQuota.actions.policy') }}
              </a-button>
              <a-button type="link" size="small" @click="openScheduleDrawer(record)">
                {{ t('aiQuota.actions.schedule') }}
              </a-button>
            </template>
          </a-table-column>
        </a-table>
      </template>
    </PageSection>

    <a-modal
      :open="quotaModalOpen"
      :confirm-loading="quotaSubmitting"
      :title="quotaModalAction === 'refresh' ? t('aiQuota.modals.refreshBalanceTitle') : t('aiQuota.modals.deltaBalanceTitle')"
      destroy-on-close
      @cancel="closeQuotaModal"
      @ok="submitQuotaModal"
    >
      <a-form layout="vertical">
        <a-form-item :label="t('aiQuota.columns.consumer')">
          <span>{{ currentConsumer?.consumerName || '-' }}</span>
        </a-form-item>
        <a-form-item
          :label="`${quotaModalAction === 'refresh' ? t('aiQuota.modals.refreshBalanceValue') : t('aiQuota.modals.deltaBalanceValue')} (RMB)`"
          :extra="t('aiQuota.form.amountValueHelp')"
        >
          <a-input-number v-model:value="quotaModalValue" :precision="6" :step="0.01" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <AiQuotaPolicyDrawer
      :open="policyOpen"
      :consumer-name="policyConsumer?.consumerName"
      :initial-values="policyInitialValues"
      :loading="policyLoading"
      :submitting="policySubmitting"
      @close="closePolicyDrawer"
      @reload="reloadPolicy"
      @submit="submitPolicy"
    />

    <AiQuotaScheduleDrawer
      :open="scheduleOpen"
      :consumer-name="scheduleConsumer?.consumerName"
      :current-amount="scheduleConsumer?.quota || 0"
      :rules="scheduleRules"
      :loading="scheduleLoading"
      :saving="scheduleSaving"
      :editing-rule="editingScheduleRule"
      :reset-counter="scheduleResetCounter"
      @close="closeScheduleDrawer"
      @edit="editSchedule"
      @remove="removeSchedule"
      @reset="resetSchedule"
      @submit="submitSchedule"
    />
  </div>
</template>

<style scoped>
.ai-quota-page {
  display: grid;
}

.ai-quota-page__route-select {
  width: 320px;
}

.ai-quota-page__summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.ai-quota-page__summary-card {
  min-width: 0;
  padding: 14px 16px;
  border: 1px solid var(--portal-border);
  border-radius: 14px;
  background: var(--portal-surface-soft);
}

.ai-quota-page__summary-card span {
  display: block;
  margin-bottom: 8px;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.ai-quota-page__summary-card strong {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1023px) {
  .ai-quota-page__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .ai-quota-page__route-select {
    width: 100%;
  }

  .ai-quota-page__summary {
    grid-template-columns: 1fr;
  }
}
</style>
