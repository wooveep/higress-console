<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import {
  addServiceSource,
  deleteServiceSource,
  getServiceSources,
  updateServiceSource,
} from '@/services/service-source';
import {
  addProxyServer,
  deleteProxyServer,
  getProxyServers,
  updateProxyServer,
} from '@/services/proxy-server';
import { safeParseJson, stringifyPretty } from '@/lib/portal';
import { showSuccess } from '@/lib/feedback';

const loading = ref(false);
const proxyLoading = ref(false);
const search = ref('');
const deleteOpen = ref(false);
const deleteTarget = ref<any>(null);
const activeTab = ref('sources');
const sourceDrawerOpen = ref(false);
const proxyDrawerOpen = ref(false);
const editingSource = ref<any>(null);
const editingProxy = ref<any>(null);
const sources = ref<any[]>([]);
const proxies = ref<any[]>([]);

const sourceForm = reactive({
  name: '',
  type: 'static',
  domain: '',
  port: 80,
  protocol: 'http',
  proxyName: '',
  propertiesJson: '{}',
});

const proxyForm = reactive({
  name: '',
  type: 'http',
  serverAddress: '',
  serverPort: 8080,
  connectTimeout: 5000,
});

const filteredSources = computed(() => sources.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, item.type, item.domain, item.proxyName].some((value) => String(value || '').toLowerCase().includes(keyword));
}));

const filteredProxies = computed(() => proxies.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, item.type, item.serverAddress].some((value) => String(value || '').toLowerCase().includes(keyword));
}));

async function loadSources() {
  loading.value = true;
  try {
    sources.value = await getServiceSources({} as any).catch(() => []);
  } finally {
    loading.value = false;
  }
}

async function loadProxies() {
  proxyLoading.value = true;
  try {
    proxies.value = await getProxyServers().catch(() => []);
  } finally {
    proxyLoading.value = false;
  }
}

async function loadAll() {
  await Promise.all([loadSources(), loadProxies()]);
}

function openSourceDrawer(record?: any) {
  editingSource.value = record || null;
  Object.assign(sourceForm, {
    name: record?.name || '',
    type: record?.type || 'static',
    domain: record?.domain || '',
    port: record?.port || 80,
    protocol: record?.protocol || 'http',
    proxyName: record?.proxyName || '',
    propertiesJson: stringifyPretty(record?.properties || {}),
  });
  sourceDrawerOpen.value = true;
}

function openProxyDrawer(record?: any) {
  editingProxy.value = record || null;
  Object.assign(proxyForm, {
    name: record?.name || '',
    type: record?.type || 'http',
    serverAddress: record?.serverAddress || '',
    serverPort: record?.serverPort || 8080,
    connectTimeout: record?.connectTimeout || 5000,
  });
  proxyDrawerOpen.value = true;
}

async function submitSource() {
  const payload = {
    ...(editingSource.value?.version ? { version: editingSource.value.version } : {}),
    name: sourceForm.name,
    type: sourceForm.type,
    domain: sourceForm.domain,
    port: sourceForm.port,
    protocol: sourceForm.protocol,
    proxyName: sourceForm.proxyName || undefined,
    properties: safeParseJson(sourceForm.propertiesJson, {}),
  };
  if (editingSource.value) {
    await updateServiceSource(payload as any);
  } else {
    await addServiceSource(payload as any);
  }
  sourceDrawerOpen.value = false;
  await loadSources();
  showSuccess('保存成功');
}

async function submitProxy() {
  const payload = {
    ...(editingProxy.value?.version ? { version: editingProxy.value.version } : {}),
    name: proxyForm.name,
    type: proxyForm.type,
    serverAddress: proxyForm.serverAddress,
    serverPort: proxyForm.serverPort,
    connectTimeout: proxyForm.connectTimeout,
  };
  if (editingProxy.value) {
    await updateProxyServer(payload as any);
  } else {
    await addProxyServer(payload as any);
  }
  proxyDrawerOpen.value = false;
  await loadProxies();
  showSuccess('保存成功');
}

function requestDelete(record: any, type: 'source' | 'proxy') {
  deleteTarget.value = { ...record, entityType: type };
  deleteOpen.value = true;
}

async function confirmDelete() {
  if (!deleteTarget.value) {
    return;
  }
  if (deleteTarget.value.entityType === 'proxy') {
    await deleteProxyServer(deleteTarget.value.name);
    await loadProxies();
  } else {
    await deleteServiceSource(deleteTarget.value.name);
    await loadSources();
  }
  deleteOpen.value = false;
  showSuccess('删除成功');
}

onMounted(loadAll);
</script>

<template>
  <PageSection title="服务来源">
    <ListToolbar
      v-model:search="search"
      search-placeholder="搜索名称、类型、域名"
      :create-text="activeTab === 'sources' ? '新增服务来源' : '新增代理服务'"
      @refresh="loadAll"
      @create="activeTab === 'sources' ? openSourceDrawer() : openProxyDrawer()"
    />

    <a-tabs v-model:activeKey="activeTab">
      <a-tab-pane key="sources" tab="服务来源">
        <a-table :data-source="filteredSources" :loading="loading" row-key="name" :scroll="{ x: 960 }">
          <a-table-column key="type" data-index="type" title="类型" />
          <a-table-column key="name" data-index="name" title="名称" />
          <a-table-column key="domain" data-index="domain" title="域名" />
          <a-table-column key="port" data-index="port" title="端口" />
          <a-table-column key="protocol" data-index="protocol" title="协议" />
          <a-table-column key="proxyName" data-index="proxyName" title="代理服务" />
          <a-table-column key="actions" title="操作" width="180">
            <template #default="{ record }">
              <a-button type="link" size="small" @click="openSourceDrawer(record)">编辑</a-button>
              <a-button type="link" size="small" danger @click="requestDelete(record, 'source')">删除</a-button>
            </template>
          </a-table-column>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="proxies" tab="代理服务">
        <a-table :data-source="filteredProxies" :loading="proxyLoading" row-key="name">
          <a-table-column key="name" data-index="name" title="名称" />
          <a-table-column key="type" data-index="type" title="类型" />
          <a-table-column key="serverAddress" data-index="serverAddress" title="服务地址" />
          <a-table-column key="serverPort" data-index="serverPort" title="端口" />
          <a-table-column key="connectTimeout" data-index="connectTimeout" title="超时(ms)" />
          <a-table-column key="actions" title="操作" width="180">
            <template #default="{ record }">
              <a-button type="link" size="small" @click="openProxyDrawer(record)">编辑</a-button>
              <a-button type="link" size="small" danger @click="requestDelete(record, 'proxy')">删除</a-button>
            </template>
          </a-table-column>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <a-drawer v-model:open="sourceDrawerOpen" width="640" :title="editingSource ? '编辑服务来源' : '新增服务来源'">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="sourceForm.name" /></a-form-item>
        <a-form-item label="类型"><a-input v-model:value="sourceForm.type" /></a-form-item>
        <a-form-item label="域名"><a-input v-model:value="sourceForm.domain" /></a-form-item>
        <a-form-item label="端口"><a-input-number v-model:value="sourceForm.port" style="width: 100%" /></a-form-item>
        <a-form-item label="协议"><a-input v-model:value="sourceForm.protocol" /></a-form-item>
        <a-form-item label="代理服务"><a-input v-model:value="sourceForm.proxyName" /></a-form-item>
        <a-form-item label="扩展属性(JSON)"><a-textarea v-model:value="sourceForm.propertiesJson" :rows="10" /></a-form-item>
      </a-form>
      <DrawerFooter @cancel="sourceDrawerOpen = false" @confirm="submitSource" />
    </a-drawer>

    <a-drawer v-model:open="proxyDrawerOpen" width="520" :title="editingProxy ? '编辑代理服务' : '新增代理服务'">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="proxyForm.name" /></a-form-item>
        <a-form-item label="类型"><a-input v-model:value="proxyForm.type" /></a-form-item>
        <a-form-item label="服务地址"><a-input v-model:value="proxyForm.serverAddress" /></a-form-item>
        <a-form-item label="服务端口"><a-input-number v-model:value="proxyForm.serverPort" style="width: 100%" /></a-form-item>
        <a-form-item label="连接超时(ms)"><a-input-number v-model:value="proxyForm.connectTimeout" style="width: 100%" /></a-form-item>
      </a-form>
      <DrawerFooter @cancel="proxyDrawerOpen = false" @confirm="submitProxy" />
    </a-drawer>

    <DeleteConfirmModal
      v-model:open="deleteOpen"
      :content="deleteTarget ? `确认删除 ${deleteTarget.name} 吗？` : ''"
      @confirm="confirmDelete"
    />
  </PageSection>
</template>
