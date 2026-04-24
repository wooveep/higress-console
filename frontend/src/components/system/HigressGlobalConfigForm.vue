<script setup lang="ts">
import type { HigressGlobalConfigFormState } from '@/interfaces/system';
import HigressTracingSection from './HigressTracingSection.vue';
import HigressGzipSection from './HigressGzipSection.vue';
import HigressNetworkSection from './HigressNetworkSection.vue';

defineProps<{
  loading: boolean;
  saving: boolean;
  value: HigressGlobalConfigFormState;
  rawYaml: string;
  dirty: boolean;
  saveDisabled: boolean;
  parseError: string;
  saveError: string;
  validationErrors: string[];
}>();

const emit = defineEmits<{
  (e: 'refresh'): void;
  (e: 'save'): void;
  (e: 'update-form', value: HigressGlobalConfigFormState): void;
  (e: 'update-raw-yaml', value: string): void;
}>();
</script>

<template>
  <div class="higress-config-form">
    <div class="higress-config-form__toolbar">
      <div class="higress-config-form__summary">
        <h2 class="higress-config-form__title">AIGateway 配置</h2>
      </div>
      <div class="higress-config-form__actions">
        <a-tag v-if="dirty" color="gold">有未保存修改</a-tag>
        <a-button @click="emit('refresh')">刷新</a-button>
        <a-button type="primary" :loading="saving" :disabled="saveDisabled" @click="emit('save')">保存</a-button>
      </div>
    </div>

    <a-skeleton v-if="loading" active />

    <template v-else>
      <a-alert v-if="parseError" type="error" show-icon :message="parseError" />
      <a-alert v-else-if="validationErrors.length" type="warning" show-icon message="当前配置校验未通过">
        <template #description>
          <ul class="higress-config-form__error-list">
            <li v-for="item in validationErrors" :key="item">{{ item }}</li>
          </ul>
        </template>
      </a-alert>
      <a-alert v-if="saveError" type="error" show-icon :message="saveError" />

      <a-form layout="vertical" class="higress-config-form__layout">
        <HigressNetworkSection :value="value" @change="emit('update-form', $event)" />
        <HigressTracingSection :value="value.tracing" @change="emit('update-form', { ...value, tracing: $event })" />
        <HigressGzipSection :value="value.gzip" @change="emit('update-form', { ...value, gzip: $event })" />
      </a-form>

      <section class="higress-config-form__editor">
        <div class="higress-config-form__editor-header">
          <h3 class="higress-config-form__editor-title">高级 YAML</h3>
          <p class="higress-config-form__editor-description">可直接编辑完整的 `data.higress` 原始 YAML，合法修改会自动回灌表单。</p>
        </div>
        <a-textarea
          :value="rawYaml"
          :rows="18"
          spellcheck="false"
          @update:value="emit('update-raw-yaml', $event)"
        />
      </section>
    </template>
  </div>
</template>

<style scoped>
.higress-config-form {
  display: grid;
  gap: 18px;
}

.higress-config-form__toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.higress-config-form__summary {
  display: grid;
  gap: 6px;
}

.higress-config-form__title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
}

.higress-config-form__actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.higress-config-form__layout {
  display: grid;
  gap: 18px;
}

.higress-config-form__editor {
  display: grid;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--portal-border);
  border-radius: 18px;
  background: var(--portal-surface-soft);
}

.higress-config-form__editor-header {
  display: grid;
  gap: 6px;
}

.higress-config-form__editor-title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
}

.higress-config-form__editor-description {
  margin: 0;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.higress-config-form__error-list {
  margin: 0;
  padding-left: 18px;
}

@media (max-width: 1023px) {
  .higress-config-form__toolbar {
    flex-direction: column;
  }

  .higress-config-form__actions {
    width: 100%;
    justify-content: flex-end;
    flex-wrap: wrap;
  }
}
</style>
