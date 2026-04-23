<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import StrategyLink from '@/components/common/StrategyLink.vue';
import { addGatewayDomain, deleteGatewayDomain, getGatewayDomains, updateGatewayDomain } from '@/services/domain';
import { showSuccess } from '@/lib/feedback';

const DEFAULT_DOMAIN = 'aigateway-default-domain';
const router = useRouter();
const loading = ref(false);
const search = ref('');
const drawerOpen = ref(false);
const deleteOpen = ref(false);
const editing = ref<any>(null);
const deleting = ref<any>(null);
const rows = ref<any[]>([]);

const formState = reactive({
  name: '',
  enableHttps: 'off',
  certIdentifier: '',
});

const filtered = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, item.certIdentifier, item.protocol].some((value) => String(value || '').toLowerCase().includes(keyword));
}));

async function load() {
  loading.value = true;
  try {
    const result = await getGatewayDomains().catch(() => []);
    rows.value = (result || []).map((item: any) => ({
      ...item,
      protocol: item.enableHttps === 'off' ? 'HTTP' : 'HTTPS',
    }));
  } finally {
    loading.value = false;
  }
}

function openDrawer(record?: any) {
  editing.value = record || null;
  Object.assign(formState, {
    name: record?.name || '',
    enableHttps: record?.enableHttps || 'off',
    certIdentifier: record?.certIdentifier || '',
  });
  drawerOpen.value = true;
}

async function submit() {
  const payload = {
    ...(editing.value?.version ? { version: editing.value.version } : {}),
    name: editing.value?.name || formState.name,
    enableHttps: formState.enableHttps,
    certIdentifier: formState.certIdentifier || undefined,
  };
  if (editing.value) {
    await updateGatewayDomain(payload as any);
  } else {
    await addGatewayDomain(payload as any);
  }
  drawerOpen.value = false;
  await load();
  showSuccess('保存成功');
}

async function confirmDelete() {
  if (!deleting.value) {
    return;
  }
  await deleteGatewayDomain(deleting.value.name);
  deleteOpen.value = false;
  await load();
  showSuccess('删除成功');
}

onMounted(load);
</script>

<template>
  <PageSection title="域名管理">
    <ListToolbar v-model:search="search" search-placeholder="搜索域名或证书" create-text="新增域名" @refresh="load" @create="openDrawer()" />
    <a-table :data-source="filtered" :loading="loading" row-key="name">
      <a-table-column key="name" title="域名">
        <template #default="{ record }">{{ record.name === DEFAULT_DOMAIN ? '默认域名' : record.name }}</template>
      </a-table-column>
      <a-table-column key="protocol" data-index="protocol" title="协议" />
      <a-table-column key="certIdentifier" data-index="certIdentifier" title="证书" />
      <a-table-column key="actions" title="操作" width="220">
        <template #default="{ record }">
          <StrategyLink v-if="record.name !== DEFAULT_DOMAIN" :path="`/domain/config?type=domain&name=${encodeURIComponent(record.name)}`" />
          <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          <a-button
            v-if="record.name !== DEFAULT_DOMAIN"
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

    <a-drawer v-model:open="drawerOpen" width="520" :title="editing ? '编辑域名' : '新增域名'">
      <a-form layout="vertical">
        <a-form-item label="域名"><a-input v-model:value="formState.name" :disabled="Boolean(editing)" /></a-form-item>
        <a-form-item label="HTTPS">
          <a-select v-model:value="formState.enableHttps">
            <a-select-option value="off">关闭</a-select-option>
            <a-select-option value="on">开启</a-select-option>
            <a-select-option value="force">强制 HTTPS</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="证书标识"><a-input v-model:value="formState.certIdentifier" /></a-form-item>
      </a-form>
      <DrawerFooter @cancel="drawerOpen = false" @confirm="submit" />
    </a-drawer>

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.name} 吗？` : ''" @confirm="confirmDelete" />
  </PageSection>
</template>
