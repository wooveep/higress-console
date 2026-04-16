<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import StrategyLink from '@/components/common/StrategyLink.vue';
import StatusTag from '@/components/common/StatusTag.vue';
import { addAiRoute, deleteAiRoute, getAiRoutes, updateAiRoute } from '@/services/ai-route';
import { safeParseJson, splitLines, stringifyPretty } from '@/lib/portal';
import { showSuccess } from '@/lib/feedback';

const loading = ref(false);
const search = ref('');
const rows = ref<any[]>([]);
const drawerOpen = ref(false);
const usageOpen = ref(false);
const deleteOpen = ref(false);
const editing = ref<any>(null);
const deleting = ref<any>(null);
const usageCommand = ref('');

const formState = reactive({
  name: '',
  domainsText: '',
  pathMatchType: 'PRE',
  pathMatchValue: '/v1/chat/completions',
  modelPredicatesJson: '[]',
  upstreamsJson: '[\n  {\n    "provider": "",\n    "weight": 100\n  }\n]',
  allowedConsumerLevels: ['normal'] as string[],
  fallbackEnabled: false,
  fallbackJson: '{\n  "upstreams": []\n}',
});

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

function openDrawer(record?: any) {
  editing.value = record || null;
  Object.assign(formState, {
    name: record?.name || '',
    domainsText: (record?.domains || []).join('\n'),
    pathMatchType: record?.pathPredicate?.matchType || 'PRE',
    pathMatchValue: record?.pathPredicate?.matchValue || '/v1/chat/completions',
    modelPredicatesJson: stringifyPretty(record?.modelPredicates || []),
    upstreamsJson: stringifyPretty(record?.upstreams || []),
    allowedConsumerLevels: record?.authConfig?.allowedConsumerLevels || ['normal'],
    fallbackEnabled: Boolean(record?.fallbackConfig?.enabled),
    fallbackJson: stringifyPretty(record?.fallbackConfig || { upstreams: [] }),
  });
  drawerOpen.value = true;
}

function openUsage(record: any) {
  usageCommand.value = `curl -sv http://<aigateway-gateway-ip>/v1/chat/completions \\
-X POST \\
-H 'Content-Type: application/json'${record.domains?.[0] ? ` \\\n+-H 'Host: ${record.domains[0]}'` : ''} \\
-d '{\n  "model": "<model-name>",\n  "messages": [{"role":"user","content":"Hello!"}]\n}'`;
  usageOpen.value = true;
}

async function submit() {
  const fallbackConfig = safeParseJson(formState.fallbackJson, { upstreams: [] });
  const payload = {
    ...editing.value,
    name: formState.name,
    domains: splitLines(formState.domainsText),
    pathPredicate: {
      matchType: formState.pathMatchType,
      matchValue: formState.pathMatchValue,
    },
    modelPredicates: safeParseJson(formState.modelPredicatesJson, []),
    upstreams: safeParseJson(formState.upstreamsJson, []),
    authConfig: {
      enabled: formState.allowedConsumerLevels.length > 0,
      allowedConsumerLevels: formState.allowedConsumerLevels,
    },
    fallbackConfig: {
      ...fallbackConfig,
      enabled: formState.fallbackEnabled,
    },
  };
  if (editing.value) {
    await updateAiRoute(payload as any);
  } else {
    await addAiRoute(payload as any);
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
    <a-table :data-source="filtered" :loading="loading" row-key="name" :scroll="{ x: 1120 }">
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
      <a-table-column key="actions" title="操作" width="300">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="openUsage(record)">使用说明</a-button>
          <StrategyLink :path="`/ai/route/config?type=aiRoute&name=${encodeURIComponent(record.name)}`" />
          <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          <a-button type="link" size="small" danger @click="deleting = record; deleteOpen = true">删除</a-button>
        </template>
      </a-table-column>
    </a-table>

    <a-drawer v-model:open="drawerOpen" width="760" :title="editing ? '编辑 AI 路由' : '创建 AI 路由'">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="formState.name" :disabled="Boolean(editing)" /></a-form-item>
        <a-form-item label="域名（一行一个）"><a-textarea v-model:value="formState.domainsText" :rows="4" /></a-form-item>
        <a-form-item label="路径匹配方式"><a-input v-model:value="formState.pathMatchType" /></a-form-item>
        <a-form-item label="路径匹配值"><a-input v-model:value="formState.pathMatchValue" /></a-form-item>
        <a-form-item label="模型匹配(JSON)"><a-textarea v-model:value="formState.modelPredicatesJson" :rows="6" /></a-form-item>
        <a-form-item label="上游服务(JSON)"><a-textarea v-model:value="formState.upstreamsJson" :rows="10" /></a-form-item>
        <a-form-item label="允许用户等级">
          <a-select v-model:value="formState.allowedConsumerLevels" mode="multiple">
            <a-select-option value="normal">normal</a-select-option>
            <a-select-option value="plus">plus</a-select-option>
            <a-select-option value="pro">pro</a-select-option>
            <a-select-option value="ultra">ultra</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="启用 fallback"><a-switch v-model:checked="formState.fallbackEnabled" /></a-form-item>
        <a-form-item label="Fallback 配置(JSON)"><a-textarea v-model:value="formState.fallbackJson" :rows="8" /></a-form-item>
      </a-form>
      <DrawerFooter @cancel="drawerOpen = false" @confirm="submit" />
    </a-drawer>

    <a-drawer v-model:open="usageOpen" width="720" title="AI 路由使用方法">
      <pre class="portal-pre">{{ usageCommand }}</pre>
    </a-drawer>

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.name} 吗？` : ''" @confirm="confirmDelete" />
  </PageSection>
</template>
