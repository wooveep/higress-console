<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import AppSidebar from '@/components/app/AppSidebar.vue';
import AppTopbar from '@/components/app/AppTopbar.vue';
import { showSuccess } from '@/lib/feedback';
import { useAppStore } from '@/stores/app';

const ChangePasswordForm = defineAsyncComponent(() => import('@/components/app/ChangePasswordForm.vue'));

const appStore = useAppStore();
const router = useRouter();
const { t } = useI18n();

const viewportWidth = ref(window.innerWidth);
const mobileNavOpen = ref(false);
const passwordModalOpen = ref(false);

const isMobile = computed(() => viewportWidth.value < 768);
const isRail = computed(() => viewportWidth.value >= 768 && viewportWidth.value < 1200);
const collapsed = computed(() => isRail.value);

function onResize() {
  viewportWidth.value = window.innerWidth;
  if (!isMobile.value) {
    mobileNavOpen.value = false;
  }
}

function toggleNav() {
  if (isMobile.value) {
    mobileNavOpen.value = !mobileNavOpen.value;
  }
}

async function handleLogout() {
  await appStore.signOut();
  showSuccess(t('misc.logout'));
  router.push('/login');
}

async function handlePasswordSuccess() {
  passwordModalOpen.value = false;
  await appStore.signOut();
  router.push('/login');
}

onMounted(() => {
  window.addEventListener('resize', onResize);
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize);
});
</script>

<template>
  <div class="app-shell" :class="{ 'app-shell--mobile': isMobile, 'app-shell--rail': isRail }">
    <div v-if="!isMobile" class="app-shell__sidebar">
      <AppSidebar :collapsed="collapsed" @navigate="mobileNavOpen = false" />
    </div>

    <a-drawer
      v-model:open="mobileNavOpen"
      placement="left"
      width="280"
      :body-style="{ padding: 0 }"
      :closable="false"
      class="app-shell__drawer"
    >
      <AppSidebar mobile @navigate="mobileNavOpen = false" :collapsed="false" />
    </a-drawer>

    <div class="app-shell__main">
      <AppTopbar
        :collapsed="collapsed"
        :mobile="isMobile"
        @toggle-nav="toggleNav"
        @change-password="passwordModalOpen = true"
        @logout="handleLogout"
      />

      <main class="app-shell__content">
        <RouterView />
      </main>
    </div>

    <a-modal
      v-model:open="passwordModalOpen"
      :footer="null"
      :title="t('user.changePassword.title')"
      destroy-on-close
    >
      <ChangePasswordForm
        @cancel="passwordModalOpen = false"
        @success="handlePasswordSuccess"
      />
    </a-modal>
  </div>
</template>

<style scoped>
.app-shell {
  display: grid;
  grid-template-columns: 288px minmax(0, 1fr);
  min-height: 100vh;
}

.app-shell--rail {
  grid-template-columns: 84px minmax(0, 1fr);
}

.app-shell--mobile {
  grid-template-columns: minmax(0, 1fr);
}

.app-shell__sidebar {
  position: sticky;
  top: 0;
  height: 100vh;
}

.app-shell__main {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.app-shell__content {
  min-width: 0;
  padding: 22px;
}

@media (max-width: 767px) {
  .app-shell__content {
    padding: 14px;
  }
}
</style>
