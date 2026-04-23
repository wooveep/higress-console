<script setup lang="ts">
import StatusTag from '@/components/common/StatusTag.vue';
import { describePricing } from './model-asset-form';

defineProps<{
  open: boolean;
  title: string;
  loading?: boolean;
  versions: any[];
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  restore: [versionId: number];
}>();
</script>

<template>
  <a-drawer
    :open="open"
    width="920"
    :title="title"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <p class="model-binding-history-drawer__tip">
      历史版本只支持恢复到草稿，不会直接重新生效；回退流程固定为 restore -> publish。
    </p>
    <a-table :data-source="versions" :loading="loading" row-key="versionId" :scroll="{ x: 880 }">
      <a-table-column key="versionId" data-index="versionId" title="版本">
        <template #default="{ record }">#{{ record.versionId }}</template>
      </a-table-column>
      <a-table-column key="status" title="状态" width="150">
        <template #default="{ record }">
          <div class="model-binding-history-drawer__status">
            <StatusTag :value="record.status || '-'" />
            <a-tag v-if="record.active" color="green">current</a-tag>
          </div>
        </template>
      </a-table-column>
      <a-table-column key="pricing" title="价格摘要">
        <template #default="{ record }">{{ describePricing(record.pricing) }}</template>
      </a-table-column>
      <a-table-column key="effectiveFrom" data-index="effectiveFrom" title="生效时间" width="180" />
      <a-table-column key="effectiveTo" data-index="effectiveTo" title="失效时间" width="180" />
      <a-table-column key="actions" title="操作" width="140" fixed="right">
        <template #default="{ record }">
          <a-button type="link" size="small" @click="emit('restore', record.versionId)">恢复到草稿</a-button>
        </template>
      </a-table-column>
    </a-table>
  </a-drawer>
</template>

<style scoped>
.model-binding-history-drawer__tip {
  margin: 0 0 14px;
  color: var(--portal-text-soft);
}

.model-binding-history-drawer__status {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
