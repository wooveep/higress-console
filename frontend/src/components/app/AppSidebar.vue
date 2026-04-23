<script setup lang="ts">
import { computed, h } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { navItems } from '@/router/routes';

const props = defineProps<{
  collapsed: boolean;
  mobile?: boolean;
}>();

const emit = defineEmits<{
  navigate: [];
}>();

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const selectedKeys = computed(() => [route.meta.navKey || route.path]);

const openKeys = computed(() => {
  const matched = navItems.find((item) => item.children?.some((child) => child.key === route.meta.navKey));
  return matched ? [matched.key] : [];
});

const menuItems = computed(() => navItems.map((item) => {
  if (item.children?.length) {
    return {
      key: item.key,
      icon: item.icon ? () => h(item.icon) : undefined,
      label: t(item.titleKey),
      children: item.children.map((child) => ({
        key: child.path || child.key,
        icon: child.icon ? () => h(child.icon) : undefined,
        label: t(child.titleKey),
      })),
    };
  }

  return {
    key: item.path || item.key,
    icon: item.icon ? () => h(item.icon) : undefined,
    label: t(item.titleKey),
  };
}));

function handleSelect(info: { key: string | number }) {
  router.push(String(info.key));
  emit('navigate');
}
</script>

<template>
  <aside class="app-sidebar" :class="{ 'app-sidebar--collapsed': collapsed, 'app-sidebar--mobile': mobile }">
    <div class="app-sidebar__brand">
      <img src="/logo-ai.svg" alt="aigateway" class="app-sidebar__logo" />
      <div v-if="!collapsed || mobile" class="app-sidebar__copy">
        <div class="app-sidebar__title">aigateway</div>
        <div class="app-sidebar__meta">Console</div>
      </div>
    </div>
    <div class="app-sidebar__menu">
      <a-menu
        mode="inline"
        :inline-collapsed="collapsed && !mobile"
        :selected-keys="selectedKeys"
        :default-open-keys="openKeys"
        :items="menuItems"
        @select="handleSelect"
      />
    </div>
  </aside>
</template>

<style scoped>
.app-sidebar {
  display: flex;
  flex-direction: column;
  gap: 14px;
  height: 100%;
  min-height: 0;
  padding: 18px 14px;
  border-right: 1px solid var(--portal-border);
  background: rgba(255, 255, 255, 0.96);
}

.app-sidebar--collapsed {
  padding-inline: 10px;
}

.app-sidebar__brand {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 48px;
  padding: 6px 8px;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(24, 144, 255, 0.13), rgba(24, 144, 255, 0.03));
}

.app-sidebar__logo {
  width: 34px;
  height: 34px;
  flex-shrink: 0;
}

.app-sidebar__title {
  font-size: 15px;
  font-weight: 700;
}

.app-sidebar__meta {
  color: var(--portal-text-muted);
  font-size: 12px;
}

.app-sidebar__menu {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 2px;
  scrollbar-width: thin;
  scrollbar-color: rgba(92, 112, 143, 0.34) transparent;
}

.app-sidebar__menu::-webkit-scrollbar {
  width: 8px;
}

.app-sidebar__menu::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(92, 112, 143, 0.26);
}

:deep(.ant-menu) {
  border-inline-end: none !important;
  background: transparent;
}

:deep(.ant-menu-item),
:deep(.ant-menu-submenu-title) {
  height: 40px;
  line-height: 40px;
  margin-block: 3px;
  border-radius: 12px;
}

:deep(.ant-menu-item-selected) {
  background: var(--portal-primary-soft);
  color: var(--portal-primary);
  font-weight: 600;
}
</style>
