<script setup lang="ts">
import { computed } from 'vue';
import { RouterView, useRoute } from 'vue-router';
import { ConfigProvider, theme } from 'ant-design-vue';
import enUS from 'ant-design-vue/es/locale/en_US';
import zhCN from 'ant-design-vue/es/locale/zh_CN';
import { useI18n } from 'vue-i18n';
import AppShell from '@/components/app/AppShell.vue';

const { locale } = useI18n();
const route = useRoute();

const antLocale = computed(() => (locale.value === 'en-US' ? enUS : zhCN));
const blankLayout = computed(() => Boolean(route.meta.blank));
</script>

<template>
  <ConfigProvider
    :locale="antLocale"
    :theme="{
      algorithm: theme.defaultAlgorithm,
      token: {
        colorPrimary: '#1890ff',
        colorLink: '#1890ff',
        borderRadius: 14,
        fontFamily: `'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif`,
      },
    }"
  >
    <RouterView v-if="blankLayout" />
    <AppShell v-else />
  </ConfigProvider>
</template>
