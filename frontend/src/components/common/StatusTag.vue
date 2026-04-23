<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  value?: string | boolean | null;
  text?: string | null;
}>();

const color = computed(() => {
  const value = String(props.value ?? '').toLowerCase();
  if (value === 'published' || value === 'success' || value === 'true' || value === 'active' || value === 'enabled') {
    return 'green';
  }
  if (value === 'draft' || value === 'pending') {
    return 'blue';
  }
  if (value === 'disabled' || value === 'false' || value === 'rejected' || value === 'error') {
    return 'red';
  }
  if (value === 'unpublished' || value === 'warning') {
    return 'orange';
  }
  return 'default';
});
</script>

<template>
  <a-tag :color="color">
    {{ text ?? value ?? '-' }}
  </a-tag>
</template>
