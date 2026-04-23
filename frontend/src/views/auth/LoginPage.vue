<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import LanguageSwitcher from '@/components/app/LanguageSwitcher.vue';
import { login } from '@/services/user';
import { useAppStore } from '@/stores/app';
import { showError, showSuccess } from '@/lib/feedback';

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const appStore = useAppStore();
const loading = ref(false);

const formState = reactive({
  username: '',
  password: '',
});

async function submit() {
  loading.value = true;
  try {
    const user = await login(formState);
    appStore.setUser({
      ...user,
      type: user.type || 'admin',
    });
    showSuccess(t('login.loginSuccess'));
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard';
    router.replace(redirect);
  } catch {
    showError(t('login.loginFailed'));
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
      <div class="auth-card__title">{{ t('login.title') }}</div>

      <a-form
        layout="vertical"
        :model="formState"
        @finish="submit"
      >
        <a-form-item
          name="username"
          :rules="[{ required: true, message: t('login.usernameRequired') }]"
        >
          <a-input v-model:value="formState.username" :placeholder="t('login.usernamePlaceholder')" size="large" />
        </a-form-item>
        <a-form-item
          name="password"
          :rules="[{ required: true, message: t('login.passwordRequired') }]"
        >
          <a-input-password v-model:value="formState.password" :placeholder="t('login.passwordPlaceholder')" size="large" />
        </a-form-item>
        <a-button type="primary" html-type="submit" size="large" block :loading="loading">
          {{ t('login.buttonText') }}
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
