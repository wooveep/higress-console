<script setup lang="ts">
import { computed } from 'vue';
import { DownOutlined, MenuFoldOutlined, MenuUnfoldOutlined, UserOutlined } from '@ant-design/icons-vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import LanguageSwitcher from '@/components/app/LanguageSwitcher.vue';
import { useAppStore } from '@/stores/app';

const props = defineProps<{
  collapsed: boolean;
  mobile: boolean;
}>();

const emit = defineEmits<{
  toggleNav: [];
  changePassword: [];
  logout: [];
}>();

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const appStore = useAppStore();

const title = computed(() => t(route.meta.titleKey || 'index.title'));

const dropdownItems = computed(() => ([
  {
    key: 'change-password',
    label: t('user.changePassword.title'),
  },
  {
    key: 'logout',
    label: t('misc.logout'),
  },
]));

function handleMenuClick(event: { key: string | number }) {
  if (String(event.key) === 'change-password') {
    emit('changePassword');
    return;
  }
  emit('logout');
}

function jumpHome() {
  router.push('/dashboard');
}
</script>

<template>
  <header class="app-topbar">
    <div class="app-topbar__left">
      <a-button type="text" class="app-topbar__nav-toggle" @click="emit('toggleNav')">
        <MenuUnfoldOutlined v-if="collapsed || mobile" />
        <MenuFoldOutlined v-else />
      </a-button>
      <button class="app-topbar__title-wrap" type="button" @click="jumpHome">
        <div class="app-topbar__title">{{ title }}</div>
      </button>
    </div>

    <div class="app-topbar__right">
      <LanguageSwitcher />
      <a-dropdown
        :menu="{ items: dropdownItems, onClick: handleMenuClick }"
        :trigger="['click']"
        placement="bottomRight"
      >
        <button
          class="app-topbar__account"
          type="button"
          :aria-label="t('user.changePassword.title')"
        >
          <span class="app-topbar__avatar">
            <UserOutlined />
          </span>
          <span class="app-topbar__account-copy">
            <span class="app-topbar__account-name">{{ appStore.displayName }}</span>
            <span class="app-topbar__account-meta">{{ appStore.currentUser?.username || 'admin' }}</span>
          </span>
          <DownOutlined class="app-topbar__account-arrow" />
        </button>
      </a-dropdown>
    </div>
  </header>
</template>

<style scoped>
.app-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 64px;
  padding: 10px 20px;
  border-bottom: 1px solid var(--portal-border);
  background: rgba(255, 255, 255, 0.94);
}

.app-topbar__left,
.app-topbar__right {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.app-topbar__nav-toggle {
  flex-shrink: 0;
}

.app-topbar__title-wrap {
  padding: 0;
  border: none;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.app-topbar__title {
  font-size: 22px;
  font-weight: 700;
  color: var(--portal-text);
}

.app-topbar__account {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  border: 1px solid var(--portal-border);
  border-radius: 999px;
  background: #fff;
  cursor: pointer;
}

.app-topbar__account-arrow {
  color: var(--portal-text-muted);
  font-size: 12px;
}

.app-topbar__avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--portal-primary), #53b7ff);
  color: #fff;
}

.app-topbar__account-copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  line-height: 1.2;
}

.app-topbar__account-name {
  font-size: 13px;
  font-weight: 700;
}

.app-topbar__account-meta {
  color: var(--portal-text-muted);
  font-size: 12px;
}

@media (max-width: 767px) {
  .app-topbar {
    padding-inline: 14px;
  }

  .app-topbar__account-copy {
    display: none;
  }

  .app-topbar__title {
    font-size: 18px;
  }
}
</style>
