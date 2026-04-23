<script setup lang="ts">
defineProps<{
  title: string;
  subtle?: boolean;
}>();
</script>

<template>
  <section class="page-section" :class="{ 'page-section--subtle': subtle }">
    <header class="page-section__header">
      <div class="page-section__title">{{ title }}</div>
      <div class="page-section__actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="page-section__body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.page-section {
  min-width: 0;
  border: 1px solid var(--portal-border);
  border-radius: 20px;
  background: var(--portal-surface);
  box-shadow: var(--portal-shadow);
}

.page-section--subtle {
  background: rgba(255, 255, 255, 0.96);
}

.page-section__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 18px 20px 0;
}

.page-section__title {
  font-size: 16px;
  font-weight: 700;
  color: var(--portal-text);
}

.page-section__actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.page-section__body {
  min-width: 0;
  padding: 18px 20px 20px;
}
</style>
