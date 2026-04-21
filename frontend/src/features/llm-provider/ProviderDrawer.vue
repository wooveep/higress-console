<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons-vue';
import { useI18n } from 'vue-i18n';
import DrawerFooter from '@/components/common/DrawerFooter.vue';
import type { LlmProvider } from '@/interfaces/llm-provider';
import type { ProxyServer } from '@/interfaces/proxy-server';
import type { Service } from '@/interfaces/service';
import { serviceToString } from '@/interfaces/service';
import { showError } from '@/lib/feedback';
import { getProxyServers } from '@/services/proxy-server';
import { getGatewayServices } from '@/services/service';
import {
  bedrockRegionOptions,
  buildProviderPayload,
  createProviderFormState,
  geminiSafetyCategoryOptions,
  geminiSafetyThresholdOptions,
  getProviderTypeOptions,
  providerProtocolOptions,
  shouldShowTokenInputs,
  toProviderFormState,
  type ProviderFormState,
  vertexRegionOptions,
} from './provider-form';

const props = defineProps<{
  open: boolean;
  provider?: (LlmProvider & Record<string, any>) | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [payload: LlmProvider & Record<string, any>, isEdit: boolean];
}>();

const { t } = useI18n();
const saving = ref(false);
const secretModalOpen = ref(false);
const loadingOptions = ref(false);
const proxyServers = ref<ProxyServer[]>([]);
const services = ref<Service[]>([]);
const formState = reactive<ProviderFormState>(createProviderFormState());

const providerTypeOptions = computed(() => getProviderTypeOptions(t));
const servicesByDisplayName = computed(() => new Map(services.value.map((item) => [serviceToString(item), item])));
const proxyOptions = computed(() => {
  const options = [{ label: t('serviceSource.serviceSourceForm.proxyServerNone'), value: '' }];
  const currentValue = formState.proxyName.trim();
  for (const item of [...proxyServers.value].sort((left, right) => left.name.localeCompare(right.name))) {
    options.push({ label: item.name, value: item.name });
  }
  if (currentValue && !options.some((item) => item.value === currentValue)) {
    options.push({ label: `${currentValue}（历史值）`, value: currentValue });
  }
  return options;
});

const serviceOptions = computed(() => {
  const options = [...services.value]
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((item) => ({ label: serviceToString(item), value: serviceToString(item) }));
  const currentValue = formState.openaiCustomService.trim();
  if (currentValue && !options.some((item) => item.value === currentValue)) {
    options.unshift({ label: `${currentValue}（历史值）`, value: currentValue });
  }
  return options;
});

const tokenInputVisible = computed(() => shouldShowTokenInputs(formState));
const openaiCustomUrlMode = computed(() => formState.type === 'openai' && formState.openaiServerType === 'custom' && formState.openaiCustomServerType === 'url');
const openaiCustomServiceMode = computed(() => formState.type === 'openai' && formState.openaiServerType === 'custom' && formState.openaiCustomServerType === 'service');
const qwenCustomMode = computed(() => formState.type === 'qwen' && formState.qwenServerType === 'custom');
const vertexServiceAccountMode = computed(() => formState.type === 'vertex' && formState.vertexAuthMode === 'serviceAccount');
const vertexApiKeyMode = computed(() => formState.type === 'vertex' && formState.vertexAuthMode === 'apiKey');
const isVolcengine = computed(() => formState.type === 'volcengine');

watch(() => props.open, async (open) => {
  if (!open) {
    return;
  }
  loadingOptions.value = true;
  try {
    const [proxyList, serviceList] = await Promise.all([
      getProxyServers().catch(() => []),
      getGatewayServices().catch(() => []),
    ]);
    proxyServers.value = proxyList || [];
    services.value = serviceList || [];
  } finally {
    loadingOptions.value = false;
  }
  Object.assign(formState, toProviderFormState(props.provider || undefined));
});

watch(() => formState.hiclawMode, (enabled) => {
  if (enabled) {
    formState.promoteThinkingOnEmpty = true;
  }
});

watch(() => formState.type, (type) => {
  if (type === 'volcengine' && !formState.volcengineBaseUrl.trim()) {
    formState.volcengineBaseUrl = 'https://ark.cn-beijing.volces.com/api/v3';
  }
  if (type === 'openai' && !formState.openaiServerType) {
    formState.openaiServerType = 'official';
  }
  if (type === 'qwen' && !formState.qwenServerType) {
    formState.qwenServerType = 'official';
  }
  if (type === 'vertex' && !formState.vertexAuthMode) {
    formState.vertexAuthMode = 'serviceAccount';
  }
});

function close() {
  emit('update:open', false);
}

function addToken() {
  formState.tokens.push('');
}

function removeToken(index: number) {
  if (formState.tokens.length <= 1) {
    return;
  }
  formState.tokens.splice(index, 1);
}

function addOpenAICustomUrl() {
  formState.openaiCustomUrls.push('');
}

function removeOpenAICustomUrl(index: number) {
  if (formState.openaiCustomUrls.length <= 1) {
    return;
  }
  formState.openaiCustomUrls.splice(index, 1);
}

function addQwenFileId() {
  formState.qwenFileIds.push('');
}

function removeQwenFileId(index: number) {
  formState.qwenFileIds.splice(index, 1);
}

function addGeminiSafetySetting() {
  formState.geminiSafetySettings.push({ category: '', threshold: '' });
}

function removeGeminiSafetySetting(index: number) {
  formState.geminiSafetySettings.splice(index, 1);
}

async function submit() {
  saving.value = true;
  try {
    const payload = buildProviderPayload(formState, {
      original: props.provider,
      servicesByDisplayName: servicesByDisplayName.value,
      t,
    });
    emit('submit', payload, Boolean(props.provider));
  } catch (error) {
    showError((error as Error).message || '表单校验失败');
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <a-drawer
    :open="open"
    width="880"
    :title="provider ? t('llmProvider.edit') : t('llmProvider.create')"
    destroy-on-close
    @update:open="(value) => emit('update:open', value)"
  >
    <a-alert
      v-if="loadingOptions"
      type="info"
      show-icon
      style="margin-bottom: 16px"
      message="正在刷新 Provider 相关选项..."
    />

    <a-form layout="vertical">
      <section class="provider-drawer__section">
        <div class="provider-drawer__section-title">基础信息</div>
        <div class="provider-drawer__grid">
          <a-form-item :label="t('llmProvider.providerForm.label.type')" required>
            <a-select
              v-model:value="formState.type"
              :disabled="Boolean(provider)"
              show-search
              :options="providerTypeOptions"
            />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.serviceName')" required>
            <a-input
              v-model:value="formState.name"
              :disabled="Boolean(provider)"
              show-count
              :maxlength="200"
            />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.protocol')" required>
            <a-select v-model:value="formState.protocol" :options="providerProtocolOptions" />
          </a-form-item>
          <a-form-item :label="t('serviceSource.serviceSourceForm.proxyName')">
            <a-select
              v-model:value="formState.proxyName"
              show-search
              :options="proxyOptions"
            />
            <div class="provider-drawer__field-extra">
              {{ t('serviceSource.serviceSourceForm.proxyServerLimitations') }}
            </div>
          </a-form-item>
        </div>
      </section>

      <section class="provider-drawer__section">
        <div class="provider-drawer__section-title">认证与凭证</div>

        <template v-if="tokenInputVisible">
          <div v-for="(token, index) in formState.tokens" :key="`token-${index}`" class="provider-drawer__inline-row">
            <a-form-item :label="index === 0 ? t('llmProvider.columns.tokens') : ' '" class="provider-drawer__inline-field" :required="index === 0">
              <a-input v-model:value="formState.tokens[index]" />
            </a-form-item>
            <a-button class="provider-drawer__icon-button" :disabled="formState.tokens.length <= 1" @click="removeToken(index)">
              <MinusCircleOutlined />
            </a-button>
          </div>

          <div class="provider-drawer__actions">
            <a-button @click="addToken">
              <PlusOutlined />
              添加凭证
            </a-button>
            <a-button type="link" @click="secretModalOpen = true">
              {{ t('llmProvider.providerForm.secretRefModal.entry') }}
            </a-button>
          </div>
        </template>

        <template v-else>
          <a-alert
            type="info"
            show-icon
            message="当前 Provider 使用专属认证配置，不需要单独填写 Tokens。"
          />
        </template>
      </section>

      <section v-if="formState.type" class="provider-drawer__section">
        <div class="provider-drawer__section-title">Provider 专属配置</div>

        <template v-if="isVolcengine">
          <div class="provider-drawer__grid">
            <a-form-item :label="t('llmProvider.providerForm.label.volcengineBaseUrl')" required class="provider-drawer__grid-span-2">
              <a-input
                v-model:value="formState.volcengineBaseUrl"
                :placeholder="t('llmProvider.providerForm.placeholder.volcengineBaseUrlPlaceholder')"
              />
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.volcengineClientRequestId')">
              <a-input
                v-model:value="formState.volcengineClientRequestId"
                :placeholder="t('llmProvider.providerForm.placeholder.volcengineClientRequestIdPlaceholder')"
              />
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.volcengineEnableEncryption')">
              <a-switch v-model:checked="formState.volcengineEnableEncryption" />
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.volcengineEnableTrace')" class="provider-drawer__grid-span-2">
              <a-switch v-model:checked="formState.volcengineEnableTrace" />
            </a-form-item>
          </div>

          <div class="provider-drawer__section-title provider-drawer__section-title--sub">{{ t('llmProvider.providerForm.sections.retryOnFailure') }}</div>
          <a-form-item :label="t('llmProvider.providerForm.label.retryOnFailureEnabled')">
            <a-switch v-model:checked="formState.retryOnFailureEnabled" />
          </a-form-item>
          <div v-if="formState.retryOnFailureEnabled" class="provider-drawer__grid provider-drawer__grid--compact">
            <a-form-item :label="t('llmProvider.providerForm.label.retryOnFailureMaxRetries')" required>
              <a-input-number v-model:value="formState.retryOnFailureMaxRetries" style="width: 100%" :min="1" />
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.retryOnFailureTimeout')" required>
              <a-input-number v-model:value="formState.retryOnFailureTimeout" style="width: 100%" :min="1" />
            </a-form-item>
            <a-form-item
              :label="t('llmProvider.providerForm.label.retryOnFailureStatusText')"
              :extra="t('llmProvider.providerForm.tooltips.retryOnFailureStatusTextTooltip')"
              class="provider-drawer__grid-span-2"
            >
              <a-input
                v-model:value="formState.retryOnFailureStatusText"
                :placeholder="t('llmProvider.providerForm.placeholder.retryOnFailureStatusTextPlaceholder')"
              />
            </a-form-item>
          </div>
        </template>

        <template v-if="formState.type === 'openai'">
          <div class="provider-drawer__grid">
            <a-form-item :label="t('llmProvider.providerForm.label.openaiServerType')" required>
              <a-select v-model:value="formState.openaiServerType">
                <a-select-option value="official">{{ t('llmProvider.providerForm.openaiServerType.official') }}</a-select-option>
                <a-select-option value="custom">{{ t('llmProvider.providerForm.openaiServerType.custom') }}</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item v-if="formState.openaiServerType === 'custom'" :label="t('llmProvider.providerForm.label.openaiCustomServerType')" required>
              <a-select v-model:value="formState.openaiCustomServerType">
                <a-select-option value="url">{{ t('llmProvider.providerForm.openaiCustomServerType.url') }}</a-select-option>
                <a-select-option value="service">{{ t('llmProvider.providerForm.openaiCustomServerType.service') }}</a-select-option>
              </a-select>
            </a-form-item>
          </div>

          <template v-if="openaiCustomUrlMode">
            <div v-for="(item, index) in formState.openaiCustomUrls" :key="`openai-url-${index}`" class="provider-drawer__inline-row">
              <a-form-item :label="index === 0 ? t('llmProvider.providerForm.label.openaiCustomUrl') : ' '" class="provider-drawer__inline-field" :required="index === 0">
                <a-input
                  v-model:value="formState.openaiCustomUrls[index]"
                  :placeholder="t('llmProvider.providerForm.placeholder.openaiCustomUrlPlaceholder')"
                />
              </a-form-item>
              <a-button class="provider-drawer__icon-button" :disabled="formState.openaiCustomUrls.length <= 1" @click="removeOpenAICustomUrl(index)">
                <MinusCircleOutlined />
              </a-button>
            </div>

            <div class="provider-drawer__actions">
              <a-button @click="addOpenAICustomUrl">
                <PlusOutlined />
                添加 URL
              </a-button>
            </div>
          </template>

          <div v-if="openaiCustomServiceMode" class="provider-drawer__grid">
            <a-form-item :label="t('llmProvider.providerForm.label.openaiCustomService')" required>
              <a-select
                v-model:value="formState.openaiCustomService"
                show-search
                :options="serviceOptions"
              />
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.openaiCustomServicePath')" required>
              <a-input
                v-model:value="formState.openaiCustomServicePath"
                :placeholder="t('llmProvider.providerForm.placeholder.openaiCustomServicePathPlaceholder')"
              />
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.openaiCustomServiceHost')">
              <a-input
                v-model:value="formState.openaiCustomServiceHost"
                :placeholder="t('llmProvider.providerForm.placeholder.openaiCustomServiceHostPlaceholder')"
              />
            </a-form-item>
          </div>
        </template>

        <template v-if="formState.type === 'qwen'">
          <div class="provider-drawer__grid">
            <a-form-item :label="t('llmProvider.providerForm.label.qwenEnableSearch')">
              <a-switch v-model:checked="formState.qwenEnableSearch" />
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.qwenEnableCompatible')" :extra="t('llmProvider.providerForm.tooltips.qwenEnableCompatibleTooltip')">
              <a-switch v-model:checked="formState.qwenEnableCompatible" />
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.qwenServerType')" required>
              <a-select v-model:value="formState.qwenServerType">
                <a-select-option value="official">{{ t('llmProvider.providerForm.qwenServerType.official') }}</a-select-option>
                <a-select-option value="custom">{{ t('llmProvider.providerForm.qwenServerType.custom') }}</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item v-if="qwenCustomMode" :label="t('llmProvider.providerForm.label.qwenDomain')" required>
              <a-input
                v-model:value="formState.qwenDomain"
                :placeholder="t('llmProvider.providerForm.placeholder.qwenDomainPlaceholder')"
              />
            </a-form-item>
          </div>

          <div v-for="(item, index) in formState.qwenFileIds" :key="`qwen-file-${index}`" class="provider-drawer__inline-row">
            <a-form-item
              :label="index === 0 ? t('llmProvider.providerForm.label.qwenFileIds') : ' '"
              class="provider-drawer__inline-field"
              :extra="index === 0 ? t('llmProvider.providerForm.tooltips.qwenFileIdsTooltip') : undefined"
            >
              <a-input
                v-model:value="formState.qwenFileIds[index]"
                :placeholder="t('llmProvider.providerForm.placeholder.qwenFileIdsPlaceholder')"
              />
            </a-form-item>
            <a-button class="provider-drawer__icon-button" @click="removeQwenFileId(index)">
              <MinusCircleOutlined />
            </a-button>
          </div>

          <div class="provider-drawer__actions">
            <a-button @click="addQwenFileId">
              <PlusOutlined />
              添加文件 ID
            </a-button>
          </div>
        </template>

        <div v-if="formState.type === 'azure'" class="provider-drawer__grid">
          <a-form-item :label="t('llmProvider.providerForm.label.azureServiceUrl')" required class="provider-drawer__grid-span-2">
            <a-input
              v-model:value="formState.azureServiceUrl"
              :placeholder="t('llmProvider.providerForm.placeholder.azureServiceUrlPlaceholder')"
            />
          </a-form-item>
        </div>

        <div v-if="formState.type === 'zhipuai'" class="provider-drawer__grid">
          <a-form-item :label="t('llmProvider.providerForm.label.zhipuDomain')" :extra="t('llmProvider.providerForm.tooltips.zhipuDomainTooltip')">
            <a-input
              v-model:value="formState.zhipuDomain"
              :placeholder="t('llmProvider.providerForm.placeholder.zhipuDomainPlaceholder')"
            />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.zhipuCodePlanMode')" :extra="t('llmProvider.providerForm.tooltips.zhipuCodePlanModeTooltip')">
            <a-switch v-model:checked="formState.zhipuCodePlanMode" />
          </a-form-item>
        </div>

        <div v-if="formState.type === 'claude'" class="provider-drawer__grid">
          <a-form-item :label="t('llmProvider.providerForm.label.claudeVersion')" :extra="t('llmProvider.providerForm.tooltips.claudeVersionTooltip')">
            <a-input v-model:value="formState.claudeVersion" placeholder="2023-06-01" />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.claudeCodeMode')" :extra="t('llmProvider.providerForm.tooltips.claudeCodeModeTooltip')">
            <a-switch v-model:checked="formState.claudeCodeMode" />
          </a-form-item>
        </div>

        <div v-if="formState.type === 'ollama'" class="provider-drawer__grid">
          <a-form-item :label="t('llmProvider.providerForm.label.ollamaServerHost')" required>
            <a-input
              v-model:value="formState.ollamaServerHost"
              :placeholder="t('llmProvider.providerForm.placeholder.ollamaServerHostPlaceholder')"
            />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.ollamaServerPort')" required>
            <a-input-number v-model:value="formState.ollamaServerPort" style="width: 100%" :min="1" :max="65535" />
          </a-form-item>
        </div>

        <div v-if="formState.type === 'bedrock'" class="provider-drawer__grid">
          <a-form-item :label="t('llmProvider.providerForm.label.awsRegion')" required>
            <a-auto-complete v-model:value="formState.awsRegion" :options="bedrockRegionOptions.map((value) => ({ value }))" />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.awsAccessKey')" required>
            <a-input v-model:value="formState.awsAccessKey" />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.awsSecretKey')" required class="provider-drawer__grid-span-2">
            <a-input-password v-model:value="formState.awsSecretKey" />
          </a-form-item>
        </div>

        <template v-if="formState.type === 'vertex'">
          <div class="provider-drawer__grid">
            <a-form-item label="认证方式" required>
              <a-select v-model:value="formState.vertexAuthMode">
                <a-select-option value="serviceAccount">Service Account Key</a-select-option>
                <a-select-option value="apiKey">API Key / Express Mode</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item :label="t('llmProvider.providerForm.label.vertexRegion')" :required="vertexServiceAccountMode">
              <a-auto-complete v-model:value="formState.vertexRegion" :options="vertexRegionOptions.map((value) => ({ value }))" />
            </a-form-item>
            <a-form-item v-if="vertexServiceAccountMode" :label="t('llmProvider.providerForm.label.vertexProjectId')" required>
              <a-input v-model:value="formState.vertexProjectId" />
            </a-form-item>
            <a-form-item v-if="vertexServiceAccountMode" :label="t('llmProvider.providerForm.label.vertexTokenRefreshAhead')" :extra="t('llmProvider.providerForm.tooltips.vertexTokenRefreshAheadTooltip')">
              <a-input-number v-model:value="formState.vertexTokenRefreshAhead" style="width: 100%" :min="0" :max="1800" />
            </a-form-item>
          </div>

          <a-form-item
            v-if="vertexServiceAccountMode"
            :label="t('llmProvider.providerForm.label.vertexAuthKey')"
            required
          >
            <a-textarea v-model:value="formState.vertexAuthKey" :rows="10" />
            <div class="provider-drawer__actions provider-drawer__actions--inline">
              <a-button type="link" @click="secretModalOpen = true">
                {{ t('llmProvider.providerForm.secretRefModal.entry') }}
              </a-button>
            </div>
          </a-form-item>

          <a-form-item :label="t('llmProvider.providerForm.label.geminiSafetySettings')">
            <div v-if="formState.geminiSafetySettings.length" class="provider-drawer__table">
              <div v-for="(item, index) in formState.geminiSafetySettings" :key="`safety-${index}`" class="provider-drawer__table-row">
                <a-auto-complete
                  v-model:value="formState.geminiSafetySettings[index].category"
                  :options="geminiSafetyCategoryOptions.map((value) => ({ value }))"
                  placeholder="Category"
                />
                <a-auto-complete
                  v-model:value="formState.geminiSafetySettings[index].threshold"
                  :options="geminiSafetyThresholdOptions.map((value) => ({ value }))"
                  placeholder="Threshold"
                />
                <a-button @click="removeGeminiSafetySetting(index)">
                  <MinusCircleOutlined />
                </a-button>
              </div>
            </div>
            <a-empty v-else :image="false" description="暂无安全设置" />
            <div class="provider-drawer__actions">
              <a-button @click="addGeminiSafetySetting">
                <PlusOutlined />
                {{ t('llmProvider.providerForm.addGeminiSafetySetting') }}
              </a-button>
            </div>
          </a-form-item>

          <a-alert
            v-if="vertexApiKeyMode"
            type="info"
            show-icon
            message="API Key / Express Mode 下只要求填写 Tokens，`vertexProjectId` 与 `vertexAuthKey` 可留空。"
          />
        </template>
      </section>

      <section class="provider-drawer__section">
        <div class="provider-drawer__section-title">令牌降级</div>
        <a-form-item
          :label="t('llmProvider.providerForm.label.failoverEnabled')"
          :extra="t('llmProvider.providerForm.label.failoverEnabledExtra')"
        >
          <a-switch v-model:checked="formState.failoverEnabled" />
        </a-form-item>

        <div v-if="formState.failoverEnabled" class="provider-drawer__grid provider-drawer__grid--compact">
          <a-form-item :label="t('llmProvider.providerForm.label.failureThreshold')" required>
            <a-input-number v-model:value="formState.failureThreshold" style="width: 100%" :min="1" />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.successThreshold')" required>
            <a-input-number v-model:value="formState.successThreshold" style="width: 100%" :min="1" />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.healthCheckInterval')" required>
            <a-input-number v-model:value="formState.healthCheckInterval" style="width: 100%" :min="1" />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.healthCheckTimeout')" required>
            <a-input-number v-model:value="formState.healthCheckTimeout" style="width: 100%" :min="1" />
          </a-form-item>
          <a-form-item :label="t('llmProvider.providerForm.label.healthCheckModel')" required class="provider-drawer__grid-span-2">
            <a-input v-model:value="formState.healthCheckModel" />
          </a-form-item>
        </div>
      </section>

      <section class="provider-drawer__section">
        <div class="provider-drawer__section-title">高级配置</div>
        <div class="provider-drawer__grid">
          <a-form-item v-if="!isVolcengine" label="Provider 域名覆写">
            <a-input v-model:value="formState.providerDomain" placeholder="例如：proxy.example.com" />
          </a-form-item>
          <a-form-item v-if="!isVolcengine" label="Provider Base Path">
            <a-input v-model:value="formState.providerBasePath" placeholder="例如：/v1" />
          </a-form-item>
          <a-form-item label="promoteThinkingOnEmpty">
            <a-switch v-model:checked="formState.promoteThinkingOnEmpty" />
          </a-form-item>
          <a-form-item label="hiclawMode">
            <a-switch v-model:checked="formState.hiclawMode" />
          </a-form-item>
        </div>
      </section>

      <section class="provider-drawer__section">
        <div class="provider-drawer__section-title">兼容扩展</div>
        <a-form-item label="rawConfigs 扩展 JSON">
          <a-textarea
            v-model:value="formState.extraRawConfigsJson"
            :rows="8"
            spellcheck="false"
          />
          <div class="provider-drawer__field-extra">
            这里会保留当前 UI 没直接建模的高级字段，避免编辑 Provider 时把历史配置意外丢掉。
          </div>
        </a-form-item>
      </section>
    </a-form>

    <DrawerFooter
      :loading="saving"
      @cancel="close"
      @confirm="submit"
    />

    <a-modal
      :open="secretModalOpen"
      :title="t('llmProvider.providerForm.secretRefModal.title')"
      @ok="secretModalOpen = false"
      @cancel="secretModalOpen = false"
    >
      <p>{{ t('llmProvider.providerForm.secretRefModal.content_brief') }}</p>
      <ul class="provider-drawer__secret-list">
        <li>
          <div>{{ t('llmProvider.providerForm.secretRefModal.content_sameNs') }}</div>
          <a-typography-text code>${secret.secret-name.field-name}</a-typography-text>
          <div>{{ t('llmProvider.providerForm.secretRefModal.example') }}</div>
          <a-typography-text code>${secret.my-token-secret.openai-token}</a-typography-text>
        </li>
        <li>
          <div>{{ t('llmProvider.providerForm.secretRefModal.content_diffNs') }}</div>
          <a-typography-text code>${secret.ns-name/secret-name.field-name}</a-typography-text>
          <div>{{ t('llmProvider.providerForm.secretRefModal.example') }}</div>
          <a-typography-text code>${secret.ai-ns/my-token-secret.openai-token}</a-typography-text>
        </li>
      </ul>
      <p>{{ t('llmProvider.providerForm.secretRefModal.roleConfig') }}</p>
    </a-modal>
  </a-drawer>
</template>

<style scoped>
.provider-drawer__section {
  margin-bottom: 20px;
  padding: 18px 18px 4px;
  border: 1px solid var(--portal-border);
  border-radius: 16px;
  background: linear-gradient(180deg, var(--portal-surface) 0%, var(--portal-surface-soft) 100%);
}

.provider-drawer__section-title {
  margin-bottom: 16px;
  font-size: 15px;
  font-weight: 600;
}

.provider-drawer__section-title--sub {
  margin-top: 8px;
}

.provider-drawer__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

.provider-drawer__grid--compact {
  gap: 0 12px;
}

.provider-drawer__grid-span-2 {
  grid-column: 1 / -1;
}

.provider-drawer__inline-row {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.provider-drawer__inline-field {
  flex: 1;
}

.provider-drawer__icon-button {
  margin-top: 30px;
  flex: none;
}

.provider-drawer__actions {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 12px;
}

.provider-drawer__actions--inline {
  margin-top: 8px;
  margin-bottom: 0;
}

.provider-drawer__field-extra {
  margin-top: 6px;
  font-size: 12px;
  color: var(--portal-text-soft);
}

.provider-drawer__table {
  display: grid;
  gap: 10px;
}

.provider-drawer__table-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  gap: 12px;
}

.provider-drawer__secret-list {
  padding-left: 18px;
}

.provider-drawer__secret-list li {
  margin-bottom: 14px;
}

@media (max-width: 960px) {
  .provider-drawer__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .provider-drawer__grid-span-2 {
    grid-column: auto;
  }

  .provider-drawer__inline-row {
    flex-direction: column;
    gap: 0;
  }

  .provider-drawer__icon-button {
    margin-top: 0;
  }

  .provider-drawer__table-row {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
