<script setup lang="ts">
import type { HigressTracingFormState, TracingBackendKind } from '@/interfaces/system';

type ActiveTracingBackendKind = Exclude<TracingBackendKind, 'none'>;

const props = defineProps<{
  value: HigressTracingFormState;
}>();

const emit = defineEmits<{
  (e: 'change', value: HigressTracingFormState): void;
}>();

function emitChange(patch: Partial<HigressTracingFormState>) {
  emit('change', {
    ...props.value,
    ...patch,
  });
}

function emitBackendChange(kind: ActiveTracingBackendKind, patch: Record<string, string>) {
  emit('change', {
    ...props.value,
    [kind]: {
      ...props.value[kind],
      ...patch,
    },
  });
}
</script>

<template>
  <section class="system-config-section">
    <header class="system-config-section__header">
      <div>
        <h3 class="system-config-section__title">Tracing</h3>
        <p class="system-config-section__description">开启链路追踪并选择唯一的后端类型。</p>
      </div>
      <a-switch :checked="value.enable" @update:checked="emitChange({ enable: Boolean($event) })" />
    </header>

    <div class="system-config-section__grid system-config-section__grid--three">
      <a-form-item label="Sampling">
        <a-input-number
          :value="value.sampling"
          :min="0"
          :max="100"
          :step="1"
          style="width: 100%"
          @update:value="emitChange({ sampling: Number($event ?? 0) })"
        />
      </a-form-item>
      <a-form-item label="Timeout (ms)">
        <a-input-number
          :value="value.timeout"
          :min="1"
          :step="100"
          style="width: 100%"
          @update:value="emitChange({ timeout: Number($event ?? 0) })"
        />
      </a-form-item>
      <a-form-item label="Backend">
        <a-select :value="value.backendKind" @update:value="emitChange({ backendKind: String($event) as TracingBackendKind })">
          <a-select-option value="none">未配置</a-select-option>
          <a-select-option value="skywalking">SkyWalking</a-select-option>
          <a-select-option value="zipkin">Zipkin</a-select-option>
          <a-select-option value="opentelemetry">OpenTelemetry</a-select-option>
        </a-select>
      </a-form-item>
    </div>

    <div v-if="value.backendKind !== 'none'" class="system-config-section__grid system-config-section__grid--three">
      <a-form-item label="Service">
        <a-input
          :value="value[value.backendKind].service"
          placeholder="xxx.namespace.svc.cluster.local"
          @update:value="emitBackendChange(value.backendKind, { service: $event })"
        />
      </a-form-item>
      <a-form-item label="Port">
        <a-input
          :value="value[value.backendKind].port"
          placeholder="11800 / 9411"
          @update:value="emitBackendChange(value.backendKind, { port: $event })"
        />
      </a-form-item>
      <a-form-item v-if="value.backendKind === 'skywalking'" label="Access Token">
        <a-input
          :value="value.skywalking.accessToken"
          placeholder="可选"
          @update:value="emitBackendChange('skywalking', { accessToken: $event })"
        />
      </a-form-item>
    </div>
  </section>
</template>

<style scoped>
.system-config-section {
  display: grid;
  gap: 16px;
  padding: 18px;
  border: 1px solid var(--portal-border);
  border-radius: 18px;
  background: var(--portal-surface-soft);
}

.system-config-section__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.system-config-section__title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
}

.system-config-section__description {
  margin: 6px 0 0;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.system-config-section__grid {
  display: grid;
  gap: 14px;
}

.system-config-section__grid--three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

@media (max-width: 1023px) {
  .system-config-section__grid--three {
    grid-template-columns: 1fr;
  }
}
</style>
