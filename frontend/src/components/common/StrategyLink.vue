<script setup lang="ts">
import { useRouter } from 'vue-router';

const props = defineProps<{
  path: string;
}>();

const router = useRouter();

function openStrategy() {
  router.push(props.path);
}
</script>

<template>
  <a-button type="link" size="small" @click="openStrategy">
    策略
  </a-button>
</template>
