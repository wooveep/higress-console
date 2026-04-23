<script setup lang="ts">
defineProps<{
  open: boolean;
  title?: string;
  content?: string;
  loading?: boolean;
}>();

const emit = defineEmits<{
  cancel: [];
  confirm: [];
  'update:open': [value: boolean];
}>();

function handleCancel() {
  emit('update:open', false);
  emit('cancel');
}
</script>

<template>
  <a-modal
    :open="open"
    :title="title || '删除'"
    :confirm-loading="loading"
    ok-text="确定"
    cancel-text="取消"
    @ok="emit('confirm')"
    @cancel="handleCancel"
  >
    <p>{{ content || '确认执行删除操作？' }}</p>
  </a-modal>
</template>
