<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import StatusTag from '@/components/common/StatusTag.vue';
import { showConfirm, showError, showSuccess } from '@/lib/feedback';
import {
  createWasmPlugin,
  deleteWasmPlugin,
  deleteDomainPluginInstance,
  deleteGlobalPluginInstance,
  deleteRoutePluginInstance,
  deleteServicePluginInstance,
  getDomainPluginInstance,
  getDomainPluginInstances,
  getGlobalPluginInstance,
  getRoutePluginInstance,
  getRoutePluginInstances,
  getServicePluginInstance,
  getServicePluginInstances,
  getWasmPluginReadme,
  getWasmPlugins,
  getWasmPluginsConfig,
  updateDomainPluginInstance,
  updateGlobalPluginInstance,
  updateRoutePluginInstance,
  updateServicePluginInstance,
  updateWasmPlugin,
} from '@/services/plugin';
import { getAiRoute, updateAiRoute } from '@/services/ai-route';
import { getGatewayRouteDetail, updateRouteConfig } from '@/services/route';
import { BUILTIN_ROUTE_PLUGIN_LIST } from '@/plugins/constants';
import {
  QueryType,
  filterVisiblePlugins,
  resolvePluginVisibilityScope,
} from '@/plugins/visibility';

const WasmPluginDrawer = defineAsyncComponent(() => import('@/features/plugin/WasmPluginDrawer.vue'));
const PluginConfigDrawer = defineAsyncComponent(() => import('@/features/plugin/PluginConfigDrawer.vue'));

const route = useRoute();
const router = useRouter();
const { locale } = useI18n();

const loading = ref(false);
const search = ref('');
const rows = ref<any[]>([]);
const targetDetail = ref<any>(null);

const wasmDrawerOpen = ref(false);
const configDrawerOpen = ref(false);
const deleteOpen = ref(false);
const configLoading = ref(false);
const instanceLoading = ref(false);
const detailLoading = ref(false);
const deletingBinding = ref(false);

const editingWasm = ref<any>(null);
const deleting = ref<any>(null);
const configuring = ref<any>(null);
const currentConfigData = ref<any>(null);
const currentInstanceData = ref<any>(null);
const selectedPluginName = ref('');
const selectedPluginSchema = ref<any>(null);
const selectedPluginReadme = ref('');

const queryType = computed(() => String(route.query.type || ''));
const queryName = computed(() => String(route.query.name || ''));
const isTargetMode = computed(() => Boolean(queryType.value && queryName.value));
const visibilityScope = computed(() => resolvePluginVisibilityScope(queryType.value));
const selectedPlugin = computed(() => rows.value.find((item) => item.name === selectedPluginName.value) || null);
const selectedPluginSchemaText = computed(() => (
  selectedPluginSchema.value ? JSON.stringify(selectedPluginSchema.value, null, 2) : ''
));
const canDeleteCurrentBinding = computed(() => (
  Boolean(configuring.value && !configuring.value.builtIn && currentInstanceData.value)
));

const backPath = computed(() => {
  if (queryType.value === QueryType.ROUTE) return '/route';
  if (queryType.value === QueryType.DOMAIN) return '/domain';
  if (queryType.value === QueryType.SERVICE) return '/service';
  if (queryType.value === QueryType.AI_ROUTE) return '/ai/route';
  return '';
});

const filteredRows = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, item.title, item.category, item.description]
    .some((value) => String(value || '').toLowerCase().includes(keyword));
}));

function isBuiltInPlugin(name: string) {
  return BUILTIN_ROUTE_PLUGIN_LIST.some((item) => item.key === name);
}

function getBuiltInPluginReadme(record: any) {
  const detailByName: Record<string, string> = {
    rewrite: '内置路由能力，用于修改请求的 Host 与 Path。请通过“配置”直接编辑路由上的重写规则。',
    headerModify: '内置路由能力，用于修改请求头和响应头。AI 路由场景下会直接写入路由的 Header 配置，不依赖 wasm 插件资源。',
    cors: '内置路由能力，用于配置跨域访问规则。请通过“配置”直接编辑路由上的跨域策略。',
    retries: '内置路由能力，用于配置后端重试策略。请通过“配置”直接编辑路由上的重试规则。',
  };

  return detailByName[record.name]
    || record.description
    || '内置路由能力，请通过“配置”直接编辑该策略。';
}

function getBuiltInEnabled(name: string) {
  if (name === 'rewrite') {
    return Boolean(targetDetail.value?.rewrite?.enabled);
  }
  if (name === 'headerModify') {
    return Boolean(targetDetail.value?.headerModify?.enabled || targetDetail.value?.headerControl?.enabled);
  }
  if (name === 'cors') {
    return Boolean(targetDetail.value?.cors?.enabled);
  }
  return Boolean(targetDetail.value?.retries?.enabled || targetDetail.value?.proxyNextUpstream?.enabled);
}

function getBuiltInRows() {
  if (queryType.value !== QueryType.ROUTE && queryType.value !== QueryType.AI_ROUTE) {
    return [];
  }
  const list = queryType.value === QueryType.AI_ROUTE
    ? BUILTIN_ROUTE_PLUGIN_LIST.filter((item) => item.enabledInAiRoute !== false)
    : BUILTIN_ROUTE_PLUGIN_LIST;

  return list.map((item) => ({
    ...item,
    name: item.key,
    enabled: getBuiltInEnabled(item.key),
    boundStatus: getBuiltInEnabled(item.key) ? '已绑定' : '未绑定',
  }));
}

async function loadTargetDetail() {
  if (!isTargetMode.value) {
    targetDetail.value = null;
    return;
  }
  if (queryType.value === QueryType.DOMAIN) {
    targetDetail.value = { name: queryName.value };
    return;
  }
  if (queryType.value === QueryType.AI_ROUTE) {
    targetDetail.value = await getAiRoute(queryName.value).catch(() => null);
    return;
  }
  targetDetail.value = await getGatewayRouteDetail(queryName.value).catch(() => null);
}

function getPluginTargetName() {
  if (queryType.value === QueryType.AI_ROUTE) {
    return `ai-route-${queryName.value}.internal`;
  }
  return queryName.value;
}

async function load() {
  loading.value = true;
  try {
    await loadTargetDetail();
    const plugins = await getWasmPlugins(locale.value).catch(() => []);
    const visiblePlugins = filterVisiblePlugins(plugins || [], visibilityScope.value);
    let merged = visiblePlugins;

    if (isTargetMode.value) {
      let enabledList: any[] = [];
      if (queryType.value === QueryType.DOMAIN) {
        enabledList = await getDomainPluginInstances(queryName.value).catch(() => []);
      } else if (queryType.value === QueryType.SERVICE) {
        enabledList = await getServicePluginInstances(queryName.value).catch(() => []);
      } else {
        enabledList = await getRoutePluginInstances(getPluginTargetName()).catch(() => []);
      }

      merged = visiblePlugins.map((item: any) => {
        const enabledInstance = enabledList.find((plugin: any) => plugin.pluginName === item.name);
        return {
          ...item,
          enabled: Boolean(enabledInstance?.enabled),
          boundStatus: enabledInstance ? (enabledInstance.enabled ? '已绑定' : '已创建') : '未绑定',
        };
      });
      merged = [...getBuiltInRows(), ...merged];
    } else {
      merged = visiblePlugins.map((item: any) => ({
        ...item,
        boundStatus: item.internal ? '内置' : '可配置',
      }));
    }

    rows.value = merged;
    await syncSelectedPlugin(merged);
  } finally {
    loading.value = false;
  }
}

async function syncSelectedPlugin(nextRows: any[]) {
  const nextSelected = nextRows.find((item) => item.name === selectedPluginName.value) || nextRows[0] || null;
  selectedPluginName.value = nextSelected?.name || '';
  await loadPluginDetail(nextSelected);
}

async function loadPluginDetail(record: any | null) {
  selectedPluginSchema.value = null;
  selectedPluginReadme.value = '';
  if (!record) {
    return;
  }

  detailLoading.value = true;
  try {
    if (record.builtIn || isBuiltInPlugin(record.name)) {
      selectedPluginReadme.value = getBuiltInPluginReadme(record);
      return;
    }

    const [configData, readme] = await Promise.all([
      getWasmPluginsConfig(record.name).catch(() => null),
      getWasmPluginReadme(record.name).catch(() => ''),
    ]);
    selectedPluginSchema.value = configData?.schema?.jsonSchema || null;
    selectedPluginReadme.value = typeof readme === 'string' ? readme : '';
  } finally {
    detailLoading.value = false;
  }
}

function openWasmDrawer(record?: any) {
  editingWasm.value = record || null;
  wasmDrawerOpen.value = true;
}

function openPluginDetail(record: any) {
  selectedPluginName.value = record.name;
  void loadPluginDetail(record);
}

async function submitWasm(payload: any, isEdit: boolean) {
  if (isEdit && editingWasm.value) {
    await updateWasmPlugin(editingWasm.value.name, payload);
  } else {
    await createWasmPlugin(payload);
  }
  wasmDrawerOpen.value = false;
  await load();
  showSuccess('插件已保存');
}

async function openConfig(record: any) {
  configuring.value = { ...record, queryType: queryType.value };
  currentConfigData.value = null;
  currentInstanceData.value = null;
  configDrawerOpen.value = true;

  if (record.builtIn) {
    return;
  }

  instanceLoading.value = true;
  configLoading.value = true;
  try {
    const [instanceData, configData] = await Promise.all([
      loadPluginInstance(record),
      getWasmPluginsConfig(record.name).catch(() => null),
    ]);
    currentInstanceData.value = instanceData;
    currentConfigData.value = configData;
  } finally {
    instanceLoading.value = false;
    configLoading.value = false;
  }
}

async function loadPluginInstance(record: any) {
  try {
    if (!isTargetMode.value) {
      return await getGlobalPluginInstance(record.name);
    }
    if (queryType.value === QueryType.DOMAIN) {
      return await getDomainPluginInstance({ name: queryName.value, pluginName: record.name });
    }
    if (queryType.value === QueryType.SERVICE) {
      return await getServicePluginInstance({ name: queryName.value, pluginName: record.name });
    }
    return await getRoutePluginInstance({ name: getPluginTargetName(), pluginName: record.name });
  } catch {
    return null;
  }
}

async function submitBuiltIn(payload: Record<string, any>) {
  if (!configuring.value) {
    return;
  }

  const nextPayload = {
    ...(targetDetail.value || {}),
    ...payload,
  };

  if (queryType.value === QueryType.AI_ROUTE) {
    await updateAiRoute(nextPayload);
  } else {
    await updateRouteConfig(targetDetail.value.name, nextPayload);
  }

  configDrawerOpen.value = false;
  await load();
  showSuccess('配置已保存');
}

async function submitPlugin(payload: { enabled: boolean; rawConfigurations: string }) {
  if (!configuring.value) {
    return;
  }

  const nextPayload = {
    enabled: payload.enabled,
    pluginName: configuring.value.name,
    rawConfigurations: payload.rawConfigurations,
  };

  if (!isTargetMode.value) {
    await updateGlobalPluginInstance(configuring.value.name, nextPayload);
  } else if (queryType.value === QueryType.DOMAIN) {
    await updateDomainPluginInstance({ name: queryName.value, pluginName: configuring.value.name }, nextPayload);
  } else if (queryType.value === QueryType.SERVICE) {
    await updateServicePluginInstance({ name: queryName.value, pluginName: configuring.value.name }, nextPayload);
  } else {
    await updateRoutePluginInstance({ name: getPluginTargetName(), pluginName: configuring.value.name }, nextPayload);
  }

  configDrawerOpen.value = false;
  await load();
  showSuccess('配置已保存');
}

async function deleteConfiguredBinding() {
  if (!configuring.value) {
    return;
  }

  deletingBinding.value = true;
  try {
    if (!isTargetMode.value) {
      await deleteGlobalPluginInstance(configuring.value.name);
    } else if (queryType.value === QueryType.DOMAIN) {
      await deleteDomainPluginInstance({ name: queryName.value, pluginName: configuring.value.name });
    } else if (queryType.value === QueryType.SERVICE) {
      await deleteServicePluginInstance({ name: queryName.value, pluginName: configuring.value.name });
    } else {
      await deleteRoutePluginInstance({ name: getPluginTargetName(), pluginName: configuring.value.name });
    }
    configDrawerOpen.value = false;
    await load();
    showSuccess('插件绑定已删除');
  } finally {
    deletingBinding.value = false;
  }
}

function requestDeleteConfiguredBinding() {
  if (!configuring.value) {
    return;
  }
  showConfirm({
    title: '删除当前绑定',
    content: `确定删除 ${configuring.value.name} 的当前绑定配置吗？`,
    okText: '删除',
    okType: 'danger',
    onOk: deleteConfiguredBinding,
  });
}

async function confirmDelete() {
  if (!deleting.value) {
    return;
  }
  await deleteWasmPlugin(deleting.value.name);
  deleteOpen.value = false;
  await load();
  showSuccess('插件已删除');
}

watch(() => [route.fullPath, locale.value], () => {
  void load();
});
onMounted(load);
</script>

<template>
  <div class="plugin-page">
    <PageSection :title="isTargetMode ? `插件配置 · ${queryType} / ${queryName}` : '插件配置'">
      <ListToolbar
        v-model:search="search"
        search-placeholder="搜索插件名、标题、分类"
        :create-text="isTargetMode ? '' : '新增插件'"
        @refresh="load"
        @create="openWasmDrawer()"
      >
        <template #left>
          <a-button v-if="backPath" @click="router.push(backPath)">返回</a-button>
        </template>
      </ListToolbar>

      <a-table :data-source="filteredRows" :loading="loading" row-key="name" :scroll="{ x: 1240 }">
        <a-table-column key="name" data-index="name" title="插件名" width="220" />
        <a-table-column key="title" data-index="title" title="标题" width="180" />
        <a-table-column key="category" data-index="category" title="分类" width="120" />
        <a-table-column key="enabled" title="状态" width="120">
          <template #default="{ record }">
            <StatusTag :value="record.enabled ? 'enabled' : 'disabled'" />
          </template>
        </a-table-column>
        <a-table-column key="boundStatus" data-index="boundStatus" title="绑定状态" width="120" />
        <a-table-column key="description" data-index="description" title="描述" />
        <a-table-column key="actions" title="操作" width="320" fixed="right">
          <template #default="{ record }">
            <a-button type="link" size="small" @click="openPluginDetail(record)">详情</a-button>
            <a-button type="link" size="small" @click="openConfig(record)">配置</a-button>
            <a-button
              v-if="!record.builtIn && !isTargetMode"
              type="link"
              size="small"
              @click="openWasmDrawer(record)"
            >
              编辑
            </a-button>
            <a-button
              v-if="!record.builtIn && !isTargetMode"
              type="link"
              size="small"
              danger
              @click="deleting = record; deleteOpen = true"
            >
              删除
            </a-button>
          </template>
        </a-table-column>
      </a-table>
    </PageSection>

    <PageSection v-if="selectedPlugin" :title="`插件详情 · ${selectedPlugin.title || selectedPlugin.name}`">
      <a-skeleton :loading="detailLoading" active>
        <div class="plugin-page__detail-grid">
          <a-descriptions bordered size="small" :column="2">
            <a-descriptions-item label="插件名">{{ selectedPlugin.name }}</a-descriptions-item>
            <a-descriptions-item label="标题">{{ selectedPlugin.title || '-' }}</a-descriptions-item>
            <a-descriptions-item label="分类">{{ selectedPlugin.category || '-' }}</a-descriptions-item>
            <a-descriptions-item label="绑定状态">{{ selectedPlugin.boundStatus || '-' }}</a-descriptions-item>
            <a-descriptions-item label="启用状态">
              <StatusTag :value="selectedPlugin.enabled ? 'enabled' : 'disabled'" />
            </a-descriptions-item>
            <a-descriptions-item label="作用域">{{ visibilityScope }}</a-descriptions-item>
            <a-descriptions-item label="描述" :span="2">{{ selectedPlugin.description || '-' }}</a-descriptions-item>
          </a-descriptions>

          <a-tabs>
            <a-tab-pane key="schema" tab="配置 Schema">
              <a-empty v-if="!selectedPluginSchemaText" description="当前插件未提供可展示的 Schema。" />
              <pre v-else class="plugin-page__pre">{{ selectedPluginSchemaText }}</pre>
            </a-tab-pane>
            <a-tab-pane key="readme" tab="README">
              <a-empty v-if="!selectedPluginReadme" description="当前插件未提供 README。" />
              <pre v-else class="plugin-page__pre plugin-page__readme">{{ selectedPluginReadme }}</pre>
            </a-tab-pane>
          </a-tabs>
        </div>
      </a-skeleton>
    </PageSection>

    <WasmPluginDrawer
      v-model:open="wasmDrawerOpen"
      :record="editingWasm"
      @submit="submitWasm"
    />

    <PluginConfigDrawer
      v-model:open="configDrawerOpen"
      :record="configuring"
      :target-detail="targetDetail"
      :loading="configLoading"
      :instance-loading="instanceLoading"
      :deleting="deletingBinding"
      :allow-delete="canDeleteCurrentBinding"
      :config-data="currentConfigData"
      :instance-data="currentInstanceData"
      @submit-built-in="submitBuiltIn"
      @submit-plugin="submitPlugin"
      @delete-plugin="requestDeleteConfiguredBinding"
    />

    <DeleteConfirmModal
      v-model:open="deleteOpen"
      title="删除插件"
      :content="deleting ? `确定删除插件 ${deleting.name} 吗？` : ''"
      @confirm="confirmDelete"
    />
  </div>
</template>

<style scoped>
.plugin-page {
  display: grid;
  gap: 18px;
}

.plugin-page__detail-grid {
  display: grid;
  gap: 18px;
}

.plugin-page__pre {
  margin: 0;
  padding: 16px;
  border: 1px solid var(--portal-border);
  border-radius: 12px;
  background: var(--portal-surface-soft);
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.6;
}

.plugin-page__readme {
  min-height: 240px;
}
</style>
