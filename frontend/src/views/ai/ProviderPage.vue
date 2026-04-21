<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import SecretMaskText from '@/components/common/SecretMaskText.vue';
import ProviderDrawer from '@/features/llm-provider/ProviderDrawer.vue';
import { getProviderCredentialValues, getProviderTypeLabel } from '@/features/llm-provider/provider-form';
import type { LlmProvider } from '@/interfaces/llm-provider';
import { addLlmProvider, deleteLlmProvider, getLlmProviders, updateLlmProvider } from '@/services/llm-provider';
import { showSuccess } from '@/lib/feedback';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const loading = ref(false);
const search = ref('');
const rows = ref<Array<LlmProvider & Record<string, any>>>([]);
const drawerOpen = ref(false);
const deleteOpen = ref(false);
const editing = ref<(LlmProvider & Record<string, any>) | null>(null);
const deleting = ref<(LlmProvider & Record<string, any>) | null>(null);

const filtered = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, item.type, item.protocol, item.proxyName].some((value) => String(value || '').toLowerCase().includes(keyword));
}));

async function load() {
  loading.value = true;
  try {
    rows.value = await getLlmProviders().catch(() => []);
  } finally {
    loading.value = false;
  }
}

function openDrawer(record?: LlmProvider & Record<string, any>) {
  editing.value = record || null;
  drawerOpen.value = true;
}

async function submit(payload: LlmProvider & Record<string, any>, isEdit: boolean) {
  if (editing.value) {
    await updateLlmProvider(payload as any);
  } else {
    await addLlmProvider(payload as any);
  }
  drawerOpen.value = false;
  await load();
  showSuccess(isEdit ? 'Provider 已更新' : 'Provider 已创建');
}

async function confirmDelete() {
  if (!deleting.value) {
    return;
  }
  await deleteLlmProvider(deleting.value.name);
  deleteOpen.value = false;
  await load();
  showSuccess('删除成功');
}

onMounted(load);
</script>

<template>
  <PageSection title="AI 服务提供者管理">
    <ListToolbar v-model:search="search" search-placeholder="搜索名称、类型、协议" create-text="新增 Provider" @refresh="load" @create="openDrawer()" />
    <a-table :data-source="filtered" :loading="loading" row-key="name" :scroll="{ x: 980 }">
      <a-table-column key="type" data-index="type" :title="t('llmProvider.columns.type')">
        <template #default="{ record }">
          {{ getProviderTypeLabel(record.type, t) }}
        </template>
      </a-table-column>
      <a-table-column key="name" data-index="name" title="名称" />
      <a-table-column key="protocol" data-index="protocol" title="协议" />
      <a-table-column key="proxyName" data-index="proxyName" title="代理服务" />
      <a-table-column key="tokens" title="Tokens" width="220">
        <template #default="{ record }">
          <div class="provider-page__tokens">
            <SecretMaskText
              v-for="token in getProviderCredentialValues(record)"
              :key="token"
              :value="token"
            />
            <span v-if="!getProviderCredentialValues(record).length">-</span>
          </div>
        </template>
      </a-table-column>
      <a-table-column key="actions" title="操作" width="180">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          <a-button type="link" size="small" danger @click="deleting = record; deleteOpen = true">删除</a-button>
        </template>
      </a-table-column>
    </a-table>

    <ProviderDrawer
      v-model:open="drawerOpen"
      :provider="editing"
      @submit="submit"
    />

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.name} 吗？` : ''" @confirm="confirmDelete" />
  </PageSection>
</template>

<style scoped>
.provider-page__tokens {
  display: grid;
  gap: 6px;
}
</style>
