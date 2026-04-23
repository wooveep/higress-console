<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import { addTlsCertificate, deleteTlsCertificate, getTlsCertificates, updateTlsCertificate } from '@/services/tls-certificate';
import { joinLines, splitLines } from '@/lib/portal';
import { showSuccess } from '@/lib/feedback';

const loading = ref(false);
const search = ref('');
const rows = ref<any[]>([]);
const drawerOpen = ref(false);
const deleteOpen = ref(false);
const editing = ref<any>(null);
const deleting = ref<any>(null);

const formState = reactive({
  name: '',
  domainsText: '',
  cert: '',
  key: '',
});

const filtered = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, joinLines(item.domains)].some((value) => String(value || '').toLowerCase().includes(keyword));
}));

async function load() {
  loading.value = true;
  try {
    const result = await getTlsCertificates().catch(() => ({ data: [] }));
    rows.value = Array.isArray(result) ? result : (result.data || []);
  } finally {
    loading.value = false;
  }
}

function openDrawer(record?: any) {
  editing.value = record || null;
  Object.assign(formState, {
    name: record?.name || '',
    domainsText: joinLines(record?.domains),
    cert: record?.cert || '',
    key: record?.key || '',
  });
  drawerOpen.value = true;
}

async function submit() {
  const payload = {
    ...(editing.value?.version ? { version: editing.value.version } : {}),
    name: formState.name,
    cert: formState.cert,
    key: formState.key,
    domains: splitLines(formState.domainsText),
  };
  if (editing.value) {
    await updateTlsCertificate(payload as any);
  } else {
    await addTlsCertificate(payload as any);
  }
  drawerOpen.value = false;
  await load();
  showSuccess('保存成功');
}

async function confirmDelete() {
  if (!deleting.value) {
    return;
  }
  await deleteTlsCertificate(deleting.value.name);
  deleteOpen.value = false;
  await load();
  showSuccess('删除成功');
}

onMounted(load);
</script>

<template>
  <PageSection title="证书管理">
    <ListToolbar v-model:search="search" search-placeholder="搜索证书名或域名" create-text="新增证书" @refresh="load" @create="openDrawer()" />
    <a-table :data-source="filtered" :loading="loading" row-key="name" :scroll="{ x: 900 }">
      <a-table-column key="name" data-index="name" title="名称" />
      <a-table-column key="domains" title="关联域名">
        <template #default="{ record }">{{ joinLines(record.domains) || '-' }}</template>
      </a-table-column>
      <a-table-column key="validityStart" data-index="validityStart" title="生效时间" />
      <a-table-column key="validityEnd" data-index="validityEnd" title="失效时间" />
      <a-table-column key="actions" title="操作" width="180">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          <a-button type="link" size="small" danger @click="deleting = record; deleteOpen = true">删除</a-button>
        </template>
      </a-table-column>
    </a-table>

    <a-drawer v-model:open="drawerOpen" width="720" :title="editing ? '编辑证书' : '新增证书'">
      <a-form layout="vertical">
        <a-form-item label="名称"><a-input v-model:value="formState.name" :disabled="Boolean(editing)" /></a-form-item>
        <a-form-item label="关联域名"><a-textarea v-model:value="formState.domainsText" :rows="4" /></a-form-item>
        <a-form-item label="证书内容"><a-textarea v-model:value="formState.cert" :rows="10" /></a-form-item>
        <a-form-item label="私钥"><a-textarea v-model:value="formState.key" :rows="10" /></a-form-item>
      </a-form>
      <DrawerFooter @cancel="drawerOpen = false" @confirm="submit" />
    </a-drawer>

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.name} 吗？` : ''" @confirm="confirmDelete" />
  </PageSection>
</template>
