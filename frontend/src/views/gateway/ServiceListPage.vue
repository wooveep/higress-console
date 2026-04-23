<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import PageSection from '@/components/common/PageSection.vue';
import ListToolbar from '@/components/common/ListToolbar.vue';
import { buildPluginTargetPath, QueryType } from '@/plugins/visibility';
import { getGatewayServices } from '@/services/service';

const router = useRouter();
const loading = ref(false);
const search = ref('');
const services = ref<any[]>([]);

const filtered = computed(() => services.value.filter((item) => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) {
    return true;
  }
  return [item.name, item.namespace, (item.endpoints || []).join(',')].some((value) => String(value || '').toLowerCase().includes(keyword));
}));

async function load() {
  loading.value = true;
  try {
    services.value = await getGatewayServices().catch(() => []);
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <PageSection title="服务列表">
    <ListToolbar v-model:search="search" search-placeholder="搜索服务名、命名空间、端点" create-text="" @refresh="load" />
    <a-table :data-source="filtered" :loading="loading" row-key="name" :scroll="{ x: 900 }">
      <a-table-column key="name" data-index="name" title="服务名" />
      <a-table-column key="namespace" data-index="namespace" title="命名空间" />
      <a-table-column key="port" data-index="port" title="端口" />
      <a-table-column key="endpoints" title="端点">
        <template #default="{ record }">{{ (record.endpoints || []).join(', ') || '-' }}</template>
      </a-table-column>
      <a-table-column key="actions" title="操作" width="120" fixed="right">
        <template #default="{ record }">
          <a-button
            type="link"
            size="small"
            @click="router.push(buildPluginTargetPath(QueryType.SERVICE, record.name))"
          >
            插件配置
          </a-button>
        </template>
      </a-table-column>
    </a-table>
  </PageSection>
</template>
