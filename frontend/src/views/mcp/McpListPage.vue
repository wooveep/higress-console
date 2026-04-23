<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import { createOrUpdateMcpServer, deleteMcpServer, listMcpServers } from '@/services/mcp';
import { joinLines, splitLines } from '@/lib/portal';
import { showSuccess } from '@/lib/feedback';

const router = useRouter();
const loading = ref(false);
const search = ref('');
const typeFilter = ref<string | undefined>();
const rows = ref<any[]>([]);
const drawerOpen = ref(false);
const deleteOpen = ref(false);
const editing = ref<any>(null);
const deleting = ref<any>(null);

const formState = reactive({
  name: '',
  type: 'OPEN_API',
  description: '',
  domainsText: '',
  upstreamPathPrefix: '',
  dsn: '',
  rawConfigurations: '',
});

const filtered = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  const matchKeyword = !keyword || [item.name, item.type, item.description].some((value) => String(value || '').toLowerCase().includes(keyword));
  const matchType = !typeFilter.value || item.type === typeFilter.value;
  return matchKeyword && matchType;
}));

async function load() {
  loading.value = true;
  try {
    const result = await listMcpServers({ pageNum: 1, pageSize: 100 }).catch(() => ({ data: [] }));
    rows.value = Array.isArray(result) ? result : (result.data || []);
  } finally {
    loading.value = false;
  }
}

function openDrawer(record?: any) {
  editing.value = record || null;
  Object.assign(formState, {
    name: record?.name || '',
    type: record?.type || 'OPEN_API',
    description: record?.description || '',
    domainsText: joinLines(record?.domains),
    upstreamPathPrefix: record?.upstreamPathPrefix || '',
    dsn: record?.dsn || '',
    rawConfigurations: record?.rawConfigurations || '',
  });
  drawerOpen.value = true;
}

async function submit() {
  const payload = {
    ...editing.value,
    name: formState.name,
    type: formState.type,
    description: formState.description,
    domains: splitLines(formState.domainsText),
    upstreamPathPrefix: formState.upstreamPathPrefix || undefined,
    dsn: formState.dsn || undefined,
    rawConfigurations: formState.rawConfigurations || undefined,
  };
  await createOrUpdateMcpServer(payload as any);
  drawerOpen.value = false;
  await load();
  showSuccess('保存成功');
}

async function confirmDelete() {
  if (!deleting.value) {
    return;
  }
  await deleteMcpServer(deleting.value.name);
  deleteOpen.value = false;
  await load();
  showSuccess('删除成功');
}

onMounted(load);
</script>

<template>
  <PageSection title="MCP 配置">
    <ListToolbar v-model:search="search" search-placeholder="搜索名称或描述" create-text="创建 MCP 服务" @refresh="load" @create="openDrawer()">
      <template #left>
        <a-select v-model:value="typeFilter" allow-clear placeholder="类型" style="width: 180px">
          <a-select-option value="OPEN_API">OPEN_API</a-select-option>
          <a-select-option value="DATABASE">DATABASE</a-select-option>
          <a-select-option value="REDIRECT_ROUTE">REDIRECT_ROUTE</a-select-option>
        </a-select>
      </template>
    </ListToolbar>

    <a-table :data-source="filtered" :loading="loading" row-key="name">
      <a-table-column key="name" title="名称">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="router.push(`/mcp/detail/${encodeURIComponent(record.name)}`)">{{ record.name }}</a-button>
        </template>
      </a-table-column>
      <a-table-column key="description" data-index="description" title="描述" />
      <a-table-column key="type" data-index="type" title="类型" />
      <a-table-column key="actions" title="操作" width="220">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="router.push(`/mcp/detail/${encodeURIComponent(record.name)}`)">详情</a-button>
          <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          <a-button type="link" size="small" danger @click="deleting = record; deleteOpen = true">删除</a-button>
        </template>
      </a-table-column>
    </a-table>

    <a-drawer v-model:open="drawerOpen" width="720" :title="editing ? '编辑 MCP 服务' : '创建 MCP 服务'">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="formState.name" :disabled="Boolean(editing)" /></a-form-item>
        <a-form-item label="类型"><a-input v-model:value="formState.type" /></a-form-item>
        <a-form-item label="描述"><a-input v-model:value="formState.description" /></a-form-item>
        <a-form-item label="接入域名"><a-textarea v-model:value="formState.domainsText" :rows="4" /></a-form-item>
        <a-form-item label="路径前缀"><a-input v-model:value="formState.upstreamPathPrefix" /></a-form-item>
        <a-form-item label="DSN"><a-input v-model:value="formState.dsn" /></a-form-item>
        <a-form-item label="原始配置"><a-textarea v-model:value="formState.rawConfigurations" :rows="10" /></a-form-item>
      </a-form>
      <DrawerFooter @cancel="drawerOpen = false" @confirm="submit" />
    </a-drawer>

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.name} 吗？` : ''" @confirm="confirmDelete" />
  </PageSection>
</template>
