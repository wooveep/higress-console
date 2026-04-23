<script setup lang="ts">
defineProps<{
  search?: string;
  searchPlaceholder?: string;
  showSearch?: boolean;
  createText?: string;
  createDisabled?: boolean;
}>();

const emit = defineEmits<{
  'update:search': [value: string];
  search: [];
  refresh: [];
  create: [];
}>();
</script>

<template>
  <div class="list-toolbar">
    <div class="list-toolbar__left">
      <slot name="left" />
      <a-input-search
        v-if="showSearch !== false"
        :value="search"
        :placeholder="searchPlaceholder"
        allow-clear
        class="list-toolbar__search"
        @update:value="(value) => emit('update:search', String(value ?? ''))"
        @search="emit('search')"
      />
    </div>
    <div class="list-toolbar__right">
      <slot name="right" />
      <a-button @click="emit('refresh')">刷新</a-button>
      <a-button
        v-if="createText"
        type="primary"
        :disabled="createDisabled"
        @click="emit('create')"
      >
        {{ createText }}
      </a-button>
    </div>
  </div>
</template>

<style scoped>
.list-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 16px;
}

.list-toolbar__left,
.list-toolbar__right {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.list-toolbar__search {
  width: 280px;
}

@media (max-width: 767px) {
  .list-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .list-toolbar__left,
  .list-toolbar__right {
    flex-wrap: wrap;
  }

  .list-toolbar__search {
    width: 100%;
  }
}
</style>
