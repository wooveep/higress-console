<script setup lang="ts">
import { onMounted, shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import PageSection from '@/components/common/PageSection.vue';
import HigressGlobalConfigForm from '@/components/system/HigressGlobalConfigForm.vue';
import { useHigressGlobalConfig } from '@/composables/system/useHigressGlobalConfig';
import { getSystemInfo } from '@/services/system';

const router = useRouter();
const { t } = useI18n();
const systemInfoLoading = shallowRef(false);
const systemInfo = shallowRef<Record<string, any>>({});
const configState = useHigressGlobalConfig();

async function loadSystemInfo() {
  systemInfoLoading.value = true;
  try {
    const info = await getSystemInfo().catch(() => ({}));
    systemInfo.value = info || {};
  } finally {
    systemInfoLoading.value = false;
  }
}

async function load() {
  await Promise.all([
    loadSystemInfo(),
    configState.load(),
  ]);
}

onMounted(load);
</script>

<template>
  <div class="system-page">
    <PageSection :title="t('menu.systemSettings')">
      <template #actions>
        <a-button @click="router.push('/system/jobs')">Jobs 运维</a-button>
        <a-button @click="loadSystemInfo">{{ t('misc.refresh') }}</a-button>
      </template>
      <a-skeleton v-if="systemInfoLoading" active />
      <div v-else class="system-page__overview">
        <article
          v-for="(value, key) in systemInfo"
          :key="key"
          class="system-page__stat"
        >
          <span>{{ key }}</span>
          <strong>{{ typeof value === 'object' ? JSON.stringify(value) : String(value) }}</strong>
        </article>
      </div>
    </PageSection>

    <PageSection title="Higress Config">
      <HigressGlobalConfigForm
        :loading="configState.loading.value"
        :saving="configState.saving.value"
        :value="configState.formState.value"
        :raw-yaml="configState.rawYaml.value"
        :dirty="configState.dirty.value"
        :save-disabled="configState.saveDisabled.value"
        :parse-error="configState.parseError.value"
        :save-error="configState.saveError.value"
        :validation-errors="configState.validationErrors.value"
        @refresh="configState.load"
        @save="configState.save"
        @update-form="configState.updateForm"
        @update-raw-yaml="configState.updateRawYaml"
      />
    </PageSection>
  </div>
</template>

<style scoped>
.system-page {
  display: grid;
  gap: 18px;
}

.system-page__overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.system-page__stat {
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--portal-border);
  border-radius: 16px;
  background: var(--portal-surface-soft);
}

.system-page__stat span {
  display: block;
  margin-bottom: 8px;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.system-page__stat strong {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (max-width: 1023px) {
  .system-page__overview {
    grid-template-columns: 1fr;
  }
}
</style>
