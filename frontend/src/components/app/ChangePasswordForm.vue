<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { changePassword } from '@/services/user';
import { showError, showSuccess } from '@/lib/feedback';

const emit = defineEmits<{
  success: [];
  cancel: [];
}>();

const { t } = useI18n();
const loading = ref(false);

const formState = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
});

async function submit() {
  if (!formState.oldPassword || !formState.newPassword || !formState.confirmPassword) {
    return;
  }
  if (formState.newPassword !== formState.confirmPassword) {
    showError(t('user.changePassword.confirmPasswordMismatched'));
    return;
  }

  loading.value = true;
  try {
    await changePassword({
      oldPassword: formState.oldPassword,
      newPassword: formState.newPassword,
    });
    showSuccess(t('user.changePassword.reloginPrompt'));
    emit('success');
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <a-form
    layout="vertical"
    :model="formState"
    @finish="submit"
  >
    <a-form-item
      :label="t('user.changePassword.oldPassword')"
      name="oldPassword"
      :rules="[{ required: true, message: t('user.changePassword.oldPasswordRequired') }]"
    >
      <a-input-password v-model:value="formState.oldPassword" />
    </a-form-item>
    <a-form-item
      :label="t('user.changePassword.newPassword')"
      name="newPassword"
      :rules="[{ required: true, message: t('user.changePassword.newPasswordRequired') }]"
    >
      <a-input-password v-model:value="formState.newPassword" />
    </a-form-item>
    <a-form-item
      :label="t('user.changePassword.confirmPassword')"
      name="confirmPassword"
      :rules="[{ required: true, message: t('user.changePassword.confirmPasswordRequired') }]"
    >
      <a-input-password v-model:value="formState.confirmPassword" />
    </a-form-item>
    <div class="change-password-form__actions">
      <a-button @click="emit('cancel')">{{ t('misc.cancel') }}</a-button>
      <a-button type="primary" html-type="submit" :loading="loading">
        {{ t('misc.submit') }}
      </a-button>
    </div>
  </a-form>
</template>

<style scoped>
.change-password-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
