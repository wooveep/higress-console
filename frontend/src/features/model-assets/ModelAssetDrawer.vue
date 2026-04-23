<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import type { ModelAsset, ModelAssetOptions } from '@/interfaces/model-asset';
import { MODEL_ASSET_PRESET_TAGS } from '@/interfaces/model-asset';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import {
  FEATURE_FLAG_LABELS,
  MODALITY_LABELS,
  MODEL_TYPE_LABELS,
  hasLegacyAssetValues,
  toAssetFormState,
} from './model-asset-form';

const props = defineProps<{
  open: boolean;
  asset?: ModelAsset | null;
  assetOptions: ModelAssetOptions;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [payload: ModelAsset, isEdit: boolean];
}>();

const formState = reactive(toAssetFormState());
const formRef = ref();

watch(() => [props.open, props.asset], () => {
  Object.assign(formState, toAssetFormState(props.asset || undefined));
}, { immediate: true });

const legacyFlags = computed(() => hasLegacyAssetValues(props.asset || undefined, props.assetOptions));

function close() {
  emit('update:open', false);
}

async function submit() {
  await formRef.value?.validate();
  emit('submit', {
    ...(props.asset || {}),
    assetId: formState.assetId.trim(),
    canonicalName: formState.canonicalName.trim(),
    displayName: formState.displayName.trim(),
    intro: formState.intro.trim(),
    modelType: formState.modelType,
    tags: [...formState.tags],
    capabilities: {
      inputModalities: [...formState.inputModalities],
      outputModalities: [...formState.outputModalities],
      featureFlags: [...formState.featureFlags],
    },
  }, Boolean(props.asset));
}
</script>

<template>
  <a-drawer
    :open="open"
    width="700"
    :title="asset ? '编辑模型资产' : '新建模型资产'"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <a-alert
      v-if="legacyFlags.tags || legacyFlags.capabilities"
      type="warning"
      show-icon
      style="margin-bottom: 16px"
      message="该资产包含历史非预置字段，保存后会按当前预置选项收口。"
    />
    <a-form ref="formRef" layout="vertical" :model="formState">
      <a-form-item
        label="资产 ID"
        name="assetId"
        :rules="asset ? [{ required: true, message: '请输入资产 ID' }] : []"
      >
        <a-input
          v-model:value="formState.assetId"
          :disabled="Boolean(asset)"
          :placeholder="asset ? '' : '留空则自动生成 UUID'"
        />
      </a-form-item>
      <a-form-item
        label="规范名"
        name="canonicalName"
        :rules="[{ required: true, message: '请输入规范名' }]"
      >
        <a-input v-model:value="formState.canonicalName" placeholder="例如 openai/gpt-4o-mini" />
      </a-form-item>
      <a-form-item
        label="展示名"
        name="displayName"
        :rules="[{ required: true, message: '请输入展示名' }]"
      >
        <a-input v-model:value="formState.displayName" placeholder="例如 GPT-4o mini" />
      </a-form-item>
      <a-form-item label="简介">
        <a-textarea v-model:value="formState.intro" :rows="4" />
      </a-form-item>
      <a-form-item
        label="模型类型"
        name="modelType"
        :rules="[{ required: true, message: '请选择模型类型' }]"
      >
        <a-select
          v-model:value="formState.modelType"
          :options="(assetOptions.capabilities?.modelTypes || []).map((item) => ({ label: MODEL_TYPE_LABELS[item] || item, value: item }))"
        />
      </a-form-item>
      <a-form-item label="标签">
        <a-select
          v-model:value="formState.tags"
          mode="multiple"
          :options="MODEL_ASSET_PRESET_TAGS.map((tag) => ({ label: tag, value: tag }))"
        />
      </a-form-item>
      <a-form-item label="输入模态">
        <a-select
          v-model:value="formState.inputModalities"
          mode="multiple"
          :options="(assetOptions.capabilities?.inputModalities || []).map((item) => ({ label: MODALITY_LABELS[item] || item, value: item }))"
        />
      </a-form-item>
      <a-form-item label="输出模态">
        <a-select
          v-model:value="formState.outputModalities"
          mode="multiple"
          :options="(assetOptions.capabilities?.outputModalities || []).map((item) => ({ label: MODALITY_LABELS[item] || item, value: item }))"
        />
      </a-form-item>
      <a-form-item label="固定能力标签">
        <a-select
          v-model:value="formState.featureFlags"
          mode="multiple"
          :options="(assetOptions.capabilities?.featureFlags || []).map((item) => ({ label: FEATURE_FLAG_LABELS[item] || item, value: item }))"
        />
      </a-form-item>
    </a-form>
    <DrawerFooter @cancel="close" @confirm="submit" />
  </a-drawer>
</template>
