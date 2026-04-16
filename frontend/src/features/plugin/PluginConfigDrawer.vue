<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import BuiltInPluginForm from './BuiltInPluginForm.vue';
import PluginSchemaEditor from './PluginSchemaEditor.vue';
import { showError, showWarning } from '@/lib/feedback';
import {
  AI_DATA_MASKING_PLUGIN_NAME,
  cloneDeep,
  dumpYamlObject,
  getExampleRaw,
  omitAiDataMaskingManagedKeys,
  omitManagedSchema,
  parseYamlObject,
  sanitizeSchemaValue,
  validateSchemaValue,
} from './plugin-config';

const props = defineProps<{
  open: boolean;
  record?: any | null;
  targetDetail?: Record<string, any> | null;
  loading?: boolean;
  instanceLoading?: boolean;
  deleting?: boolean;
  allowDelete?: boolean;
  configData?: any;
  instanceData?: any;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submitBuiltIn: [payload: Record<string, any>];
  submitPlugin: [payload: { enabled: boolean; rawConfigurations: string }];
  deletePlugin: [];
}>();

const { locale } = useI18n();
const activeTab = ref<'form' | 'yaml'>('form');
const builtInRef = ref<InstanceType<typeof BuiltInPluginForm> | null>(null);

const schemaState = reactive<Record<string, any>>({});
const yamlState = ref('');
const enabledState = ref(false);

const currentConfigData = computed(() => {
  if (props.record?.name === AI_DATA_MASKING_PLUGIN_NAME) {
    return omitManagedSchema(props.configData);
  }
  return props.configData;
});

const currentSchema = computed(() => currentConfigData.value?.schema?.jsonSchema);
const canRenderSchemaForm = computed(() => Boolean(currentSchema.value?.properties));

watch(
  () => [props.open, props.record, props.instanceData, currentConfigData.value],
  () => {
    if (!props.open || !props.record || props.record.builtIn) {
      activeTab.value = 'form';
      return;
    }

    enabledState.value = Boolean(props.instanceData?.enabled);
    const exampleRaw = getExampleRaw(currentConfigData.value, !props.record?.queryType && props.record?.category === 'auth');
    const raw = props.instanceData?.rawConfigurations || exampleRaw || '';
    yamlState.value = raw;

    let nextSchema = {};
    try {
      nextSchema = parseYamlObject(raw);
    } catch {
      nextSchema = {};
    }
    Object.keys(schemaState).forEach((key) => delete schemaState[key]);
    Object.assign(schemaState, cloneDeep(nextSchema));
    activeTab.value = canRenderSchemaForm.value ? 'form' : 'yaml';
  },
  { immediate: true, deep: true },
);

watch(activeTab, (nextTab) => {
  if (props.record?.builtIn) {
    return;
  }
  if (nextTab === 'yaml') {
    yamlState.value = dumpYamlObject(sanitizeSchemaValue(cloneDeep(schemaState)));
    return;
  }
  try {
    const parsed = parseYamlObject(yamlState.value);
    Object.keys(schemaState).forEach((key) => delete schemaState[key]);
    Object.assign(schemaState, cloneDeep(parsed));
  } catch {
        showWarning('YAML 解析失败，继续保留当前表单值');
  }
});

function close() {
  emit('update:open', false);
}

function syncYamlFromForm() {
  if (activeTab.value !== 'form' || props.record?.builtIn) {
    return;
  }
  yamlState.value = dumpYamlObject(sanitizeSchemaValue(cloneDeep(schemaState)));
}

watch(schemaState, syncYamlFromForm, { deep: true });

function submit() {
  if (!props.record) {
    return;
  }

  if (props.record.builtIn) {
    const payload = builtInRef.value?.serialize?.();
    if (!payload) {
      return;
    }
    emit('submitBuiltIn', payload);
    return;
  }

  if (activeTab.value === 'form' && canRenderSchemaForm.value) {
    const errors = validateSchemaValue(currentSchema.value, schemaState, locale.value);
    if (errors.length) {
      showError(`请补全必填项：${errors[0]}`);
      return;
    }
    yamlState.value = dumpYamlObject(sanitizeSchemaValue(cloneDeep(schemaState)));
  } else {
    try {
      parseYamlObject(yamlState.value);
    } catch {
      showError('YAML 格式不正确');
      return;
    }
  }

  let rawConfigurations = yamlState.value;
  if (props.record.name === AI_DATA_MASKING_PLUGIN_NAME) {
    rawConfigurations = dumpYamlObject(omitAiDataMaskingManagedKeys(parseYamlObject(rawConfigurations)));
  }

  emit('submitPlugin', {
    enabled: enabledState.value,
    rawConfigurations,
  });
}
</script>

<template>
  <a-drawer
    :open="open"
    width="820"
    :title="record ? `配置 · ${record.title || record.name}` : '插件配置'"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <a-skeleton :loading="Boolean(loading || instanceLoading)" active>
      <div v-if="record?.builtIn">
        <BuiltInPluginForm
          ref="builtInRef"
          :plugin-name="record.name"
          :target-detail="targetDetail || null"
          :state="schemaState"
        />
      </div>

      <div v-else class="plugin-config-drawer">
        <a-alert
          v-if="record?.name === AI_DATA_MASKING_PLUGIN_NAME"
          type="info"
          show-icon
          message="托管敏感词规则不在此处编辑，这里只保留可直接下发的插件配置。"
        />

        <a-form layout="vertical">
          <a-form-item label="启用状态">
            <a-switch v-model:checked="enabledState" />
          </a-form-item>
        </a-form>

        <a-tabs v-model:activeKey="activeTab">
          <a-tab-pane key="form" tab="表单配置">
            <a-empty v-if="!canRenderSchemaForm" description="当前插件未提供可渲染的结构化 Schema，请切换到 YAML。" />
            <PluginSchemaEditor
              v-else
              :schema="currentSchema"
              :state="schemaState"
              :locale="locale"
              :allow-custom-fields="record?.name === AI_DATA_MASKING_PLUGIN_NAME"
            />
          </a-tab-pane>
          <a-tab-pane key="yaml" tab="YAML">
            <a-textarea v-model:value="yamlState" :rows="24" spellcheck="false" />
          </a-tab-pane>
        </a-tabs>
      </div>
    </a-skeleton>
    <DrawerFooter :loading="deleting" @cancel="close" @confirm="submit">
      <template v-if="allowDelete" #extra>
        <a-button danger :loading="deleting" @click="emit('deletePlugin')">删除当前绑定</a-button>
      </template>
    </DrawerFooter>
  </a-drawer>
</template>

<style scoped>
.plugin-config-drawer {
  display: grid;
  gap: 14px;
}
</style>
