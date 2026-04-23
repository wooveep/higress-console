import type { LlmProvider } from '@/interfaces/llm-provider';
import type { Service } from '@/interfaces/service';
import { stringifyPretty } from '@/lib/portal';

type Translate = (key: string, params?: Record<string, unknown>) => string;

export interface ProviderFormSafetySetting {
  category: string;
  threshold: string;
}

export interface ProviderFormState {
  name: string;
  type: string;
  protocol: string;
  proxyName: string;
  tokens: string[];
  volcengineBaseUrl: string;
  volcengineClientRequestId: string;
  volcengineEnableEncryption: boolean;
  volcengineEnableTrace: boolean;
  openaiServerType: 'official' | 'custom';
  openaiCustomServerType: 'url' | 'service';
  openaiCustomUrls: string[];
  openaiCustomService: string;
  openaiCustomServicePath: string;
  openaiCustomServiceHost: string;
  qwenEnableSearch: boolean;
  qwenEnableCompatible: boolean;
  qwenFileIds: string[];
  qwenServerType: 'official' | 'custom';
  qwenDomain: string;
  azureServiceUrl: string;
  zhipuDomain: string;
  zhipuCodePlanMode: boolean;
  claudeVersion: string;
  claudeCodeMode: boolean;
  ollamaServerHost: string;
  ollamaServerPort: number | null;
  awsRegion: string;
  awsAccessKey: string;
  awsSecretKey: string;
  vertexAuthMode: 'serviceAccount' | 'apiKey';
  vertexRegion: string;
  vertexProjectId: string;
  vertexAuthKey: string;
  vertexTokenRefreshAhead: number | null;
  geminiSafetySettings: ProviderFormSafetySetting[];
  failoverEnabled: boolean;
  failureThreshold: number;
  successThreshold: number;
  healthCheckInterval: number;
  healthCheckTimeout: number;
  healthCheckModel: string;
  retryOnFailureEnabled: boolean;
  retryOnFailureMaxRetries: number;
  retryOnFailureTimeout: number;
  retryOnFailureStatusText: string;
  providerDomain: string;
  providerBasePath: string;
  promoteThinkingOnEmpty: boolean;
  hiclawMode: boolean;
  extraRawConfigsJson: string;
}

export interface BuildProviderPayloadOptions {
  original?: Partial<LlmProvider> & Record<string, any> | null;
  servicesByDisplayName?: Map<string, Service>;
  t: Translate;
}

const SURFACED_RAW_CONFIG_KEYS = [
  'volcengineBaseUrl',
  'volcengineClientRequestId',
  'volcengineEnableEncryption',
  'volcengineEnableTrace',
  'retryOnFailure',
  'providerDomain',
  'providerBasePath',
  'baseUrl',
  'endpoint',
  'doubaoDomain',
  'promoteThinkingOnEmpty',
  'hiclawMode',
  'openaiCustomUrl',
  'openaiExtraCustomUrls',
  'openaiCustomServiceName',
  'openaiCustomServicePort',
  'azureServiceUrl',
  'qwenEnableSearch',
  'qwenEnableCompatible',
  'qwenFileIds',
  'qwenDomain',
  'zhipuDomain',
  'zhipuCodePlanMode',
  'claudeVersion',
  'claudeCodeMode',
  'ollamaServerHost',
  'ollamaServerPort',
  'awsRegion',
  'awsAccessKey',
  'awsSecretKey',
  'vertexRegion',
  'vertexProjectId',
  'vertexAuthKey',
  'vertexTokenRefreshAhead',
  'vertexAuthServiceName',
  'geminiSafetySetting',
  'geminiSafetySettings',
];

export const providerProtocolOptions = [
  { label: 'openai/v1', value: 'openai/v1' },
  { label: 'original', value: 'original' },
];

export const providerTypeOrder = [
  'openai',
  'azure',
  'qwen',
  'claude',
  'zhipuai',
  'vertex',
  'bedrock',
  'ollama',
  'moonshot',
  'deepseek',
  'volcengine',
  'openrouter',
  'grok',
  'groq',
  'github',
  'gemini',
  'baidu',
  'baichuan',
  'cohere',
  'coze',
  'mistral',
  'minimax',
  'stepfun',
  'yi',
  'ai360',
  'together-ai',
  'cloudflare',
  'deepl',
  'hunyuan',
  'spark',
];

export const bedrockRegionOptions = [
  'af-south-1',
  'ap-east-1',
  'ap-northeast-1',
  'ap-northeast-2',
  'ap-northeast-3',
  'ap-south-1',
  'ap-south-2',
  'ap-southeast-1',
  'ap-southeast-2',
  'ap-southeast-3',
  'ap-southeast-4',
  'ap-southeast-5',
  'ap-southeast-7',
  'ca-central-1',
  'ca-west-1',
  'eu-central-1',
  'eu-central-2',
  'eu-north-1',
  'eu-south-1',
  'eu-south-2',
  'eu-west-1',
  'eu-west-2',
  'eu-west-3',
  'il-central-1',
  'me-central-1',
  'me-south-1',
  'mx-central-1',
  'sa-east-1',
  'us-east-1',
  'us-east-2',
  'us-west-1',
  'us-west-2',
];

export const vertexRegionOptions = [
  'global',
  'africa-south1',
  'asia-east1',
  'asia-east2',
  'asia-northeast1',
  'asia-northeast2',
  'asia-northeast3',
  'asia-south1',
  'asia-southeast1',
  'asia-southeast2',
  'australia-southeast1',
  'australia-southeast2',
  'europe-central2',
  'europe-north1',
  'europe-southwest1',
  'europe-west1',
  'europe-west2',
  'europe-west3',
  'europe-west4',
  'europe-west6',
  'europe-west8',
  'europe-west9',
  'europe-west12',
  'me-central1',
  'me-central2',
  'me-west1',
  'northamerica-northeast1',
  'northamerica-northeast2',
  'southamerica-east1',
  'southamerica-west1',
  'us-central1',
  'us-east1',
  'us-east4',
  'us-east5',
  'us-south1',
  'us-west1',
  'us-west2',
  'us-west3',
  'us-west4',
];

export const geminiSafetyCategoryOptions = [
  'HARM_CATEGORY_HATE_SPEECH',
  'HARM_CATEGORY_DANGEROUS_CONTENT',
  'HARM_CATEGORY_HARASSMENT',
  'HARM_CATEGORY_SEXUALLY_EXPLICIT',
];

export const geminiSafetyThresholdOptions = [
  'HARM_BLOCK_THRESHOLD_UNSPECIFIED',
  'OFF',
  'BLOCK_NONE',
  'BLOCK_LOW_AND_ABOVE',
  'BLOCK_MEDIUM_AND_ABOVE',
  'BLOCK_ONLY_HIGH',
];

function compactStrings(values?: Array<string | number | null | undefined>) {
  return (values || [])
    .map((item) => String(item ?? '').trim())
    .filter(Boolean);
}

function toNumberOrNull(value: unknown) {
  if (value === null || value === undefined || value === '') {
    return null;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function asObject(value: unknown): Record<string, any> {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return { ...(value as Record<string, any>) };
  }
  return {};
}

function parseGeminiSafetySettings(rawConfigs: Record<string, any>) {
  const source = asObject(rawConfigs.geminiSafetySetting || rawConfigs.geminiSafetySettings);
  return Object.entries(source).map(([category, threshold]) => ({
    category,
    threshold: String(threshold ?? '').trim(),
  })).filter((item) => item.category && item.threshold);
}

function normalizeLegacyProviderType(value: unknown) {
  const normalized = String(value || '').trim();
  if (normalized === 'doubao') {
    return 'volcengine';
  }
  return normalized;
}

function normalizeUrlPathname(pathname: string) {
  const trimmed = pathname.trim();
  if (!trimmed || trimmed === '/') {
    return '';
  }
  const normalized = trimmed.replace(/\/+$/, '');
  return normalized.startsWith('/') ? normalized : `/${normalized}`;
}

function buildVolcengineBaseUrl(rawConfigs: Record<string, any>) {
  const explicit = String(rawConfigs.volcengineBaseUrl || rawConfigs.baseUrl || rawConfigs.endpoint || '').trim();
  if (explicit) {
    return explicit;
  }
  const providerDomain = String(rawConfigs.providerDomain || rawConfigs.doubaoDomain || '').trim();
  const providerBasePath = normalizeUrlPathname(String(rawConfigs.providerBasePath || '').trim()) || '/api/v3';
  if (!providerDomain) {
    return 'https://ark.cn-beijing.volces.com/api/v3';
  }
  return `https://${providerDomain}${providerBasePath}`;
}

function parseRetryOnFailure(rawConfigs: Record<string, any>) {
  const retryOnFailure = asObject(rawConfigs.retryOnFailure);
  const retryOnStatus = Array.isArray(retryOnFailure.retryOnStatus)
    ? compactStrings(retryOnFailure.retryOnStatus as Array<string | number | null | undefined>)
    : [];
  return {
    retryOnFailureEnabled: Boolean(retryOnFailure.enabled),
    retryOnFailureMaxRetries: toNumberOrNull(retryOnFailure.maxRetries) ?? 1,
    retryOnFailureTimeout: toNumberOrNull(retryOnFailure.retryTimeout) ?? 60000,
    retryOnFailureStatusText: retryOnStatus.join(', '),
  };
}

function omitSurfacedRawConfigs(rawConfigs: Record<string, any>) {
  const extra = { ...rawConfigs };
  for (const key of SURFACED_RAW_CONFIG_KEYS) {
    delete extra[key];
  }
  delete extra.type;
  return extra;
}

function parseOpenAIServiceFields(rawConfigs: Record<string, any>) {
  const serviceName = String(rawConfigs.openaiCustomServiceName || '').trim();
  const servicePort = toNumberOrNull(rawConfigs.openaiCustomServicePort);
  const customUrl = String(rawConfigs.openaiCustomUrl || '').trim();
  if (!serviceName || !customUrl) {
    return {
      openaiCustomService: '',
      openaiCustomServicePath: '',
      openaiCustomServiceHost: '',
    };
  }
  try {
    const url = new URL(customUrl);
    const displayValue = servicePort ? `${serviceName}:${servicePort}` : serviceName;
    const knownHosts = new Set([displayValue, serviceName]);
    return {
      openaiCustomService: displayValue,
      openaiCustomServicePath: url.pathname || '/',
      openaiCustomServiceHost: knownHosts.has(url.host) ? '' : url.host,
    };
  } catch {
    return {
      openaiCustomService: servicePort ? `${serviceName}:${servicePort}` : serviceName,
      openaiCustomServicePath: '/',
      openaiCustomServiceHost: '',
    };
  }
}

export function createProviderFormState(): ProviderFormState {
  return {
    name: '',
    type: '',
    protocol: 'openai/v1',
    proxyName: '',
    tokens: [''],
    volcengineBaseUrl: 'https://ark.cn-beijing.volces.com/api/v3',
    volcengineClientRequestId: '',
    volcengineEnableEncryption: false,
    volcengineEnableTrace: false,
    openaiServerType: 'official',
    openaiCustomServerType: 'url',
    openaiCustomUrls: [''],
    openaiCustomService: '',
    openaiCustomServicePath: '/',
    openaiCustomServiceHost: '',
    qwenEnableSearch: false,
    qwenEnableCompatible: true,
    qwenFileIds: [],
    qwenServerType: 'official',
    qwenDomain: '',
    azureServiceUrl: '',
    zhipuDomain: '',
    zhipuCodePlanMode: true,
    claudeVersion: '2023-06-01',
    claudeCodeMode: false,
    ollamaServerHost: '',
    ollamaServerPort: 11434,
    awsRegion: '',
    awsAccessKey: '',
    awsSecretKey: '',
    vertexAuthMode: 'serviceAccount',
    vertexRegion: '',
    vertexProjectId: '',
    vertexAuthKey: '',
    vertexTokenRefreshAhead: null,
    geminiSafetySettings: [],
    failoverEnabled: false,
    failureThreshold: 1,
    successThreshold: 1,
    healthCheckInterval: 5000,
    healthCheckTimeout: 10000,
    healthCheckModel: '',
    retryOnFailureEnabled: false,
    retryOnFailureMaxRetries: 1,
    retryOnFailureTimeout: 60000,
    retryOnFailureStatusText: '',
    providerDomain: '',
    providerBasePath: '',
    promoteThinkingOnEmpty: false,
    hiclawMode: false,
    extraRawConfigsJson: '{}',
  };
}

export function toProviderFormState(provider?: (Partial<LlmProvider> & Record<string, any>) | null) {
  const rawConfigs = asObject(provider?.rawConfigs);
  const extraRawConfigs = omitSurfacedRawConfigs(rawConfigs);
  const openaiCustomUrls = compactStrings([
    rawConfigs.openaiCustomUrl,
    ...compactStrings(Array.isArray(rawConfigs.openaiExtraCustomUrls) ? rawConfigs.openaiExtraCustomUrls : []),
  ]);
  const openaiServiceFields = parseOpenAIServiceFields(rawConfigs);
  const tokenFailoverConfig = asObject(provider?.tokenFailoverConfig);
  const tokens = compactStrings(provider?.tokens);
  const retryOnFailure = parseRetryOnFailure(rawConfigs);

  return {
    ...createProviderFormState(),
    name: String(provider?.name || ''),
    type: normalizeLegacyProviderType(provider?.type),
    protocol: String(provider?.protocol || 'openai/v1'),
    proxyName: String(provider?.proxyName || ''),
    tokens: tokens.length ? tokens : [''],
    volcengineBaseUrl: buildVolcengineBaseUrl(rawConfigs),
    volcengineClientRequestId: String(rawConfigs.volcengineClientRequestId || ''),
    volcengineEnableEncryption: Boolean(rawConfigs.volcengineEnableEncryption),
    volcengineEnableTrace: Boolean(rawConfigs.volcengineEnableTrace),
    openaiServerType: (openaiCustomUrls.length || rawConfigs.openaiCustomServiceName) ? 'custom' : 'official',
    openaiCustomServerType: rawConfigs.openaiCustomServiceName ? 'service' : 'url',
    openaiCustomUrls: openaiCustomUrls.length ? openaiCustomUrls : [''],
    openaiCustomService: openaiServiceFields.openaiCustomService,
    openaiCustomServicePath: openaiServiceFields.openaiCustomServicePath || '/',
    openaiCustomServiceHost: openaiServiceFields.openaiCustomServiceHost,
    qwenEnableSearch: Boolean(rawConfigs.qwenEnableSearch),
    qwenEnableCompatible: rawConfigs.qwenEnableCompatible === undefined ? true : Boolean(rawConfigs.qwenEnableCompatible),
    qwenFileIds: compactStrings(Array.isArray(rawConfigs.qwenFileIds) ? rawConfigs.qwenFileIds : []),
    qwenServerType: rawConfigs.qwenDomain ? 'custom' : 'official',
    qwenDomain: String(rawConfigs.qwenDomain || ''),
    azureServiceUrl: String(rawConfigs.azureServiceUrl || ''),
    zhipuDomain: String(rawConfigs.zhipuDomain || ''),
    zhipuCodePlanMode: rawConfigs.zhipuCodePlanMode === undefined ? true : Boolean(rawConfigs.zhipuCodePlanMode),
    claudeVersion: String(rawConfigs.claudeVersion || '2023-06-01'),
    claudeCodeMode: Boolean(rawConfigs.claudeCodeMode),
    ollamaServerHost: String(rawConfigs.ollamaServerHost || ''),
    ollamaServerPort: toNumberOrNull(rawConfigs.ollamaServerPort) ?? 11434,
    awsRegion: String(rawConfigs.awsRegion || ''),
    awsAccessKey: String(rawConfigs.awsAccessKey || ''),
    awsSecretKey: String(rawConfigs.awsSecretKey || ''),
    vertexAuthMode: tokens.length > 0 && !String(rawConfigs.vertexAuthKey || '').trim() ? 'apiKey' : 'serviceAccount',
    vertexRegion: String(rawConfigs.vertexRegion || ''),
    vertexProjectId: String(rawConfigs.vertexProjectId || ''),
    vertexAuthKey: String(rawConfigs.vertexAuthKey || ''),
    vertexTokenRefreshAhead: toNumberOrNull(rawConfigs.vertexTokenRefreshAhead),
    geminiSafetySettings: parseGeminiSafetySettings(rawConfigs),
    failoverEnabled: Boolean(tokenFailoverConfig.enabled),
    failureThreshold: toNumberOrNull(tokenFailoverConfig.failureThreshold) ?? 1,
    successThreshold: toNumberOrNull(tokenFailoverConfig.successThreshold) ?? 1,
    healthCheckInterval: toNumberOrNull(tokenFailoverConfig.healthCheckInterval) ?? 5000,
    healthCheckTimeout: toNumberOrNull(tokenFailoverConfig.healthCheckTimeout) ?? 10000,
    healthCheckModel: String(tokenFailoverConfig.healthCheckModel || ''),
    ...retryOnFailure,
    providerDomain: String(rawConfigs.providerDomain || ''),
    providerBasePath: String(rawConfigs.providerBasePath || ''),
    promoteThinkingOnEmpty: Boolean(rawConfigs.promoteThinkingOnEmpty),
    hiclawMode: Boolean(rawConfigs.hiclawMode),
    extraRawConfigsJson: stringifyPretty(extraRawConfigs),
  };
}

function ensureJsonObject(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }
  const parsed = JSON.parse(trimmed);
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('扩展配置必须是 JSON 对象');
  }
  return { ...(parsed as Record<string, any>) };
}

function validateOpenAICustomUrls(urls: string[], t: Translate) {
  let protocol = '';
  let path = '';
  for (const item of urls) {
    let parsed: URL;
    try {
      parsed = new URL(item);
    } catch {
      throw new Error(`${t('llmProvider.providerForm.rules.invalidOpenaiCustomUrl')}${item}`);
    }
    if (urls.length > 1 && !/^(\b25[0-5]|\b2[0-4][0-9]|\b[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}$/.test(parsed.hostname)) {
      throw new Error(t('llmProvider.providerForm.rules.openaiCustomUrlMultipleValuesWithIpOnly'));
    }
    if (protocol && parsed.protocol !== protocol) {
      throw new Error(t('llmProvider.providerForm.rules.openaiCustomUrlInconsistentProtocols'));
    }
    if (path && parsed.pathname !== path) {
      throw new Error(t('llmProvider.providerForm.rules.openaiCustomUrlInconsistentContextPaths'));
    }
    protocol = parsed.protocol;
    path = parsed.pathname;
  }
}

function validateVertexAuthKey(value: string, t: Translate) {
  const trimmed = value.trim();
  if (!trimmed) {
    throw new Error(t('llmProvider.providerForm.rules.vertexAuthKeyRequired'));
  }
  let parsed: Record<string, any>;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error(t('llmProvider.providerForm.rules.vertexAuthKeyBadFormat'));
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(t('llmProvider.providerForm.rules.vertexAuthKeyBadFormat'));
  }
  for (const key of ['client_email', 'private_key_id', 'private_key', 'token_uri']) {
    if (typeof parsed[key] !== 'string' || !parsed[key].trim()) {
      throw new Error(t('llmProvider.providerForm.rules.vertexAuthKeyBadRequiredProperty', { key }));
    }
  }
  return trimmed;
}

function buildGeminiSafetySetting(settings: ProviderFormSafetySetting[], t: Translate) {
  const result: Record<string, string> = {};
  for (const item of settings) {
    const category = item.category.trim();
    const threshold = item.threshold.trim();
    if (!category && !threshold) {
      continue;
    }
    if (!category) {
      throw new Error(t('llmProvider.providerForm.rules.geminiSafetyCategoryRequired'));
    }
    if (!threshold) {
      throw new Error(t('llmProvider.providerForm.rules.geminiSafetyThresholdRequired'));
    }
    if (result[category]) {
      throw new Error(t('llmProvider.providerForm.rules.geminiSafetyCategoryDuplicated'));
    }
    result[category] = threshold;
  }
  return result;
}

function parseVolcengineBaseUrl(value: string, t: Translate) {
  const trimmed = value.trim();
  if (!trimmed) {
    throw new Error(t('llmProvider.providerForm.rules.volcengineBaseUrlRequired'));
  }
  let parsed: URL;
  try {
    parsed = new URL(trimmed);
  } catch {
    throw new Error(t('llmProvider.providerForm.rules.volcengineBaseUrlRequired'));
  }
  if (!/^https?:$/.test(parsed.protocol)) {
    throw new Error(t('llmProvider.providerForm.rules.volcengineBaseUrlRequired'));
  }
  if (parsed.port) {
    throw new Error(t('llmProvider.providerForm.rules.volcengineBaseUrlNoPort'));
  }
  if (parsed.search || parsed.hash) {
    throw new Error(t('llmProvider.providerForm.rules.volcengineBaseUrlNoQuery'));
  }
  return {
    providerDomain: parsed.hostname,
    providerBasePath: normalizeUrlPathname(parsed.pathname) || '/api/v3',
  };
}

function parseRetryOnFailureStatusText(value: string) {
  return Array.from(new Set(
    value
      .split(/[\n,]/)
      .map((item) => item.trim())
      .filter(Boolean),
  ));
}

function isTokenRequired(state: ProviderFormState) {
  if (state.type === 'ollama' || state.type === 'bedrock') {
    return false;
  }
  if (state.type === 'vertex') {
    return state.vertexAuthMode === 'apiKey';
  }
  if (state.type === 'openai') {
    return state.openaiServerType === 'official';
  }
  return true;
}

export function shouldShowTokenInputs(state: ProviderFormState) {
  if (state.type === 'ollama' || state.type === 'bedrock') {
    return false;
  }
  if (state.type === 'vertex') {
    return state.vertexAuthMode === 'apiKey';
  }
  return true;
}

export function buildProviderPayload(state: ProviderFormState, options: BuildProviderPayloadOptions): LlmProvider & Record<string, any> {
  const { original, servicesByDisplayName, t } = options;
  const payloadName = state.name.trim();
  const providerType = normalizeLegacyProviderType(state.type);
  if (!providerType) {
    throw new Error(t('llmProvider.providerForm.rules.typeRequired'));
  }
  if (!payloadName) {
    throw new Error(t('llmProvider.providerForm.rules.serviceNameRequired'));
  }
  if (payloadName.includes('/')) {
    throw new Error(t('llmProvider.providerForm.rules.serviceNameNoSlash'));
  }
  if (!state.protocol.trim()) {
    throw new Error(t('llmProvider.providerForm.rules.protocol'));
  }

  const tokens = compactStrings(state.tokens);
  if (isTokenRequired(state) && !tokens.length) {
    throw new Error(t('llmProvider.providerForm.rules.tokenRequired'));
  }

  const rawConfigs = ensureJsonObject(state.extraRawConfigsJson);
  for (const key of SURFACED_RAW_CONFIG_KEYS) {
    delete rawConfigs[key];
  }
  delete rawConfigs.type;
  const providerDomain = state.providerDomain.trim();
  const providerBasePath = state.providerBasePath.trim();
  if (providerDomain) {
    rawConfigs.providerDomain = providerDomain;
  }
  if (providerBasePath) {
    rawConfigs.providerBasePath = providerBasePath;
  }
  if (state.promoteThinkingOnEmpty) {
    rawConfigs.promoteThinkingOnEmpty = true;
  }
  if (state.hiclawMode) {
    rawConfigs.hiclawMode = true;
    rawConfigs.promoteThinkingOnEmpty = true;
  }

  const payload: LlmProvider & Record<string, any> = {
    ...(original?.version != null ? { version: original.version } : {}),
    name: payloadName,
    type: providerType,
    protocol: state.protocol.trim(),
    proxyName: state.proxyName.trim() || undefined,
    tokens: shouldShowTokenInputs(state) ? tokens : [],
    rawConfigs,
  };

  switch (providerType) {
    case 'volcengine': {
      const volcengineBase = parseVolcengineBaseUrl(state.volcengineBaseUrl, t);
      rawConfigs.providerDomain = volcengineBase.providerDomain;
      rawConfigs.providerBasePath = volcengineBase.providerBasePath;
      if (state.volcengineClientRequestId.trim()) {
        rawConfigs.volcengineClientRequestId = state.volcengineClientRequestId.trim();
      } else {
        delete rawConfigs.volcengineClientRequestId;
      }
      if (state.volcengineEnableEncryption) {
        rawConfigs.volcengineEnableEncryption = true;
      } else {
        delete rawConfigs.volcengineEnableEncryption;
      }
      if (state.volcengineEnableTrace) {
        rawConfigs.volcengineEnableTrace = true;
      } else {
        delete rawConfigs.volcengineEnableTrace;
      }
      if (state.retryOnFailureEnabled) {
        if (!Number.isFinite(state.retryOnFailureMaxRetries) || state.retryOnFailureMaxRetries < 0) {
          throw new Error(t('llmProvider.providerForm.rules.retryOnFailureMaxRetriesRequired'));
        }
        if (!Number.isFinite(state.retryOnFailureTimeout) || state.retryOnFailureTimeout < 1) {
          throw new Error(t('llmProvider.providerForm.rules.retryOnFailureTimeoutRequired'));
        }
        const retryOnFailureStatus = parseRetryOnFailureStatusText(state.retryOnFailureStatusText);
        rawConfigs.retryOnFailure = {
          enabled: true,
          maxRetries: state.retryOnFailureMaxRetries,
          retryTimeout: state.retryOnFailureTimeout,
          ...(retryOnFailureStatus.length ? { retryOnStatus: retryOnFailureStatus } : {}),
        };
      } else {
        delete rawConfigs.retryOnFailure;
      }
      break;
    }
    case 'openai': {
      if (state.openaiServerType === 'custom') {
        if (state.openaiCustomServerType === 'service') {
          const serviceKey = state.openaiCustomService.trim();
          const selectedService = servicesByDisplayName?.get(serviceKey);
          if (!selectedService) {
            throw new Error(t('llmProvider.providerForm.rules.openaiCustomServiceRequired'));
          }
          const rawPath = state.openaiCustomServicePath.trim();
          if (!rawPath) {
            throw new Error(t('llmProvider.providerForm.rules.openaiCustomServicePathRequired'));
          }
          const path = rawPath.startsWith('/') ? rawPath : `/${rawPath}`;
          const serviceName = String(selectedService.name || '').trim();
          const servicePort = toNumberOrNull(selectedService.port) || 80;
          const protocol = String((selectedService as any).protocol || '').toUpperCase() === 'HTTPS' ? 'https' : 'http';
          const host = state.openaiCustomServiceHost.trim() || (servicePort ? `${serviceName}:${servicePort}` : serviceName);
          rawConfigs.openaiCustomUrl = `${protocol}://${host}${path}`;
          rawConfigs.openaiCustomServiceName = serviceName;
          rawConfigs.openaiCustomServicePort = servicePort;
          delete rawConfigs.openaiExtraCustomUrls;
        } else {
          const customUrls = compactStrings(state.openaiCustomUrls);
          if (!customUrls.length) {
            throw new Error(t('llmProvider.providerForm.rules.openaiCustomUrlRequired'));
          }
          validateOpenAICustomUrls(customUrls, t);
          rawConfigs.openaiCustomUrl = customUrls[0];
          if (customUrls.length > 1) {
            rawConfigs.openaiExtraCustomUrls = customUrls.slice(1);
          } else {
            delete rawConfigs.openaiExtraCustomUrls;
          }
          delete rawConfigs.openaiCustomServiceName;
          delete rawConfigs.openaiCustomServicePort;
        }
      }
      break;
    }
    case 'azure': {
      const azureServiceUrl = state.azureServiceUrl.trim();
      if (!azureServiceUrl) {
        throw new Error(t('llmProvider.providerForm.rules.azureServiceUrlRequired'));
      }
      rawConfigs.azureServiceUrl = azureServiceUrl;
      break;
    }
    case 'qwen': {
      rawConfigs.qwenEnableSearch = state.qwenEnableSearch;
      rawConfigs.qwenEnableCompatible = state.qwenEnableCompatible;
      const qwenFileIds = compactStrings(state.qwenFileIds);
      if (qwenFileIds.length) {
        rawConfigs.qwenFileIds = qwenFileIds;
      }
      if (state.qwenServerType === 'custom') {
        const qwenDomain = state.qwenDomain.trim();
        if (!qwenDomain) {
          throw new Error(t('llmProvider.providerForm.rules.qwenDomainRequired'));
        }
        rawConfigs.qwenDomain = qwenDomain;
      }
      break;
    }
    case 'zhipuai': {
      if (state.zhipuDomain.trim()) {
        rawConfigs.zhipuDomain = state.zhipuDomain.trim();
      }
      rawConfigs.zhipuCodePlanMode = state.zhipuCodePlanMode;
      break;
    }
    case 'claude': {
      if (state.claudeVersion.trim()) {
        rawConfigs.claudeVersion = state.claudeVersion.trim();
      }
      if (state.claudeCodeMode) {
        rawConfigs.claudeCodeMode = true;
      }
      break;
    }
    case 'ollama': {
      const host = state.ollamaServerHost.trim();
      if (!host) {
        throw new Error(t('llmProvider.providerForm.rules.ollamaServerHostRequired'));
      }
      if (!state.ollamaServerPort) {
        throw new Error(t('llmProvider.providerForm.rules.ollamaServerPortRequired'));
      }
      rawConfigs.ollamaServerHost = host;
      rawConfigs.ollamaServerPort = state.ollamaServerPort;
      payload.tokens = [];
      break;
    }
    case 'bedrock': {
      if (!state.awsRegion.trim()) {
        throw new Error(t('llmProvider.providerForm.rules.awsRegionRequired'));
      }
      if (!state.awsAccessKey.trim()) {
        throw new Error(t('llmProvider.providerForm.rules.awsAccessKeyRequired'));
      }
      if (!state.awsSecretKey.trim()) {
        throw new Error(t('llmProvider.providerForm.rules.awsSecretKeyRequired'));
      }
      rawConfigs.awsRegion = state.awsRegion.trim();
      rawConfigs.awsAccessKey = state.awsAccessKey.trim();
      rawConfigs.awsSecretKey = state.awsSecretKey.trim();
      payload.tokens = [];
      break;
    }
    case 'vertex': {
      if (state.vertexAuthMode === 'apiKey') {
        if (!tokens.length) {
          throw new Error(t('llmProvider.providerForm.rules.tokenRequired'));
        }
        if (state.vertexRegion.trim()) {
          rawConfigs.vertexRegion = state.vertexRegion.trim();
        }
        const geminiSafetySetting = buildGeminiSafetySetting(state.geminiSafetySettings, t);
        if (Object.keys(geminiSafetySetting).length) {
          rawConfigs.geminiSafetySetting = geminiSafetySetting;
        }
      } else {
        if (!state.vertexRegion.trim()) {
          throw new Error(t('llmProvider.providerForm.rules.vertexRegionRequired'));
        }
        if (!state.vertexProjectId.trim()) {
          throw new Error(t('llmProvider.providerForm.rules.vertexProjectIdRequired'));
        }
        rawConfigs.vertexRegion = state.vertexRegion.trim();
        rawConfigs.vertexProjectId = state.vertexProjectId.trim();
        rawConfigs.vertexAuthKey = validateVertexAuthKey(state.vertexAuthKey, t);
        if (state.vertexTokenRefreshAhead != null) {
          rawConfigs.vertexTokenRefreshAhead = state.vertexTokenRefreshAhead;
        }
        const geminiSafetySetting = buildGeminiSafetySetting(state.geminiSafetySettings, t);
        if (Object.keys(geminiSafetySetting).length) {
          rawConfigs.geminiSafetySetting = geminiSafetySetting;
        }
        payload.tokens = [];
      }
      break;
    }
    default:
      break;
  }

  if (state.failoverEnabled) {
    if (!state.failureThreshold) {
      throw new Error(t('llmProvider.providerForm.rules.failureThresholdRequired'));
    }
    if (!state.successThreshold) {
      throw new Error(t('llmProvider.providerForm.rules.successThresholdRequired'));
    }
    if (!state.healthCheckInterval) {
      throw new Error(t('llmProvider.providerForm.rules.healthCheckIntervalRequired'));
    }
    if (!state.healthCheckTimeout) {
      throw new Error(t('llmProvider.providerForm.rules.healthCheckTimeoutRequired'));
    }
    if (!state.healthCheckModel.trim()) {
      throw new Error(t('llmProvider.providerForm.rules.healthCheckModelRequired'));
    }
    payload.tokenFailoverConfig = {
      enabled: true,
      failureThreshold: state.failureThreshold,
      successThreshold: state.successThreshold,
      healthCheckInterval: state.healthCheckInterval,
      healthCheckTimeout: state.healthCheckTimeout,
      healthCheckModel: state.healthCheckModel.trim(),
    };
  }

  return payload;
}

export function getProviderTypeLabel(type: string, t: Translate) {
  const key = `llmProvider.providerTypes.${type}`;
  const translated = t(key);
  if (translated && translated !== key) {
    return translated;
  }
  if (type === 'vertex') {
    return 'Vertex AI / Gemini';
  }
  if (type === 'bedrock') {
    return 'AWS Bedrock';
  }
  if (type === 'volcengine') {
    return 'Volcengine Ark';
  }
  return type;
}

export function getProviderTypeOptions(t: Translate) {
  return providerTypeOrder.map((value) => ({
    value,
    label: getProviderTypeLabel(value, t),
  }));
}

export function getProviderCredentialValues(provider: Partial<LlmProvider> & Record<string, any>) {
  const tokens = compactStrings(provider.tokens);
  if (tokens.length) {
    return tokens;
  }
  const rawConfigs = asObject(provider.rawConfigs);
  if (provider.type === 'bedrock') {
    const accessKey = String(rawConfigs.awsAccessKey || '').trim();
    const secretKey = String(rawConfigs.awsSecretKey || '').trim();
    return accessKey && secretKey ? [`${accessKey}:${secretKey}`] : [];
  }
  if (provider.type === 'vertex') {
    const authKey = String(rawConfigs.vertexAuthKey || '').trim();
    if (!authKey) {
      return [];
    }
    try {
      const parsed = JSON.parse(authKey);
      const clientEmail = String(parsed?.client_email || '').trim();
      const privateKeyId = String(parsed?.private_key_id || '').trim();
      return clientEmail && privateKeyId ? [`${clientEmail}:${privateKeyId}`] : [];
    } catch {
      return [];
    }
  }
  return [];
}
