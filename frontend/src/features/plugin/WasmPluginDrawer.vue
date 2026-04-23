<script setup lang="ts">
import { reactive, watch } from 'vue';
import { ImagePullPolicy, PluginPhase, type WasmPluginData } from '@/interfaces/wasm-plugin';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import { buildPluginImageUrl, splitPluginImageUrl } from './plugin-config';
import { showError } from '@/lib/feedback';

const props = defineProps<{
  open: boolean;
  record?: WasmPluginData | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [payload: WasmPluginData, isEdit: boolean];
}>();

const formState = reactive({
  name: '',
  title: '',
  category: 'custom',
  description: '',
  imageUrl: '',
  phase: PluginPhase.UNSPECIFIED,
  priority: 100,
  imagePullPolicy: ImagePullPolicy.UNSPECIFIED,
  imagePullSecret: '',
});

watch(() => [props.open, props.record], () => {
  Object.assign(formState, {
    name: props.record?.name || '',
    title: props.record?.title || props.record?.name || '',
    category: props.record?.category || 'custom',
    description: props.record?.description || '',
    imageUrl: buildPluginImageUrl(props.record || undefined),
    phase: props.record?.phase || PluginPhase.UNSPECIFIED,
    priority: props.record?.priority || 100,
    imagePullPolicy: props.record?.imagePullPolicy || ImagePullPolicy.UNSPECIFIED,
    imagePullSecret: props.record?.imagePullSecret || '',
  });
}, { immediate: true });

function close() {
  emit('update:open', false);
}

function submit() {
  if (!formState.name.trim()) {
    showError('请输入插件名称');
    return;
  }
  if (!props.record && !formState.imageUrl.trim()) {
    showError('请输入镜像地址');
    return;
  }

  const imageParts = splitPluginImageUrl(formState.imageUrl);
  emit('submit', {
    ...(props.record || {}),
    name: formState.name.trim(),
    title: formState.title.trim() || formState.name.trim(),
    category: formState.category.trim() || 'custom',
    description: formState.description.trim(),
    phase: formState.phase,
    priority: formState.priority,
    imagePullPolicy: formState.imagePullPolicy,
    imagePullSecret: formState.imagePullSecret.trim(),
    ...imageParts,
  }, Boolean(props.record));
}
</script>

<template>
  <a-drawer
    :open="open"
    width="720"
    :title="record ? '编辑插件' : '新增插件'"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <a-form layout="vertical">
      <a-form-item label="名称">
        <a-input v-model:value="formState.name" :disabled="Boolean(record)" />
      </a-form-item>
      <a-form-item label="标题">
        <a-input v-model:value="formState.title" />
      </a-form-item>
      <a-form-item label="分类">
        <a-input v-model:value="formState.category" />
      </a-form-item>
      <a-form-item label="描述">
        <a-textarea v-model:value="formState.description" :rows="4" />
      </a-form-item>
      <a-form-item label="镜像地址">
        <a-input v-model:value="formState.imageUrl" placeholder="registry.example.com/plugin:latest" />
      </a-form-item>
      <div class="wasm-plugin-drawer__grid">
        <a-form-item label="执行阶段">
          <a-select v-model:value="formState.phase">
            <a-select-option :value="PluginPhase.UNSPECIFIED">未指定</a-select-option>
            <a-select-option :value="PluginPhase.AUTHN">AUTHN</a-select-option>
            <a-select-option :value="PluginPhase.AUTHZ">AUTHZ</a-select-option>
            <a-select-option :value="PluginPhase.STATS">STATS</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="优先级">
          <a-input-number v-model:value="formState.priority" style="width: 100%" :min="1" :max="1000" />
        </a-form-item>
        <a-form-item label="拉取策略">
          <a-select v-model:value="formState.imagePullPolicy">
            <a-select-option :value="ImagePullPolicy.UNSPECIFIED">未指定</a-select-option>
            <a-select-option :value="ImagePullPolicy.IF_NOT_PRESENT">IfNotPresent</a-select-option>
            <a-select-option :value="ImagePullPolicy.ALWAYS">Always</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="拉取密钥">
          <a-input v-model:value="formState.imagePullSecret" placeholder="可选" />
        </a-form-item>
      </div>
    </a-form>
    <DrawerFooter @cancel="close" @confirm="submit" />
  </a-drawer>
</template>

<style scoped>
.wasm-plugin-drawer__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 14px;
}

@media (max-width: 900px) {
  .wasm-plugin-drawer__grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
