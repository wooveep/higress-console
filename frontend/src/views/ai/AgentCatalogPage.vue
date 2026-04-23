<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import PageSection from '@/components/common/PageSection.vue';
import PortalUnavailableState from '@/components/common/PortalUnavailableState.vue';
import { usePortalAvailability } from '@/composables/usePortalAvailability';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import StatusTag from '@/components/common/StatusTag.vue';
import { showSuccess } from '@/lib/feedback';
import {
  createAgentCatalog,
  getAgentCatalogs,
  getAgentCatalogOptions,
  publishAgentCatalog,
  unpublishAgentCatalog,
  updateAgentCatalog,
} from '@/services/agent-catalog';
import { joinLines, splitLines } from '@/lib/portal';

const { t } = useI18n();
const { portalUnavailable } = usePortalAvailability();
const loading = ref(false);
const search = ref('');
const rows = ref<any[]>([]);
const drawerOpen = ref(false);
const editing = ref<any>(null);
const deleteOpen = ref(false);
const deleting = ref<any>(null);
const options = ref<any[]>([]);
const optionsLoadFailed = ref(false);

const formState = reactive({
  agentId: '',
  canonicalName: '',
  displayName: '',
  intro: '',
  description: '',
  iconUrl: '',
  tagsText: '',
  mcpServerName: '',
});

const filtered = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.agentId, item.displayName, item.canonicalName, item.mcpServerName].some((value) => String(value || '').toLowerCase().includes(keyword));
}));

async function load() {
  if (portalUnavailable.value) {
    rows.value = [];
    options.value = [];
    optionsLoadFailed.value = false;
    loading.value = false;
    return;
  }
  loading.value = true;
  try {
    const [catalogs, opts] = await Promise.all([
      getAgentCatalogs().catch(() => []),
      getAgentCatalogOptions().catch(() => {
        optionsLoadFailed.value = true;
        return { servers: [] };
      }),
    ]);
    rows.value = catalogs || [];
    options.value = opts?.servers || [];
    if (opts?.servers) {
      optionsLoadFailed.value = false;
    }
  } finally {
    loading.value = false;
  }
}

function openDrawer(record?: any) {
  editing.value = record || null;
  Object.assign(formState, {
    agentId: record?.agentId || '',
    canonicalName: record?.canonicalName || '',
    displayName: record?.displayName || '',
    intro: record?.intro || '',
    description: record?.description || '',
    iconUrl: record?.iconUrl || '',
    tagsText: joinLines(record?.tags),
    mcpServerName: record?.mcpServerName || '',
  });
  drawerOpen.value = true;
}

async function submit() {
  const payload = {
    ...editing.value,
    agentId: formState.agentId,
    canonicalName: formState.canonicalName,
    displayName: formState.displayName,
    intro: formState.intro,
    description: formState.description,
    iconUrl: formState.iconUrl,
    tags: splitLines(formState.tagsText),
    mcpServerName: formState.mcpServerName || undefined,
  };
  if (editing.value) {
    await updateAgentCatalog(editing.value.agentId, payload as any);
  } else {
    await createAgentCatalog(payload as any);
  }
  drawerOpen.value = false;
  await load();
  showSuccess('保存成功');
}

async function toggleStatus(record: any) {
  if (record.status === 'published') {
    await unpublishAgentCatalog(record.agentId);
  } else {
    await publishAgentCatalog(record.agentId);
  }
  await load();
  showSuccess('状态已更新');
}

onMounted(load);
</script>

<template>
  <PageSection title="智能体目录管理">
    <PortalUnavailableState v-if="portalUnavailable" />
    <template v-else>
      <ListToolbar v-model:search="search" search-placeholder="搜索 Agent ID、显示名、MCP 服务" create-text="创建智能体目录" @refresh="load" @create="openDrawer()" />
      <a-alert
        v-if="optionsLoadFailed"
        class="agent-catalog-page__alert"
        type="warning"
        show-icon
        :message="t('agentCatalog.optionsLoadFailed')"
      />
      <a-table :data-source="filtered" :loading="loading" row-key="agentId" :scroll="{ x: 1100 }">
        <a-table-column key="agentId" data-index="agentId" title="Agent ID" />
        <a-table-column key="displayName" data-index="displayName" title="显示名" />
        <a-table-column key="mcpServerName" data-index="mcpServerName" title="MCP 服务" />
        <a-table-column key="status" title="状态">
          <template #default="{ record }"><StatusTag :value="record.status" /></template>
        </a-table-column>
        <a-table-column key="publishedAt" data-index="publishedAt" title="发布时间" />
        <a-table-column key="actions" title="操作" width="240">
          <template #default="{ record }">
            <a-button type="link" size="small" @click="toggleStatus(record)">{{ record.status === 'published' ? '下线' : '发布' }}</a-button>
            <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          </template>
        </a-table-column>
      </a-table>
    </template>

    <a-drawer v-model:open="drawerOpen" width="760" :title="editing ? '编辑智能体目录' : '创建智能体目录'">
      <a-form layout="vertical">
        <a-form-item label="Agent ID"><a-input v-model:value="formState.agentId" :disabled="Boolean(editing)" /></a-form-item>
        <a-form-item label="canonicalName"><a-input v-model:value="formState.canonicalName" /></a-form-item>
        <a-form-item label="displayName"><a-input v-model:value="formState.displayName" /></a-form-item>
        <a-form-item label="简介"><a-input v-model:value="formState.intro" /></a-form-item>
        <a-form-item label="描述"><a-textarea v-model:value="formState.description" :rows="4" /></a-form-item>
        <a-form-item label="图标 URL"><a-input v-model:value="formState.iconUrl" /></a-form-item>
        <a-form-item label="Tags（一行一个）"><a-textarea v-model:value="formState.tagsText" :rows="4" /></a-form-item>
        <a-form-item label="MCP 服务">
          <a-select v-model:value="formState.mcpServerName" allow-clear>
            <a-select-option v-for="item in options" :key="item.mcpServerName" :value="item.mcpServerName">
              {{ item.mcpServerName }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
      <DrawerFooter @cancel="drawerOpen = false" @confirm="submit" />
    </a-drawer>

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.agentId} 吗？` : ''" />
  </PageSection>
</template>

<style scoped>
.agent-catalog-page__alert {
  margin-bottom: 16px;
}
</style>
