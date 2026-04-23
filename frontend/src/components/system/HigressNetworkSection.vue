<script setup lang="ts">
import type { HigressGlobalConfigFormState } from '@/interfaces/system';

const props = defineProps<{
  value: HigressGlobalConfigFormState;
}>();

const emit = defineEmits<{
  (e: 'change', value: HigressGlobalConfigFormState): void;
}>();

function emitChange(patch: Partial<HigressGlobalConfigFormState>) {
  emit('change', {
    ...props.value,
    ...patch,
  });
}

function emitDownstreamChange(field: keyof HigressGlobalConfigFormState['downstream'], value: number) {
  emit('change', {
    ...props.value,
    downstream: {
      ...props.value.downstream,
      [field]: value,
    },
  });
}

function emitHTTP2Change(field: keyof HigressGlobalConfigFormState['downstream']['http2'], value: number) {
  emit('change', {
    ...props.value,
    downstream: {
      ...props.value.downstream,
      http2: {
        ...props.value.downstream.http2,
        [field]: value,
      },
    },
  });
}

function emitUpstreamChange(field: keyof HigressGlobalConfigFormState['upstream'], value: number) {
  emit('change', {
    ...props.value,
    upstream: {
      ...props.value.upstream,
      [field]: value,
    },
  });
}
</script>

<template>
  <section class="system-config-section">
    <div class="system-config-section__switches">
      <div class="system-config-switch">
        <div>
          <h3 class="system-config-section__title">addXRealIpHeader</h3>
          <p class="system-config-section__description">为转发请求附加 `x-real-ip` 请求头。</p>
        </div>
        <a-switch :checked="value.addXRealIpHeader" @update:checked="emitChange({ addXRealIpHeader: Boolean($event) })" />
      </div>
      <div class="system-config-switch">
        <div>
          <h3 class="system-config-section__title">disableXEnvoyHeaders</h3>
          <p class="system-config-section__description">关闭转发时自动附加的 `x-envoy-*` 请求头。</p>
        </div>
        <a-switch :checked="value.disableXEnvoyHeaders" @update:checked="emitChange({ disableXEnvoyHeaders: Boolean($event) })" />
      </div>
    </div>

    <div class="system-config-section__block">
      <div class="system-config-section__block-header">
        <h3 class="system-config-section__title">Downstream</h3>
        <p class="system-config-section__description">配置下游连接、请求头限制和 HTTP/2 窗口参数。</p>
      </div>
      <div class="system-config-section__grid system-config-section__grid--four">
        <a-form-item label="Connection Buffer Limits">
          <a-input-number
            :value="value.downstream.connectionBufferLimits"
            :min="0"
            style="width: 100%"
            @update:value="emitDownstreamChange('connectionBufferLimits', Number($event ?? 0))"
          />
        </a-form-item>
        <a-form-item label="Idle Timeout (s)">
          <a-input-number
            :value="value.downstream.idleTimeout"
            :min="0"
            style="width: 100%"
            @update:value="emitDownstreamChange('idleTimeout', Number($event ?? 0))"
          />
        </a-form-item>
        <a-form-item label="Max Request Headers (KB)">
          <a-input-number
            :value="value.downstream.maxRequestHeadersKb"
            :min="0"
            :max="8192"
            style="width: 100%"
            @update:value="emitDownstreamChange('maxRequestHeadersKb', Number($event ?? 0))"
          />
        </a-form-item>
        <a-form-item label="Route Timeout (s)">
          <a-input-number
            :value="value.downstream.routeTimeout"
            :min="0"
            style="width: 100%"
            @update:value="emitDownstreamChange('routeTimeout', Number($event ?? 0))"
          />
        </a-form-item>
      </div>
      <div class="system-config-section__grid system-config-section__grid--three">
        <a-form-item label="HTTP/2 Initial Connection Window Size">
          <a-input-number
            :value="value.downstream.http2.initialConnectionWindowSize"
            :min="65535"
            style="width: 100%"
            @update:value="emitHTTP2Change('initialConnectionWindowSize', Number($event ?? 0))"
          />
        </a-form-item>
        <a-form-item label="HTTP/2 Initial Stream Window Size">
          <a-input-number
            :value="value.downstream.http2.initialStreamWindowSize"
            :min="65535"
            style="width: 100%"
            @update:value="emitHTTP2Change('initialStreamWindowSize', Number($event ?? 0))"
          />
        </a-form-item>
        <a-form-item label="HTTP/2 Max Concurrent Streams">
          <a-input-number
            :value="value.downstream.http2.maxConcurrentStreams"
            :min="1"
            style="width: 100%"
            @update:value="emitHTTP2Change('maxConcurrentStreams', Number($event ?? 0))"
          />
        </a-form-item>
      </div>
    </div>

    <div class="system-config-section__block">
      <div class="system-config-section__block-header">
        <h3 class="system-config-section__title">Upstream</h3>
        <p class="system-config-section__description">配置上游连接缓冲区和空闲超时时间。</p>
      </div>
      <div class="system-config-section__grid system-config-section__grid--two">
        <a-form-item label="Connection Buffer Limits">
          <a-input-number
            :value="value.upstream.connectionBufferLimits"
            :min="0"
            style="width: 100%"
            @update:value="emitUpstreamChange('connectionBufferLimits', Number($event ?? 0))"
          />
        </a-form-item>
        <a-form-item label="Idle Timeout (s)">
          <a-input-number
            :value="value.upstream.idleTimeout"
            :min="0"
            style="width: 100%"
            @update:value="emitUpstreamChange('idleTimeout', Number($event ?? 0))"
          />
        </a-form-item>
      </div>
    </div>
  </section>
</template>

<style scoped>
.system-config-section {
  display: grid;
  gap: 18px;
}

.system-config-section__switches {
  display: grid;
  gap: 14px;
}

.system-config-switch,
.system-config-section__block {
  display: grid;
  gap: 16px;
  padding: 18px;
  border: 1px solid var(--portal-border);
  border-radius: 18px;
  background: var(--portal-surface-soft);
}

.system-config-switch {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
}

.system-config-section__block-header {
  display: grid;
  gap: 6px;
}

.system-config-section__title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
}

.system-config-section__description {
  margin: 0;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.system-config-section__grid {
  display: grid;
  gap: 14px;
}

.system-config-section__grid--four {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.system-config-section__grid--three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.system-config-section__grid--two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

@media (max-width: 1199px) {
  .system-config-section__grid--four {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1023px) {
  .system-config-switch,
  .system-config-section__grid--four,
  .system-config-section__grid--three,
  .system-config-section__grid--two {
    grid-template-columns: 1fr;
  }
}
</style>
