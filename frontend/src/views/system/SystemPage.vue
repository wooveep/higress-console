<script setup lang="ts">
import { computed, onMounted, reactive, shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { message } from 'ant-design-vue';
import PageSection from '@/components/common/PageSection.vue';
import HigressGlobalConfigForm from '@/components/system/HigressGlobalConfigForm.vue';
import { useHigressGlobalConfig } from '@/composables/system/useHigressGlobalConfig';
import type { PortalSSOConfigRecord } from '@/interfaces/system';
import { getPortalSSOConfig, getSystemInfo, updatePortalSSOConfig } from '@/services/system';

const router = useRouter();
const { t } = useI18n();
const systemInfoLoading = shallowRef(false);
const systemInfo = shallowRef<Record<string, any>>({});
const hiddenSystemInfoKeys = new Set(['legacyNaming', 'legacyBackend', 'phase']);
const configState = useHigressGlobalConfig();
const portalSSOLoading = shallowRef(false);
const portalSSOSaving = shallowRef(false);
const portalSSOForm = reactive(createPortalSSOFormState());
const portalSSOUpdatedMeta = shallowRef('');

async function loadSystemInfo() {
  systemInfoLoading.value = true;
  try {
    const info = await getSystemInfo().catch(() => ({}));
    systemInfo.value = info || {};
  } finally {
    systemInfoLoading.value = false;
  }
}

function createPortalSSOFormState() {
  return {
    enabled: false,
    displayName: '企业 SSO 登录',
    issuerUrl: '',
    clientId: '',
    clientSecret: '',
    clientSecretMasked: '',
    clientSecretConfigured: false,
    scopesText: 'openid profile email',
    claimEmail: 'email',
    claimDisplayName: 'name',
    claimUsername: 'preferred_username',
  };
}

function applyPortalSSOConfig(config: PortalSSOConfigRecord) {
  portalSSOForm.enabled = Boolean(config.enabled);
  portalSSOForm.displayName = config.displayName || '企业 SSO 登录';
  portalSSOForm.issuerUrl = config.issuerUrl || '';
  portalSSOForm.clientId = config.clientId || '';
  portalSSOForm.clientSecret = '';
  portalSSOForm.clientSecretMasked = config.clientSecretMasked || '';
  portalSSOForm.clientSecretConfigured = Boolean(config.clientSecretConfigured);
  portalSSOForm.scopesText = (config.scopes || []).join(' ') || 'openid profile email';
  portalSSOForm.claimEmail = config.claimMapping?.email || 'email';
  portalSSOForm.claimDisplayName = config.claimMapping?.displayName || 'name';
  portalSSOForm.claimUsername = config.claimMapping?.username || 'preferred_username';
  portalSSOUpdatedMeta.value = [config.updatedBy, config.updatedAt].filter(Boolean).join(' · ');
}

function normalizeScopes(raw: string) {
  const scopes = raw
    .split(/[\s,]+/)
    .map((item) => item.trim())
    .filter(Boolean);
  const seen = new Set<string>();
  const result: string[] = [];
  if (!scopes.includes('openid')) {
    result.push('openid');
    seen.add('openid');
  }
  scopes.forEach((scope) => {
    if (seen.has(scope))
      return;
    seen.add(scope);
    result.push(scope);
  });
  return result.length ? result : ['openid', 'profile', 'email'];
}

const portalSSODiscoveryURL = computed(() => {
  const issuer = portalSSOForm.issuerUrl.trim().replace(/\/+$/, '');
  return issuer ? `${issuer}/.well-known/openid-configuration` : '';
});

const visibleSystemInfoEntries = computed(() =>
  Object.entries(systemInfo.value).filter(([key]) => !hiddenSystemInfoKeys.has(key)),
);

async function loadPortalSSOConfig() {
  portalSSOLoading.value = true;
  try {
    const config = await getPortalSSOConfig();
    applyPortalSSOConfig(config);
  } finally {
    portalSSOLoading.value = false;
  }
}

async function savePortalSSOConfig() {
  portalSSOSaving.value = true;
  try {
    const saved = await updatePortalSSOConfig({
      enabled: portalSSOForm.enabled,
      providerType: 'oidc',
      displayName: portalSSOForm.displayName.trim(),
      issuerUrl: portalSSOForm.issuerUrl.trim(),
      clientId: portalSSOForm.clientId.trim(),
      clientSecret: portalSSOForm.clientSecret.trim(),
      clientSecretMasked: portalSSOForm.clientSecretMasked,
      clientSecretConfigured: portalSSOForm.clientSecretConfigured,
      scopes: normalizeScopes(portalSSOForm.scopesText),
      claimMapping: {
        email: portalSSOForm.claimEmail.trim(),
        displayName: portalSSOForm.claimDisplayName.trim(),
        username: portalSSOForm.claimUsername.trim(),
      },
    });
    applyPortalSSOConfig(saved);
    message.success('Portal SSO 配置已保存');
  } catch (error: any) {
    message.error(error?.response?.data?.message || '保存 Portal SSO 配置失败');
  } finally {
    portalSSOSaving.value = false;
  }
}

async function load() {
  await Promise.all([
    loadSystemInfo(),
    configState.load(),
    loadPortalSSOConfig(),
  ]);
}

onMounted(load);
</script>

<template>
  <div class="system-page">
    <PageSection :title="t('menu.systemSettings')">
      <template #actions>
        <a-button @click="router.push('/system/jobs')">Jobs 运维</a-button>
        <a-button @click="loadSystemInfo">{{ t('misc.refresh') }}</a-button>
      </template>
      <a-skeleton v-if="systemInfoLoading" active />
      <div v-else class="system-page__overview">
        <article
          v-for="[key, value] in visibleSystemInfoEntries"
          :key="key"
          class="system-page__stat"
        >
          <span>{{ key }}</span>
          <strong>{{ typeof value === 'object' ? JSON.stringify(value) : String(value) }}</strong>
        </article>
      </div>
    </PageSection>

    <PageSection title="AIGateway 配置">
      <HigressGlobalConfigForm
        :loading="configState.loading.value"
        :saving="configState.saving.value"
        :value="configState.formState.value"
        :raw-yaml="configState.rawYaml.value"
        :dirty="configState.dirty.value"
        :save-disabled="configState.saveDisabled.value"
        :parse-error="configState.parseError.value"
        :save-error="configState.saveError.value"
        :validation-errors="configState.validationErrors.value"
        @refresh="configState.load"
        @save="configState.save"
        @update-form="configState.updateForm"
        @update-raw-yaml="configState.updateRawYaml"
      />
    </PageSection>

    <PageSection title="Portal SSO">
      <template #actions>
        <a-button @click="loadPortalSSOConfig">刷新</a-button>
        <a-button type="primary" :loading="portalSSOSaving" @click="savePortalSSOConfig">保存</a-button>
      </template>

      <a-skeleton v-if="portalSSOLoading" active />
      <div v-else class="system-page__sso">
        <a-alert
          type="info"
          show-icon
          message="首版仅支持单一全局 OIDC Provider。首次邮箱首绑成功后，Portal 账号仍需管理员启用。"
        />

        <a-form layout="vertical" class="system-page__sso-form">
          <div class="system-page__sso-grid">
            <a-form-item label="启用 Portal SSO">
              <a-switch v-model:checked="portalSSOForm.enabled" />
            </a-form-item>

            <a-form-item label="登录按钮文案">
              <a-input v-model:value="portalSSOForm.displayName" placeholder="企业 SSO 登录" />
            </a-form-item>

            <a-form-item label="Issuer URL">
              <a-input v-model:value="portalSSOForm.issuerUrl" placeholder="https://idp.example.com/realms/main" />
            </a-form-item>

            <a-form-item label="Client ID">
              <a-input v-model:value="portalSSOForm.clientId" placeholder="portal-client" />
            </a-form-item>

            <a-form-item label="Client Secret" :extra="portalSSOForm.clientSecretConfigured ? `已配置：${portalSSOForm.clientSecretMasked}` : '首次保存必须填写，读取时不会回显明文。'">
              <a-input-password v-model:value="portalSSOForm.clientSecret" placeholder="留空则保留现有密钥" />
            </a-form-item>

            <a-form-item label="Scopes" extra="使用空格、换行或逗号分隔，系统会自动补 openid。">
              <a-textarea v-model:value="portalSSOForm.scopesText" :rows="3" placeholder="openid profile email" />
            </a-form-item>

            <a-form-item label="Email Claim">
              <a-input v-model:value="portalSSOForm.claimEmail" placeholder="email" />
            </a-form-item>

            <a-form-item label="Display Name Claim">
              <a-input v-model:value="portalSSOForm.claimDisplayName" placeholder="name" />
            </a-form-item>

            <a-form-item label="Username Claim">
              <a-input v-model:value="portalSSOForm.claimUsername" placeholder="preferred_username" />
            </a-form-item>
          </div>
        </a-form>

        <div class="system-page__sso-meta">
          <span>Discovery</span>
          <code>{{ portalSSODiscoveryURL || '请先填写 Issuer URL' }}</code>
        </div>
        <div v-if="portalSSOUpdatedMeta" class="system-page__sso-meta">
          <span>最近更新</span>
          <strong>{{ portalSSOUpdatedMeta }}</strong>
        </div>
      </div>
    </PageSection>
  </div>
</template>

<style scoped>
.system-page {
  display: grid;
  gap: 18px;
}

.system-page__overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.system-page__stat {
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--portal-border);
  border-radius: 16px;
  background: var(--portal-surface-soft);
}

.system-page__stat span {
  display: block;
  margin-bottom: 8px;
  color: var(--portal-text-soft);
  font-size: 12px;
}

.system-page__stat strong {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
}

.system-page__sso {
  display: grid;
  gap: 16px;
}

.system-page__sso-form {
  margin-top: 4px;
}

.system-page__sso-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

.system-page__sso-meta {
  display: grid;
  gap: 6px;
  padding: 12px 14px;
  border: 1px solid var(--portal-border);
  border-radius: 14px;
  background: var(--portal-surface-soft);
}

.system-page__sso-meta span {
  color: var(--portal-text-soft);
  font-size: 12px;
}

.system-page__sso-meta code,
.system-page__sso-meta strong {
  word-break: break-all;
}

@media (max-width: 1023px) {
  .system-page__overview {
    grid-template-columns: 1fr;
  }

  .system-page__sso-grid {
    grid-template-columns: 1fr;
  }
}
</style>
