<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue';
import { createDefaultPolicyValues, fillPolicyResetNow, type AiQuotaPolicyFormValues } from '@/features/ai-quota/ai-quota-utils';
import type { AiQuotaUserPolicy } from '@/interfaces/ai-quota';
import { useI18n } from 'vue-i18n';

const props = defineProps<{
  open: boolean;
  consumerName?: string;
  initialValues?: Partial<AiQuotaUserPolicy>;
  loading?: boolean;
  submitting?: boolean;
}>();

const emit = defineEmits<{
  close: [];
  submit: [values: AiQuotaPolicyFormValues];
  reload: [];
}>();

const { t } = useI18n();
const formRef = ref<any>();
const formState = reactive<AiQuotaPolicyFormValues>(createDefaultPolicyValues());

const mergedInitialValues = computed(() => createDefaultPolicyValues(props.initialValues));

watch(() => props.open, async (open) => {
  if (open) {
    await nextTick();
    Object.assign(formState, mergedInitialValues.value);
  } else {
    formRef.value?.resetFields();
  }
}, { immediate: true });

watch(mergedInitialValues, async (values) => {
  if (props.open) {
    await nextTick();
    Object.assign(formState, values);
  }
});

async function handleSubmit() {
  await formRef.value?.validate();
  emit('submit', { ...formState });
}

function handleFillNow() {
  formState.costResetAt = fillPolicyResetNow();
}

function handleClearResetAt() {
  formState.costResetAt = '';
}
</script>

<template>
  <a-drawer
    :open="open"
    width="640"
    :title="t('aiQuota.policy.title')"
    destroy-on-close
    @close="emit('close')"
  >
    <a-alert
      class="ai-quota-policy-drawer__alert"
      type="info"
      show-icon
      :message="t('aiQuota.policy.description')"
    />

    <div class="ai-quota-policy-drawer__consumer">
      <span>{{ t('aiQuota.columns.consumer') }}</span>
      <strong>{{ consumerName || '-' }}</strong>
    </div>

    <a-form ref="formRef" :model="formState" layout="vertical">
      <a-form-item :label="t('aiQuota.policy.form.dailyResetMode')">
        <span>{{ t('aiQuota.policy.form.fixedResetMode') }}</span>
      </a-form-item>

      <a-form-item
        name="limitTotal"
        :label="`${t('aiQuota.policy.form.limitTotal')} (RMB)`"
        :extra="t('aiQuota.policy.form.amountHelp')"
        :rules="[{ required: true, message: t('aiQuota.validation.policyLimitRequired') }]"
      >
        <a-input-number v-model:value="formState.limitTotal" :min="0" :precision="6" :step="0.01" style="width: 100%" />
      </a-form-item>

      <a-form-item
        name="limit5h"
        :label="`${t('aiQuota.policy.form.limit5h')} (RMB)`"
        :extra="t('aiQuota.policy.form.amountHelp')"
        :rules="[{ required: true, message: t('aiQuota.validation.policyLimitRequired') }]"
      >
        <a-input-number v-model:value="formState.limit5h" :min="0" :precision="6" :step="0.01" style="width: 100%" />
      </a-form-item>

      <a-form-item
        name="limitDaily"
        :label="`${t('aiQuota.policy.form.limitDaily')} (RMB)`"
        :extra="t('aiQuota.policy.form.amountHelp')"
        :rules="[{ required: true, message: t('aiQuota.validation.policyLimitRequired') }]"
      >
        <a-input-number v-model:value="formState.limitDaily" :min="0" :precision="6" :step="0.01" style="width: 100%" />
      </a-form-item>

      <a-form-item
        name="dailyResetTime"
        :label="t('aiQuota.policy.form.dailyResetTime')"
        :extra="t('aiQuota.policy.form.dailyResetTimeHelp')"
        :rules="[
          { required: true, message: t('aiQuota.validation.policyDailyResetTimeRequired') },
          { pattern: /^([01][0-9]|2[0-3]):[0-5][0-9]$/, message: t('aiQuota.validation.policyDailyResetTimeInvalid') },
        ]"
      >
        <a-input v-model:value="formState.dailyResetTime" placeholder="00:00" />
      </a-form-item>

      <a-form-item
        name="limitWeekly"
        :label="`${t('aiQuota.policy.form.limitWeekly')} (RMB)`"
        :extra="t('aiQuota.policy.form.amountHelp')"
        :rules="[{ required: true, message: t('aiQuota.validation.policyLimitRequired') }]"
      >
        <a-input-number v-model:value="formState.limitWeekly" :min="0" :precision="6" :step="0.01" style="width: 100%" />
      </a-form-item>

      <a-form-item
        name="limitMonthly"
        :label="`${t('aiQuota.policy.form.limitMonthly')} (RMB)`"
        :extra="t('aiQuota.policy.form.amountHelp')"
        :rules="[{ required: true, message: t('aiQuota.validation.policyLimitRequired') }]"
      >
        <a-input-number v-model:value="formState.limitMonthly" :min="0" :precision="6" :step="0.01" style="width: 100%" />
      </a-form-item>

      <a-form-item
        name="costResetAt"
        :label="t('aiQuota.policy.form.costResetAt')"
        :extra="t('aiQuota.policy.form.costResetAtHelp')"
      >
        <a-input v-model:value="formState.costResetAt" placeholder="2026-03-27T10:30" />
      </a-form-item>
    </a-form>

    <div class="ai-quota-policy-drawer__actions">
      <div class="ai-quota-policy-drawer__ghost-actions">
        <a-button @click="handleFillNow">{{ t('aiQuota.policy.actions.setResetNow') }}</a-button>
        <a-button @click="handleClearResetAt">{{ t('aiQuota.policy.actions.clearResetAt') }}</a-button>
      </div>
      <div class="ai-quota-policy-drawer__submit-actions">
        <a-button :loading="loading" @click="emit('reload')">{{ t('misc.reset') }}</a-button>
        <a-button type="primary" :loading="submitting" @click="handleSubmit">{{ t('misc.save') }}</a-button>
      </div>
    </div>
  </a-drawer>
</template>

<style scoped>
.ai-quota-policy-drawer__alert {
  margin-bottom: 16px;
}

.ai-quota-policy-drawer__consumer {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.ai-quota-policy-drawer__consumer span {
  color: var(--portal-text-soft);
}

.ai-quota-policy-drawer__actions {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.ai-quota-policy-drawer__ghost-actions,
.ai-quota-policy-drawer__submit-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
</style>
