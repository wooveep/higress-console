<script setup lang="ts">
defineProps<{
  confirmText?: string;
  loading?: boolean;
}>();

const emit = defineEmits<{
  cancel: [];
  confirm: [];
}>();
</script>

<template>
  <div class="drawer-footer">
    <slot name="extra" />
    <a-button @click="emit('cancel')">取消</a-button>
    <a-button type="primary" :loading="loading" @click="emit('confirm')">
      {{ confirmText || '确定' }}
    </a-button>
  </div>
</template>

<style scoped>
.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 18px;
  flex-wrap: wrap;
}
</style>
