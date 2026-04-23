<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type {
  ModelAsset,
  ModelAssetBinding,
  ModelAssetOptions,
  ModelBindingPriceVersion,
  ProviderModelOption,
} from '@/interfaces/model-asset';
import type { LlmProvider } from '@/interfaces/llm-provider';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import { buildProviderDisplayOptions } from '@/features/llm-provider/provider-display';
import {
  MODEL_TYPE_LABELS,
  applyDefaultPricingByModelType,
  buildPricing,
  clearIrrelevantPricing,
  describePricing,
  getLimitFieldsForType,
  getPricingFieldsForType,
  toBindingFormState,
} from './model-asset-form';

const props = defineProps<{
  open: boolean;
  binding?: ModelAssetBinding | null;
  assets: ModelAsset[];
  selectedAssetId?: string;
  providers: LlmProvider[];
  assetOptions: ModelAssetOptions;
  activePriceVersion?: ModelBindingPriceVersion | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [payload: ModelAssetBinding, isEdit: boolean];
  'open-history': [];
}>();

const { t } = useI18n();
const formState = reactive(toBindingFormState());
const formRef = ref();

const providerModelCatalog = computed(() =>
  (props.assetOptions.providerModels || []).reduce<Record<string, ProviderModelOption[]>>((accumulator, item) => {
    accumulator[item.providerName] = item.models || [];
    return accumulator;
  }, {}),
);
const assetOptions = computed(() =>
  (props.assets || []).map((item) => ({
    label: item.displayName || item.canonicalName || item.assetId,
    value: item.assetId,
  })),
);
const providerOptions = computed(() => buildProviderDisplayOptions(props.providers || [], t, [
  formState.providerName,
]));

const currentProviderModels = computed(() => providerModelCatalog.value[formState.providerName] || []);
const currentProviderUsesCatalog = computed(() => currentProviderModels.value.length > 0);
const currentAssetLabel = computed(() =>
  props.assets.find((item) => item.assetId === formState.selectedAssetId)?.displayName
  || props.assets.find((item) => item.assetId === formState.selectedAssetId)?.canonicalName
  || formState.selectedAssetId,
);
const currentAsset = computed(() =>
  props.assets.find((item) => item.assetId === formState.selectedAssetId)
  || props.assets.find((item) => item.assetId === props.selectedAssetId)
  || null,
);
const currentModelType = computed(() => currentAsset.value?.modelType || '');
const pricingFields = computed(() => getPricingFieldsForType(currentModelType.value));
const limitFields = computed(() => getLimitFieldsForType(currentModelType.value));

watch(() => [props.open, props.binding, props.selectedAssetId], () => {
  Object.assign(formState, toBindingFormState(props.binding || undefined, props.selectedAssetId || ''));
  clearIrrelevantPricing(formState, currentModelType.value);
  if (!props.binding) {
    applyDefaultPricingByModelType(formState, currentModelType.value);
  }
}, { immediate: true });

const currentModelIdOptions = computed(() => {
  const options = currentProviderModels.value.map((item) => ({
    label: item.modelId === item.targetModel ? item.modelId : `${item.modelId} / ${item.targetModel}`,
    value: item.modelId,
  }));
  if (formState.modelId && !currentProviderModels.value.some((item) => item.modelId === formState.modelId)) {
    options.unshift({ label: `历史值 / ${formState.modelId}`, value: formState.modelId });
  }
  return options;
});

const currentTargetModelOptions = computed(() => {
  const options = currentProviderModels.value.map((item) => ({
    label: item.targetModel === item.modelId ? item.targetModel : `${item.targetModel} / ${item.modelId}`,
    value: item.targetModel,
  }));
  if (formState.targetModel && !currentProviderModels.value.some((item) => item.targetModel === formState.targetModel)) {
    options.unshift({ label: `历史值 / ${formState.targetModel}`, value: formState.targetModel });
  }
  return options;
});

const hasLegacyCatalogValue = computed(() =>
  currentProviderUsesCatalog.value
  && (
    (Boolean(formState.modelId) && !currentProviderModels.value.some((item) => item.modelId === formState.modelId))
    || (Boolean(formState.targetModel) && !currentProviderModels.value.some((item) => item.targetModel === formState.targetModel))
  ),
);

watch(() => currentModelType.value, (nextType) => {
  clearIrrelevantPricing(formState, nextType);
  applyDefaultPricingByModelType(formState, nextType);
});

function syncBindingModelPair(field: 'modelId' | 'targetModel', selectedValue?: string) {
  if (!currentProviderUsesCatalog.value) {
    return;
  }
  const matched = currentProviderModels.value.find((item) =>
    field === 'modelId' ? item.modelId === selectedValue : item.targetModel === selectedValue);
  if (matched) {
    formState.modelId = matched.modelId;
    formState.targetModel = matched.targetModel;
  }
}

function handleProviderChange(providerName: string) {
  formState.providerName = providerName;
  const providerModels = providerModelCatalog.value[providerName] || [];
  if (!providerModels.length) {
    return;
  }
  const matched = providerModels.find((item) =>
    item.modelId === formState.modelId || item.targetModel === formState.targetModel);
  if (matched) {
    formState.modelId = matched.modelId;
    formState.targetModel = matched.targetModel;
    return;
  }
  formState.modelId = '';
  formState.targetModel = '';
}

function close() {
  emit('update:open', false);
}

async function submit() {
  await formRef.value?.validate();
  emit('submit', {
    ...(props.binding || {}),
    assetId: formState.selectedAssetId.trim(),
    bindingId: formState.bindingId.trim(),
    modelId: formState.modelId.trim(),
    providerName: formState.providerName.trim(),
    targetModel: formState.targetModel.trim(),
    protocol: formState.protocol.trim() || 'openai/v1',
    endpoint: formState.endpoint.trim(),
    pricing: buildPricing(formState, currentModelType.value),
    limits: {
      maxInputTokens: formState.maxInputTokens,
      maxOutputTokens: formState.maxOutputTokens,
      contextWindowTokens: formState.contextWindowTokens,
      maxReasoningTokens: formState.maxReasoningTokens,
      maxInputTokensInReasoningMode: formState.maxInputTokensInReasoningMode,
      maxOutputTokensInReasoningMode: formState.maxOutputTokensInReasoningMode,
      rpm: formState.rpm,
      tpm: formState.tpm,
      contextWindow: formState.contextWindowTokens,
    },
  }, Boolean(props.binding));
}
</script>

<template>
  <a-drawer
    :open="open"
    width="860"
    :title="binding ? '编辑发布绑定' : '新建发布绑定'"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <a-alert
      v-if="binding"
      type="info"
      show-icon
      style="margin-bottom: 16px"
      :message="`当前状态：${binding.status || 'draft'}`"
      :description="`发布时间：${binding.publishedAt || '-'}；下架时间：${binding.unpublishedAt || '-'}`"
    />

    <a-alert
      v-else
      type="info"
      show-icon
      style="margin-bottom: 16px"
      message="绑定 ID 将在保存后自动生成。"
    />

    <a-card v-if="activePriceVersion" size="small" title="当前生效价格版本" style="margin-bottom: 16px">
      <div class="model-binding-drawer__active">
        <span>版本 #{{ activePriceVersion.versionId }}</span>
        <span>生效时间 {{ activePriceVersion.effectiveFrom || '-' }}</span>
        <span>{{ describePricing(activePriceVersion.pricing, currentModelType) }}</span>
      </div>
    </a-card>

    <a-alert
      v-if="currentModelType"
      type="info"
      show-icon
      style="margin-bottom: 16px"
      :message="`当前模型类型：${MODEL_TYPE_LABELS[currentModelType] || currentModelType}`"
      description="价格项和限流字段会根据模型类型自动切换；不兼容的旧价格会在保存时清理。"
    />

    <a-alert
      v-if="formState.providerName && !currentProviderUsesCatalog"
      type="info"
      show-icon
      style="margin-bottom: 16px"
      message="当前 Provider 未配置系统预置模型目录，模型 ID 和目标模型继续手填即可。"
    />

    <a-alert
      v-if="hasLegacyCatalogValue"
      type="warning"
      show-icon
      style="margin-bottom: 16px"
      message="当前绑定包含历史模型值，建议重新选择预置目录中的模型。"
    />

    <a-form ref="formRef" layout="vertical" :model="formState">
      <div class="model-binding-drawer__grid">
        <a-form-item
          label="模型资产"
          name="selectedAssetId"
          :rules="[{ required: true, message: '请选择模型资产' }]"
        >
          <a-select
            v-if="!binding"
            v-model:value="formState.selectedAssetId"
            show-search
            :options="assetOptions"
          />
          <a-input v-else :value="currentAssetLabel" disabled />
        </a-form-item>
        <a-form-item
          v-if="binding"
          label="绑定 ID"
          name="bindingId"
          :rules="[{ required: true, message: '请输入绑定 ID' }]"
        >
          <a-input v-model:value="formState.bindingId" disabled />
        </a-form-item>
        <a-form-item
          label="Provider"
          name="providerName"
          :rules="[{ required: true, message: '请选择 Provider' }]"
        >
          <a-select
            :value="formState.providerName"
            show-search
            :options="providerOptions"
            @update:value="handleProviderChange"
          />
        </a-form-item>
        <a-form-item
          label="可展示模型 ID"
          name="modelId"
          :rules="[{ required: true, message: '请输入模型 ID' }]"
        >
          <a-select
            v-if="currentProviderUsesCatalog"
            :value="formState.modelId"
            show-search
            :options="currentModelIdOptions"
            @update:value="(value) => syncBindingModelPair('modelId', String(value || ''))"
          />
          <a-input v-else v-model:value="formState.modelId" />
        </a-form-item>
        <a-form-item
          label="目标模型"
          name="targetModel"
          :rules="[{ required: true, message: '请输入目标模型' }]"
        >
          <a-select
            v-if="currentProviderUsesCatalog"
            :value="formState.targetModel"
            show-search
            :options="currentTargetModelOptions"
            @update:value="(value) => syncBindingModelPair('targetModel', String(value || ''))"
          />
          <a-input v-else v-model:value="formState.targetModel" />
        </a-form-item>
        <a-form-item label="协议">
          <a-input v-model:value="formState.protocol" />
        </a-form-item>
        <a-form-item label="入口地址">
          <a-input v-model:value="formState.endpoint" />
        </a-form-item>
      </div>

      <a-divider orientation="left">限制</a-divider>
      <div class="model-binding-drawer__grid model-binding-drawer__grid--compact">
        <a-form-item
          v-for="field in limitFields"
          :key="field.name"
          :label="field.label"
        >
          <a-input-number
            :value="typeof formState[field.name] === 'number' ? formState[field.name] as number : undefined"
            style="width: 100%"
            :min="0"
            :step="1"
            @update:value="(value) => ((formState as any)[field.name] = value ?? undefined)"
          />
        </a-form-item>
      </div>

      <a-divider orientation="left">价格</a-divider>
      <div class="model-binding-drawer__grid model-binding-drawer__grid--compact">
        <a-form-item label="币种">
          <a-input v-model:value="formState.currency" />
        </a-form-item>
      </div>

      <div v-if="!pricingFields.length" class="model-binding-drawer__empty">
        请先选择模型资产并设置模型类型。
      </div>

      <div v-else class="model-binding-drawer__grid">
        <a-form-item
          v-for="field in pricingFields"
          :key="field.name"
          :label="field.label"
          :extra="field.unit"
        >
          <a-input-number
            :value="typeof formState[field.name] === 'number' ? formState[field.name] as number : undefined"
            style="width: 100%"
            :min="0"
            :step="0.000001"
            @update:value="(value) => ((formState as any)[field.name] = value ?? undefined)"
          />
        </a-form-item>
      </div>
    </a-form>

    <DrawerFooter @cancel="close" @confirm="submit">
      <template #extra>
        <a-button v-if="binding" @click="emit('open-history')">价格历史</a-button>
      </template>
    </DrawerFooter>
  </a-drawer>
</template>

<style scoped>
.model-binding-drawer__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 14px;
}

.model-binding-drawer__grid--compact {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.model-binding-drawer__active {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.model-binding-drawer__empty {
  padding: 20px 16px;
  border: 1px dashed var(--portal-border);
  border-radius: 12px;
  background: var(--portal-surface-soft);
  color: var(--portal-text-soft);
}

@media (max-width: 960px) {
  .model-binding-drawer__grid,
  .model-binding-drawer__grid--compact {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
