<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import StrategyLink from '@/components/common/StrategyLink.vue';
import StatusTag from '@/components/common/StatusTag.vue';
import AiRouteDrawer from '@/features/routes/AiRouteDrawer.vue';
import type { AiRoute } from '@/interfaces/ai-route';
import { showSuccess } from '@/lib/feedback';
import { addAiRoute, deleteAiRoute, getAiRoutes, updateAiRoute } from '@/services/ai-route';

const loading = ref(false);
const search = ref('');
const rows = ref<AiRoute[]>([]);
const drawerOpen = ref(false);
const usageOpen = ref(false);
const deleteOpen = ref(false);
const editing = ref<AiRoute | null>(null);
const deleting = ref<AiRoute | null>(null);
const usageCommand = ref('');

const filtered = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, (item.domains || []).join(','), item.pathPredicate?.matchValue, formatRouteDomainDisplay(item)]
    .some((value) => String(value || '').toLowerCase().includes(keyword));
}));

function formatRouteDomainDisplay(route: { domains?: string[]; pathPredicate?: { matchValue?: string } }) {
  const domains = (route.domains || []).map((item) => String(item || '').trim()).filter(Boolean);
  if (domains.length > 0) {
    return domains.join(', ');
  }
  const internalPath = String(route.pathPredicate?.matchValue || '').trim();
  if (internalPath) {
    return `内部路由 · ${internalPath}`;
  }
  return '内部路由';
}

async function load() {
  loading.value = true;
  try {
    const result = await getAiRoutes().catch(() => ({ data: [] }));
    rows.value = Array.isArray(result) ? result : (result.data || []);
  } finally {
    loading.value = false;
  }
}

function openDrawer(record?: AiRoute) {
  editing.value = record || null;
  drawerOpen.value = true;
}

function openUsage(record: AiRoute) {
  usageCommand.value = `curl -sv http://<aigateway-gateway-ip>/v1/chat/completions \\
-X POST \\
-H 'Content-Type: application/json'${record.domains?.[0] ? ` \\\n+-H 'Host: ${record.domains[0]}'` : ''} \\
-d '{\n  "model": "<model-name>",\n  "messages": [{"role":"user","content":"Hello!"}]\n}'`;
  usageOpen.value = true;
}

async function submit(payload: AiRoute, isEdit: boolean) {
  if (isEdit) {
    await updateAiRoute(payload);
  } else {
    await addAiRoute(payload);
  }
  drawerOpen.value = false;
  await load();
  showSuccess('保存成功');
}

async function confirmDelete() {
  if (!deleting.value) {
    return;
  }
  await deleteAiRoute(deleting.value.name);
  deleteOpen.value = false;
  await load();
  showSuccess('删除成功');
}

onMounted(load);
</script>

<template>
  <PageSection title="AI 路由管理">
    <ListToolbar v-model:search="search" search-placeholder="搜索 AI 路由名或域名" create-text="创建 AI 路由" @refresh="load" @create="openDrawer()" />
    <a-table :data-source="filtered" :loading="loading" row-key="name" :scroll="{ x: 1180 }">
      <a-table-column key="name" data-index="name" title="名称" />
      <a-table-column key="domains" title="域名">
        <template #default="{ record }">{{ formatRouteDomainDisplay(record) }}</template>
      </a-table-column>
      <a-table-column key="pathPredicate" title="路径匹配">
        <template #default="{ record }">{{ record.pathPredicate?.matchType }} | {{ record.pathPredicate?.matchValue }}</template>
      </a-table-column>
      <a-table-column key="upstreams" title="上游">
        <template #default="{ record }">{{ (record.upstreams || []).map((item: any) => `${item.provider}:${item.weight || 0}%`).join(', ') || '-' }}</template>
      </a-table-column>
      <a-table-column key="auth" title="认证">
        <template #default="{ record }"><StatusTag :value="record.authConfig?.enabled ? 'enabled' : 'disabled'" /></template>
      </a-table-column>
      <a-table-column key="actions" title="操作" width="320">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="openUsage(record)">使用说明</a-button>
          <StrategyLink :path="`/ai/route/config?type=aiRoute&name=${encodeURIComponent(record.name)}`" />
          <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          <a-button type="link" size="small" danger @click="deleting = record; deleteOpen = true">删除</a-button>
        </template>
      </a-table-column>
    </a-table>

    <AiRouteDrawer
      v-model:open="drawerOpen"
      :route="editing"
      @submit="submit"
    />

    <a-drawer v-model:open="usageOpen" width="720" title="AI 路由使用方法">
      <pre class="portal-pre">{{ usageCommand }}</pre>
    </a-drawer>

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.name} 吗？` : ''" @confirm="confirmDelete" />
  </PageSection>
</template>
