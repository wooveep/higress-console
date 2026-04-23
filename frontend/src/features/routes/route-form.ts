import type { AiRoute, AiRouteFallbackConfig, AiUpstream } from '@/interfaces/ai-route';
import type { ModelAssetOptions, PublishedBindingOption } from '@/interfaces/model-asset';
import type { OrgDepartmentNode } from '@/interfaces/org';
import type { AuthConfig, KeyedRoutePredicate, Route, RoutePredicate, UpstreamService } from '@/interfaces/route';
import type { Service } from '@/interfaces/service';
import { serviceToString } from '@/interfaces/service';
import { flattenDepartmentOptions } from '@/features/model-assets/model-asset-form';

export const routeMatchTypeOptions = [
  { label: '前缀匹配', value: 'PRE' },
  { label: '精确匹配', value: 'EQUAL' },
  { label: '正则匹配', value: 'REGULAR' },
] as const;

export const routeMethodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'].map((value) => ({
  label: value,
  value,
}));

export const userLevelOptions = ['normal', 'plus', 'pro', 'ultra'].map((value) => ({
  label: value,
  value,
}));

export const fallbackResponseCodeOptions = ['4xx', '5xx'].map((value) => ({
  label: value,
  value,
}));

export type RouteAuthFormState = {
  enabled: boolean;
  allowedDepartments: string[];
  allowedConsumerLevels: string[];
};

export type KeyedPredicateFormItem = {
  id: string;
  key: string;
  matchType: string;
  matchValue: string;
};

export type ModelPredicateFormItem = {
  id: string;
  matchType: string;
  matchValue: string;
};

export type RouteServiceFormItem = {
  id: string;
  serviceKey: string;
  weight?: number;
};

export type AiUpstreamFormItem = {
  id: string;
  provider: string;
  weight: number;
  modelMapping: string;
};

export type GatewayRouteFormState = {
  name: string;
  domains: string[];
  methods: string[];
  pathMatchType: string;
  pathMatchValue: string;
  headers: KeyedPredicateFormItem[];
  urlParams: KeyedPredicateFormItem[];
  services: RouteServiceFormItem[];
  auth: RouteAuthFormState;
};

export type AiRouteFormState = {
  name: string;
  domains: string[];
  methods: string[];
  pathMatchType: string;
  pathMatchValue: string;
  pathIgnoreCase: boolean;
  headerPredicates: KeyedPredicateFormItem[];
  urlParamPredicates: KeyedPredicateFormItem[];
  modelPredicates: ModelPredicateFormItem[];
  upstreams: AiUpstreamFormItem[];
  auth: RouteAuthFormState;
  fallback: {
    enabled: boolean;
    responseCodes: string[];
    provider: string;
    modelMapping: string;
  };
};

export function createRowId() {
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export function createRouteAuthState(authConfig?: AuthConfig, enabledDefault = false): RouteAuthFormState {
  return {
    enabled: Boolean(authConfig?.enabled ?? enabledDefault),
    allowedDepartments: [...(authConfig?.allowedDepartments || [])],
    allowedConsumerLevels: [...(authConfig?.allowedConsumerLevels || [])],
  };
}

export function buildAuthConfig(state: RouteAuthFormState, required: boolean): AuthConfig | undefined {
  const enabled = required || state.enabled;
  const allowedDepartments = uniqueStrings(state.allowedDepartments);
  const allowedConsumerLevels = uniqueStrings(state.allowedConsumerLevels);

  if (!enabled && !required) {
    return undefined;
  }
  if (allowedDepartments.length === 0 && allowedConsumerLevels.length === 0) {
    throw new Error(required ? 'AI 路由必须至少选择一个部门或用户等级。' : '启用请求认证后，请至少选择一个部门或用户等级。');
  }
  return {
    enabled: true,
    allowedDepartments,
    allowedConsumerLevels,
  };
}

export function createKeyedPredicateFormItem(predicate?: Partial<KeyedRoutePredicate>): KeyedPredicateFormItem {
  return {
    id: createRowId(),
    key: String(predicate?.key || ''),
    matchType: String(predicate?.matchType || 'EQUAL'),
    matchValue: String(predicate?.matchValue || ''),
  };
}

export function createModelPredicateFormItem(predicate?: Partial<RoutePredicate>): ModelPredicateFormItem {
  return {
    id: createRowId(),
    matchType: String(predicate?.matchType || 'EQUAL'),
    matchValue: String(predicate?.matchValue || ''),
  };
}

export function createRouteServiceFormItem(service?: Partial<UpstreamService>, servicesByKey?: Map<string, Service>): RouteServiceFormItem {
  const fallbackKey = service && service.name ? (service.port != null ? `${service.name}:${service.port}` : service.name) : '';
  return {
    id: createRowId(),
    serviceKey: resolveExistingServiceKey(service, servicesByKey) || fallbackKey,
    weight: typeof service?.weight === 'number' ? service.weight : 100,
  };
}

export function createAiUpstreamFormItem(upstream?: AiUpstream): AiUpstreamFormItem {
  return {
    id: createRowId(),
    provider: String(upstream?.provider || ''),
    weight: typeof upstream?.weight === 'number' ? upstream.weight : 100,
    modelMapping: formatAiModelMapping(upstream?.modelMapping),
  };
}

export function toGatewayRouteFormState(route?: Route, servicesByKey?: Map<string, Service>): GatewayRouteFormState {
  return {
    name: route?.name || '',
    domains: [...(route?.domains || [])],
    methods: [...(route?.methods || ['GET', 'POST'])],
    pathMatchType: route?.path?.matchType || 'PRE',
    pathMatchValue: route?.path?.matchValue || '/',
    headers: (route?.headers || []).map((item) => createKeyedPredicateFormItem(item)),
    urlParams: (route?.urlParams || []).map((item) => createKeyedPredicateFormItem(item)),
    services: (route?.services || []).map((item) => createRouteServiceFormItem(item, servicesByKey)),
    auth: createRouteAuthState(route?.authConfig, false),
  };
}

export function buildGatewayRoutePayload(
  state: GatewayRouteFormState,
  currentRoute: Route | null | undefined,
  servicesByKey: Map<string, Service>,
): Route {
  const services = buildGatewayServices(state.services, servicesByKey);
  if (!state.name.trim()) {
    throw new Error('请输入路由名称。');
  }
  if (!state.pathMatchValue.trim()) {
    throw new Error('请输入路径匹配值。');
  }
  if (services.length === 0) {
    throw new Error('请至少选择一个目标服务。');
  }
  return {
    ...(currentRoute || {}),
    name: state.name.trim(),
    domains: uniqueStrings(state.domains),
    methods: uniqueStrings(state.methods),
    path: {
      matchType: state.pathMatchType,
      matchValue: state.pathMatchValue.trim(),
    },
    headers: buildKeyedPredicates(state.headers),
    urlParams: buildKeyedPredicates(state.urlParams),
    services,
    authConfig: buildAuthConfig(state.auth, false),
  };
}

export function toAiRouteFormState(route?: AiRoute): AiRouteFormState {
  const fallbackConfig = route?.fallbackConfig;
  return {
    name: route?.name || '',
    domains: [...(route?.domains || [])],
    methods: [...(route?.methods || [])],
    pathMatchType: route?.pathPredicate?.matchType || 'PRE',
    pathMatchValue: route?.pathPredicate?.matchValue || '/',
    pathIgnoreCase: route?.pathPredicate?.caseSensitive === false,
    headerPredicates: (route?.headerPredicates || []).map((item) => createKeyedPredicateFormItem(item)),
    urlParamPredicates: (route?.urlParamPredicates || []).map((item) => createKeyedPredicateFormItem(item)),
    modelPredicates: (route?.modelPredicates || []).map((item) => createModelPredicateFormItem(item)),
    upstreams: (route?.upstreams || []).map((item) => createAiUpstreamFormItem(item)),
    auth: createRouteAuthState(route?.authConfig, true),
    fallback: {
      enabled: Boolean(fallbackConfig?.enabled),
      responseCodes: [...(fallbackConfig?.responseCodes || ['4xx', '5xx'])],
      provider: String(fallbackConfig?.upstreams?.[0]?.provider || ''),
      modelMapping: formatAiModelMapping(fallbackConfig?.upstreams?.[0]?.modelMapping),
    },
  };
}

export function buildAiRoutePayload(state: AiRouteFormState, currentRoute?: AiRoute | null): AiRoute {
  if (!state.name.trim()) {
    throw new Error('请输入 AI 路由名称。');
  }
  if (!state.pathMatchValue.trim()) {
    throw new Error('请输入路径匹配值。');
  }
  const modelPredicates = buildModelPredicates(state.modelPredicates);
  if (modelPredicates.length === 0) {
    throw new Error('请至少配置一条模型匹配规则。');
  }
  const upstreams = buildAiUpstreams(state.upstreams, true);
  if (upstreams.length === 0) {
    throw new Error('请至少配置一个目标 AI 服务。');
  }
  const totalWeight = upstreams.reduce((sum, item) => sum + Number(item.weight || 0), 0);
  if (totalWeight !== 100) {
    throw new Error('目标 AI 服务的权重总和必须为 100。');
  }

  const fallbackConfig = buildAiFallbackConfig(state.fallback);
  return {
    ...(currentRoute || {}),
    name: state.name.trim(),
    domains: uniqueStrings(state.domains),
    methods: uniqueStrings(state.methods),
    pathPredicate: {
      matchType: state.pathMatchType,
      matchValue: state.pathMatchValue.trim(),
      caseSensitive: !state.pathIgnoreCase,
    },
    headerPredicates: buildKeyedPredicates(state.headerPredicates),
    urlParamPredicates: buildKeyedPredicates(state.urlParamPredicates),
    modelPredicates,
    upstreams,
    authConfig: buildAuthConfig(state.auth, true),
    fallbackConfig,
  };
}

export function getDepartmentOptions(nodes: OrgDepartmentNode[]) {
  return flattenDepartmentOptions(nodes || []);
}

export function buildServiceOptions(services: Service[], currentServices: RouteServiceFormItem[]) {
  const options = [...services]
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((item) => ({ label: serviceToString(item), value: serviceToString(item) }));
  currentServices.forEach((item) => {
    const value = item.serviceKey.trim();
    if (value && !options.some((option) => option.value === value)) {
      options.unshift({ label: `${value}（历史值）`, value });
    }
  });
  return options;
}

export function buildPublishedBindingCatalog(assetOptions: ModelAssetOptions) {
  return (assetOptions.publishedBindings || []).reduce<Record<string, PublishedBindingOption[]>>((accumulator, item) => {
    accumulator[item.providerName] = item.bindings || [];
    return accumulator;
  }, {});
}

export function getTargetModelOptions(
  providerName: string,
  providerModelCatalog: Record<string, PublishedBindingOption[]>,
  currentValue?: string,
) {
  const options = (providerModelCatalog[providerName] || []).map((item) => ({
    label: item.displayLabel || (item.targetModel === item.modelId ? item.targetModel : `${item.modelId} / ${item.targetModel}`),
    value: item.targetModel,
  }));
  if (currentValue && !options.some((item) => item.value === currentValue)) {
    options.unshift({ label: `历史值 / ${currentValue}`, value: currentValue });
  }
  return options;
}

export function providerHasBoundModels(providerName: string, providerModelCatalog: Record<string, PublishedBindingOption[]>) {
  return (providerModelCatalog[providerName] || []).length > 0;
}

export function getAiRouteLegacyIssue(route?: AiRoute | null) {
  if (!route) {
    return null;
  }
  const fallbackUpstreams = route.fallbackConfig?.upstreams || [];
  if (fallbackUpstreams.length > 1) {
    return '当前路由包含多个 fallback 上游，现有表单只支持编辑单个 fallback 服务。';
  }
  const fallbackStrategy = String(route.fallbackConfig?.fallbackStrategy || route.fallbackConfig?.strategy || '').toUpperCase();
  if (fallbackStrategy && !['SEQ', 'SEQUENCE', 'RAND', 'RANDOM'].includes(fallbackStrategy)) {
    return `当前路由使用 fallback 策略 ${fallbackStrategy}，现有表单只支持单服务顺序降级。`;
  }
  return null;
}

function buildGatewayServices(items: RouteServiceFormItem[], servicesByKey: Map<string, Service>): UpstreamService[] {
  const result: UpstreamService[] = [];
  items.forEach((item) => {
    const key = item.serviceKey.trim();
    if (!key) {
      return;
    }
    const matched = servicesByKey.get(key);
    if (matched) {
      result.push({
        name: matched.name,
        port: matched.port,
        weight: item.weight,
      });
      return;
    }
    const [name, portText] = key.split(':');
    result.push({
      name: name.trim(),
      port: portText ? Number(portText) : undefined,
      weight: item.weight,
    });
  });
  return result;
}

function buildKeyedPredicates(items: KeyedPredicateFormItem[]): KeyedRoutePredicate[] {
  return items
    .map((item) => ({
      key: item.key.trim(),
      matchType: item.matchType,
      matchValue: item.matchValue.trim(),
    }))
    .filter((item) => item.key && item.matchValue);
}

function buildModelPredicates(items: ModelPredicateFormItem[]): RoutePredicate[] {
  return items
    .map((item) => ({
      matchType: item.matchType,
      matchValue: item.matchValue.trim(),
    }))
    .filter((item) => item.matchValue);
}

function buildAiUpstreams(items: AiUpstreamFormItem[], requireWeight: boolean): AiUpstream[] {
  return items
    .map((item) => {
      const provider = item.provider.trim();
      if (!provider) {
        return null;
      }
      const payload: AiUpstream = {
        provider,
      };
      if (requireWeight) {
        payload.weight = Number(item.weight || 0);
      }
      const modelMapping = parseAiModelMapping(item.modelMapping);
      if (Object.keys(modelMapping).length > 0) {
        payload.modelMapping = modelMapping;
      }
      return payload;
    })
    .filter(Boolean) as AiUpstream[];
}

function buildAiFallbackConfig(state: AiRouteFormState['fallback']): AiRouteFallbackConfig {
  const enabled = Boolean(state.enabled);
  const payload: AiRouteFallbackConfig = {
    enabled,
    upstreams: [],
  };
  if (enabled) {
    const provider = state.provider.trim();
    if (!provider) {
      throw new Error('启用 fallback 后，请选择一个 fallback 服务。');
    }
    const modelMapping = parseAiModelMapping(state.modelMapping);
    if (Object.keys(modelMapping).length === 0) {
      throw new Error('启用 fallback 后，请填写 fallback 目标模型。');
    }
    payload.upstreams = [{
      provider,
      modelMapping,
    }];
    payload.responseCodes = uniqueStrings(state.responseCodes);
    if (payload.responseCodes.length === 0) {
      throw new Error('启用 fallback 后，请至少选择一个触发状态码。');
    }
    payload.fallbackStrategy = 'SEQ';
    payload.strategy = 'SEQ';
  }
  return payload;
}

function resolveExistingServiceKey(service?: Partial<UpstreamService>, servicesByKey?: Map<string, Service>) {
  if (!service?.name || !servicesByKey) {
    return '';
  }
  for (const [key, value] of servicesByKey.entries()) {
    if (value.name === service.name && Number(value.port || 0) === Number(service.port || 0)) {
      return key;
    }
  }
  return '';
}

function uniqueStrings(items: Array<string | undefined>) {
  return Array.from(new Set((items || []).map((item) => String(item || '').trim()).filter(Boolean)));
}

function formatAiModelMapping(modelMapping?: Record<string, string>) {
  if (!modelMapping) {
    return '';
  }
  const entries = Object.entries(modelMapping);
  if (entries.length === 0) {
    return '';
  }
  if (entries.length === 1 && entries[0][0] === '*') {
    return entries[0][1];
  }
  return entries
    .filter(([key]) => key)
    .map(([key, value]) => `${key}=${value}`)
    .join(';');
}

function parseAiModelMapping(value?: string) {
  const text = String(value || '').trim();
  if (!text) {
    return {};
  }
  if (!text.includes(';') && !text.includes('=')) {
    return { '*': text };
  }
  return text.split(';').reduce<Record<string, string>>((accumulator, pair) => {
    const [rawKey, rawValue] = pair.split('=', 2);
    const key = String(rawKey || '').trim();
    const mappedValue = String(rawValue || '').trim();
    if (key) {
      accumulator[key] = mappedValue;
    }
    return accumulator;
  }, {});
}
