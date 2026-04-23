<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import PageSection from '@/components/common/PageSection.vue';
import { getMcpServer, listMcpConsumers } from '@/services/mcp';

const route = useRoute();
const record = ref<any>(null);
const consumers = ref<any[]>([]);

const resolveName = () => String(route.params.name || route.query.name || '');

async function load() {
  const name = resolveName();
  if (!name) {
    record.value = null;
    consumers.value = [];
    return;
  }
  record.value = await getMcpServer(name).catch(() => null);
  const result = await listMcpConsumers({ mcpServerName: name, pageNum: 1, pageSize: 200 }).catch(() => ({ data: [] }));
  consumers.value = Array.isArray(result) ? result : (result.data || []);
}

watch(() => route.fullPath, load);
onMounted(load);
</script>

<template>
  <div class="mcp-detail-page">
    <PageSection title="MCP Server">
      <pre class="portal-pre">{{ JSON.stringify(record, null, 2) }}</pre>
    </PageSection>
    <PageSection title="Consumers">
      <a-table :data-source="consumers" row-key="consumerName" size="small">
        <a-table-column key="consumerName" data-index="consumerName" title="Consumer" />
        <a-table-column key="type" data-index="type" title="Type" />
        <a-table-column key="mcpServerName" data-index="mcpServerName" title="MCP Server" />
      </a-table>
    </PageSection>
  </div>
</template>

<style scoped>
.mcp-detail-page {
  display: grid;
  gap: 18px;
}
</style>
