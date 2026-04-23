<script setup lang="ts">
import { nextTick, reactive, ref, watch } from 'vue';
import type { AiQuotaScheduleAction, AiQuotaScheduleRule } from '@/interfaces/ai-quota';
import {
  createDefaultScheduleValues,
  createScheduleValuesFromRule,
  formatScheduleTime,
  toDisplayAmount,
  type AiQuotaScheduleFormValues,
} from '@/features/ai-quota/ai-quota-utils';
import { useI18n } from 'vue-i18n';

const props = defineProps<{
  open: boolean;
  consumerName?: string;
  currentAmount: number;
  rules: AiQuotaScheduleRule[];
  loading?: boolean;
  saving?: boolean;
  editingRule?: AiQuotaScheduleRule | null;
  resetCounter?: number;
}>();

const emit = defineEmits<{
  close: [];
  submit: [values: AiQuotaScheduleFormValues];
  edit: [rule: AiQuotaScheduleRule];
  remove: [rule: AiQuotaScheduleRule];
  reset: [];
}>();

const { t } = useI18n();
const formRef = ref<any>();
const formState = reactive<AiQuotaScheduleFormValues>(createDefaultScheduleValues(props.currentAmount));

watch(() => [props.open, props.editingRule?.id, props.currentAmount, props.resetCounter] as const, async ([open]) => {
  if (!open) {
    formRef.value?.resetFields();
    return;
  }
  await nextTick();
  const nextValues = props.editingRule
    ? createScheduleValuesFromRule(props.editingRule)
    : createDefaultScheduleValues(props.currentAmount);
  Object.assign(formState, nextValues);
}, { immediate: true });

async function handleSubmit() {
  await formRef.value?.validate();
  emit('submit', { ...formState });
}

function actionLabel(action: AiQuotaScheduleAction) {
  return action === 'REFRESH'
    ? t('aiQuota.schedule.actions.refreshBalance')
    : t('aiQuota.schedule.actions.deltaBalance');
}
</script>

<template>
  <a-drawer
    :open="open"
    width="760"
    :title="t('aiQuota.schedule.balanceTitle')"
    destroy-on-close
    @close="emit('close')"
  >
    <div class="ai-quota-schedule-drawer__consumer">
      <span>{{ t('aiQuota.columns.consumer') }}</span>
      <strong>{{ consumerName || '-' }}</strong>
    </div>

    <a-form ref="formRef" :model="formState" layout="vertical">
      <a-form-item
        name="action"
        :label="t('aiQuota.schedule.form.action')"
        :rules="[{ required: true, message: t('aiQuota.validation.scheduleActionRequired') }]"
      >
        <a-select
          v-model:value="formState.action"
          :options="[
            { label: t('aiQuota.schedule.actions.refreshBalance'), value: 'REFRESH' },
            { label: t('aiQuota.schedule.actions.deltaBalance'), value: 'DELTA' },
          ]"
        />
      </a-form-item>

      <a-form-item
        name="cron"
        :label="t('aiQuota.schedule.form.cron')"
        :extra="t('aiQuota.schedule.form.cronHelp')"
        :rules="[{ required: true, message: t('aiQuota.validation.scheduleCronRequired') }]"
      >
        <a-input v-model:value="formState.cron" placeholder="0 0 0 * * *" />
      </a-form-item>

      <a-form-item
        name="value"
        :label="`${t('aiQuota.schedule.form.value')} (RMB)`"
        :extra="t('aiQuota.schedule.form.amountValueHelp')"
        :rules="[{ required: true, message: t('aiQuota.validation.scheduleValueRequired') }]"
      >
        <a-input-number v-model:value="formState.value" :precision="6" :step="0.01" style="width: 100%" />
      </a-form-item>

      <a-form-item name="enabled" :label="t('aiQuota.schedule.form.enabled')" value-prop-name="checked">
        <a-switch v-model:checked="formState.enabled" />
      </a-form-item>
    </a-form>

    <div class="ai-quota-schedule-drawer__actions">
      <div class="ai-quota-schedule-drawer__submit">
        <a-button @click="emit('reset')">{{ t('misc.reset') }}</a-button>
        <a-button type="primary" :loading="saving" @click="handleSubmit">{{ t('misc.save') }}</a-button>
      </div>
    </div>

    <a-table
      class="ai-quota-schedule-drawer__table"
      row-key="id"
      size="small"
      :loading="loading"
      :pagination="false"
      :scroll="{ x: 880 }"
      :data-source="rules"
    >
      <a-table-column key="action" :title="t('aiQuota.schedule.columns.action')">
        <template #default="{ record }">
          <a-tag :color="record.action === 'REFRESH' ? 'blue' : 'green'">
            {{ actionLabel(record.action) }}
          </a-tag>
        </template>
      </a-table-column>
      <a-table-column key="cron" data-index="cron" :title="t('aiQuota.schedule.columns.cron')" />
      <a-table-column key="value" :title="t('aiQuota.schedule.columns.value')">
        <template #default="{ record }">{{ toDisplayAmount(record.value) }}</template>
      </a-table-column>
      <a-table-column key="enabled" :title="t('aiQuota.schedule.columns.enabled')">
        <template #default="{ record }">{{ record.enabled ? t('misc.enabled') : t('misc.disabled') }}</template>
      </a-table-column>
      <a-table-column key="lastAppliedAt" :title="t('aiQuota.schedule.columns.lastAppliedAt')">
        <template #default="{ record }">{{ formatScheduleTime(record.lastAppliedAt) }}</template>
      </a-table-column>
      <a-table-column key="lastError" :title="t('aiQuota.schedule.columns.lastError')">
        <template #default="{ record }">{{ record.lastError || '-' }}</template>
      </a-table-column>
      <a-table-column key="actions" :title="t('aiQuota.columns.actions')" width="140">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="emit('edit', record)">编辑</a-button>
          <a-button type="link" size="small" danger @click="emit('remove', record)">删除</a-button>
        </template>
      </a-table-column>
    </a-table>
  </a-drawer>
</template>

<style scoped>
.ai-quota-schedule-drawer__consumer {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.ai-quota-schedule-drawer__consumer span {
  color: var(--portal-text-soft);
}

.ai-quota-schedule-drawer__actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 18px;
}

.ai-quota-schedule-drawer__submit {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
</style>
