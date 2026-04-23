<script setup lang="ts">
import { computed } from 'vue';
import { GlobalOutlined } from '@ant-design/icons-vue';
import { useI18n } from 'vue-i18n';
import i18n, { LANGUAGE_STORAGE_KEY, lngs } from '@/i18n';

const { locale } = useI18n();

const selected = computed(() => locale.value);

function updateLocale(value: string) {
  locale.value = value;
  i18n.global.locale.value = value as 'zh-CN' | 'en-US';
  localStorage.setItem(LANGUAGE_STORAGE_KEY, value);
}
</script>

<template>
  <a-select
    size="small"
    class="language-switcher"
    :value="selected"
    @update:value="updateLocale"
  >
    <template #suffixIcon>
      <GlobalOutlined />
    </template>
    <a-select-option
      v-for="item in lngs"
      :key="item.code"
      :value="item.code"
    >
      {{ item.nativeName }}
    </a-select-option>
  </a-select>
</template>

<style scoped>
.language-switcher {
  width: 122px;
}
</style>
