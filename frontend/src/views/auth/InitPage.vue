<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import LanguageSwitcher from '@/components/app/LanguageSwitcher.vue';
import { initialize } from '@/services/system';
import { useAppStore } from '@/stores/app';
import { showError, showSuccess } from '@/lib/feedback';

const router = useRouter();
const appStore = useAppStore();
const { t } = useI18n();
const loading = ref(false);

const formState = reactive({
  name: 'admin',
  password: '',
  confirmPassword: '',
});

async function submit() {
  if (formState.password !== formState.confirmPassword) {
    showError(t('init.confirmPasswordMismatched'));
    return;
  }

  loading.value = true;
  try {
    await initialize({
      adminUser: {
        name: formState.name,
        displayName: formState.name,
        password: formState.password,
      },
    });
    await appStore.bootstrap(true);
    showSuccess(t('init.initSuccess'));
    router.replace('/login');
  } catch {
    showError(t('init.initFailed'));
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-page__toolbar">
      <LanguageSwitcher />
    </div>

    <div class="auth-card">
      <img src="/banner.png" alt="Higress" class="auth-card__banner" />
      <div class="auth-card__title">{{ t('init.header') }}</div>
      <a-form layout="vertical" :model="formState" @finish="submit">
        <a-form-item name="name" :rules="[{ required: true, message: t('init.usernameRequired') }]">
          <a-input v-model:value="formState.name" :placeholder="t('init.usernamePlaceholder')" size="large" />
        </a-form-item>
        <a-form-item name="password" :rules="[{ required: true, message: t('init.passwordRequired') }]">
          <a-input-password v-model:value="formState.password" :placeholder="t('init.passwordPlaceholder')" size="large" />
        </a-form-item>
        <a-form-item name="confirmPassword" :rules="[{ required: true, message: t('init.confirmPasswordRequired') }]">
          <a-input-password v-model:value="formState.confirmPassword" :placeholder="t('init.confirmPasswordPlaceholder')" size="large" />
        </a-form-item>
        <a-button type="primary" html-type="submit" size="large" block :loading="loading">
          {{ t('misc.submit') }}
        </a-button>
      </a-form>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 24px;
}

.auth-page__toolbar {
  position: absolute;
  top: 18px;
  right: 18px;
}

.auth-card {
  width: min(440px, 100%);
  padding: 28px;
  border: 1px solid var(--portal-border);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: var(--portal-shadow);
}

.auth-card__banner {
  width: 100%;
  margin-bottom: 12px;
  border-radius: 18px;
}

.auth-card__title {
  margin-bottom: 18px;
  font-size: 22px;
  font-weight: 700;
  text-align: center;
}
</style>
