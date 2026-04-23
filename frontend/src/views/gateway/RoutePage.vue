<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import DeleteConfirmModal from '@/components/common/DeleteConfirmModal.vue';
import StrategyLink from '@/components/common/StrategyLink.vue';
import StatusTag from '@/components/common/StatusTag.vue';
import GatewayRouteDrawer from '@/features/routes/GatewayRouteDrawer.vue';
import type { Route } from '@/interfaces/route';
import { showSuccess } from '@/lib/feedback';
import {
  addGatewayRouteCompat,
  deleteGatewayRouteCompat,
  getGatewayRoutesCompat,
  updateGatewayRouteCompat,
} from '@/services/route-compat';

const loading = ref(false);
const search = ref('');
const rows = ref<Route[]>([]);
const drawerOpen = ref(false);
const deleteOpen = ref(false);
const editing = ref<Route | null>(null);
const deleting = ref<Route | null>(null);

const filtered = computed(() => rows.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, (item.domains || []).join(','), item.path?.matchValue]
    .some((value) => String(value || '').toLowerCase().includes(keyword));
}));

async function load() {
  loading.value = true;
  try {
    const result = await getGatewayRoutesCompat().catch(() => ({ data: [] }));
    rows.value = Array.isArray(result) ? result : (result.data || []);
  } finally {
    loading.value = false;
  }
}

function openDrawer(record?: Route) {
  editing.value = record || null;
  drawerOpen.value = true;
}

async function submit(payload: Route, isEdit: boolean) {
  if (isEdit) {
    await updateGatewayRouteCompat(payload as any);
  } else {
    await addGatewayRouteCompat(payload as any);
  }
  drawerOpen.value = false;
  await load();
  showSuccess('保存成功');
}

async function confirmDelete() {
  if (!deleting.value) {
    return;
  }
  await deleteGatewayRouteCompat(deleting.value.name);
  deleteOpen.value = false;
  await load();
  showSuccess('删除成功');
}

onMounted(load);
</script>

<template>
  <PageSection title="路由配置">
    <ListToolbar v-model:search="search" search-placeholder="搜索路由名或域名" create-text="新增路由" @refresh="load" @create="openDrawer()" />
    <a-table :data-source="filtered" :loading="loading" row-key="name" :scroll="{ x: 1120 }">
      <a-table-column key="name" data-index="name" title="名称" />
      <a-table-column key="domains" title="域名">
        <template #default="{ record }">{{ (record.domains || []).join(', ') || '-' }}</template>
      </a-table-column>
      <a-table-column key="path" title="路径匹配">
        <template #default="{ record }">{{ record.path?.matchType }} | {{ record.path?.matchValue }}</template>
      </a-table-column>
      <a-table-column key="services" title="目标服务">
        <template #default="{ record }">
          {{ (record.services || []).map((item: any) => item.port ? `${item.name}:${item.port}` : item.name).join(', ') || '-' }}
        </template>
      </a-table-column>
      <a-table-column key="auth" title="认证">
        <template #default="{ record }"><StatusTag :value="record.authConfig?.enabled ? 'enabled' : 'disabled'" /></template>
      </a-table-column>
      <a-table-column key="actions" title="操作" width="240">
        <template #default="{ record }">
          <StrategyLink :path="`/route/config?type=route&name=${encodeURIComponent(record.name)}`" />
          <a-button type="link" size="small" @click="openDrawer(record)">编辑</a-button>
          <a-button type="link" size="small" danger @click="deleting = record; deleteOpen = true">删除</a-button>
        </template>
      </a-table-column>
    </a-table>

    <GatewayRouteDrawer
      v-model:open="drawerOpen"
      :route="editing"
      @submit="submit"
    />

    <DeleteConfirmModal v-model:open="deleteOpen" :content="deleting ? `确认删除 ${deleting.name} 吗？` : ''" @confirm="confirmDelete" />
  </PageSection>
</template>
