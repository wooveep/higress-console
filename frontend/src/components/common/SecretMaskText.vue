<script setup lang="ts">
import { computed, ref } from 'vue';

const props = defineProps<{
  value?: string;
}>();

const reveal = ref(false);

const text = computed(() => props.value || '-');
const masked = computed(() => {
  if (!props.value || reveal.value) {
    return text.value;
  }
  if (props.value.length <= 6) {
    return `${props.value.slice(0, 1)}******`;
  }
  return `${props.value.slice(0, 3)}******${props.value.slice(-3)}`;
});
</script>

<template>
  <span class="secret-mask-text">
    <span>{{ masked }}</span>
    <a-button
      v-if="value"
      type="link"
      size="small"
      class="secret-mask-text__toggle"
      @click="reveal = !reveal"
    >
      {{ reveal ? '隐藏' : '显示' }}
    </a-button>
  </span>
</template>

<style scoped>
.secret-mask-text {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.secret-mask-text__toggle {
  padding-inline: 0;
}
</style>
