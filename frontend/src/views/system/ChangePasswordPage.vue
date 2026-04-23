<script setup lang="ts">
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import PageSection from '@/components/common/PageSection.vue';
import ChangePasswordForm from '@/components/app/ChangePasswordForm.vue';
import { useAppStore } from '@/stores/app';
import { showSuccess } from '@/lib/feedback';

const router = useRouter();
const appStore = useAppStore();
const { t } = useI18n();

async function handleSuccess() {
  await appStore.signOut();
  showSuccess(t('user.changePassword.reloginPrompt'));
  router.push('/login');
}
</script>

<template>
  <PageSection :title="t('user.changePassword.title')">
    <div class="change-password-page">
      <ChangePasswordForm
        @cancel="router.back()"
        @success="handleSuccess"
      />
    </div>
  </PageSection>
</template>

<style scoped>
.change-password-page {
  max-width: 560px;
}
</style>
