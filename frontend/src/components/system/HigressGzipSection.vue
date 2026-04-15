<script setup lang="ts">
import type { HigressGzipFormState } from '@/interfaces/system';
import {
  HIGRESS_COMPRESSION_LEVELS,
  HIGRESS_COMPRESSION_STRATEGIES,
} from '@/composables/system/higress-global-config-utils';

const props = defineProps<{
  value: HigressGzipFormState;
}>();

const emit = defineEmits<{
  (e: 'change', value: HigressGzipFormState): void;
}>();

function emitChange(patch: Partial<HigressGzipFormState>) {
  emit('change', {
    ...props.value,
    ...patch,
  });
}

function onContentTypeChange(value: string) {
  emitChange({
    contentType: value.split('\n').map((item) => item.trim()).filter(Boolean),
  });
}
</script>

<template>
  <section class="system-config-section">
    <header class="system-config-section__header">
      <div>
        <h3 class="system-config-section__title">Gzip</h3>
        <p class="system-config-section__description">配置压缩开关、压缩阈值和 zlib 参数。</p>
      </div>
      <a-switch :checked="value.enable" @update:checked="emitChange({ enable: Boolean($event) })" />
    </header>

    <div class="system-config-section__grid system-config-section__grid--three">
      <a-form-item label="Min Content Length">
        <a-input-number
          :value="value.minContentLength"
          :min="1"
          style="width: 100%"
          @update:value="emitChange({ minContentLength: Number($event ?? 0) })"
        />
      </a-form-item>
      <a-form-item label="Memory Level">
        <a-input-number
          :value="value.memoryLevel"
          :min="1"
          :max="9"
          style="width: 100%"
          @update:value="emitChange({ memoryLevel: Number($event ?? 0) })"
        />
      </a-form-item>
      <a-form-item label="Window Bits">
        <a-input-number
          :value="value.windowBits"
          :min="9"
          :max="15"
          style="width: 100%"
          @update:value="emitChange({ windowBits: Number($event ?? 0) })"
        />
      </a-form-item>
      <a-form-item label="Chunk Size">
        <a-input-number
          :value="value.chunkSize"
          :min="1"
          style="width: 100%"
          @update:value="emitChange({ chunkSize: Number($event ?? 0) })"
        />
      </a-form-item>
      <a-form-item label="Compression Level">
        <a-select :value="value.compressionLevel" @update:value="emitChange({ compressionLevel: String($event) })">
          <a-select-option v-for="item in HIGRESS_COMPRESSION_LEVELS" :key="item" :value="item">
            {{ item }}
          </a-select-option>
        </a-select>
      </a-form-item>
      <a-form-item label="Compression Strategy">
        <a-select :value="value.compressionStrategy" @update:value="emitChange({ compressionStrategy: String($event) })">
          <a-select-option v-for="item in HIGRESS_COMPRESSION_STRATEGIES" :key="item" :value="item">
            {{ item }}
          </a-select-option>
        </a-select>
      </a-form-item>
    </div>

    <div class="system-config-section__grid system-config-section__grid--two">
      <a-form-item label="Content Types">
        <a-textarea
          :value="value.contentType.join('\n')"
          :rows="6"
          spellcheck="false"
          placeholder="一行一个 content-type"
          @update:value="onContentTypeChange"
        />
      </a-form-item>
      <a-form-item label="Disable On ETag Header">
        <a-switch :checked="value.disableOnEtagHeader" @update:checked="emitChange({ disableOnEtagHeader: Boolean($event) })" />
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

.system-config-section__grid--two {
  grid-template-columns: minmax(0, 2fr) minmax(260px, 1fr);
}

@media (max-width: 1023px) {
  .system-config-section__grid--three,
  .system-config-section__grid--two {
    grid-template-columns: 1fr;
  }
}
</style>
