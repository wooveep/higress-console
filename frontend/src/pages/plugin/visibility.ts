import { WasmPluginData } from '@/interfaces/wasm-plugin';
import { QueryType } from './utils';

type PluginIdentity = Partial<Pick<WasmPluginData, 'name' | 'key' | 'category'>>;

export const PluginVisibilityScope = {
  GLOBAL: 'global',
  ROUTE: QueryType.ROUTE,
  DOMAIN: QueryType.DOMAIN,
  AI_ROUTE: QueryType.AI_ROUTE,
} as const;

export const PluginVisibilityCategory = {
  ROUTE: 'route',
  AI: 'ai',
  AUTH: 'auth',
  SECURITY: 'security',
  TRAFFIC: 'traffic',
  TRANSFORM: 'transform',
  O11Y: 'o11y',
  CUSTOM: 'custom',
} as const;

export type PluginVisibilityScopeKey =
  typeof PluginVisibilityScope[keyof typeof PluginVisibilityScope];

export type PluginVisibilityCategoryKey =
  typeof PluginVisibilityCategory[keyof typeof PluginVisibilityCategory];

type PluginVisibilityCategoryConfig = Partial<Record<PluginVisibilityCategoryKey, readonly string[]>>;
type PluginVisibilityConfig = Partial<Record<PluginVisibilityScopeKey, PluginVisibilityCategoryConfig>>;

const PLUGIN_VISIBILITY_CATEGORIES: PluginVisibilityCategoryKey[] = [
  PluginVisibilityCategory.ROUTE,
  PluginVisibilityCategory.AI,
  PluginVisibilityCategory.AUTH,
  PluginVisibilityCategory.SECURITY,
  PluginVisibilityCategory.TRAFFIC,
  PluginVisibilityCategory.TRANSFORM,
  PluginVisibilityCategory.O11Y,
  PluginVisibilityCategory.CUSTOM,
];

// 白名单模式：
// 只有这里明确列出的插件才会在对应模块和分类中展示。
// 需要展示哪个插件，就把对应项取消注释。
// 当前核心管理分类是：AI / 认证 / 安全 / 流量 / 转换 / 可观测性。
export const VISIBLE_PLUGIN_NAMES_BY_SCOPE: PluginVisibilityConfig = {
  [PluginVisibilityScope.GLOBAL]: {
    [PluginVisibilityCategory.AI]: [
      // 'ai-agent',
      // 'ai-cache',
      'ai-data-masking',
      // 'ai-history',
      // 'ai-intent',
      // 'ai-json-resp',
      // 'ai-load-balancer',
      // 'ai-prompt-decorator',
      // 'ai-prompt-template',
      // 'ai-proxy',
      'ai-quota',
      // 'ai-rag',
      // 'ai-search',
      // 'ai-security-guard',
      'ai-statistics',
      // 'ai-token-ratelimit',
      // 'mcp-server',
      // 'model-mapper',
      // 'model-router',
    ],
    [PluginVisibilityCategory.AUTH]: [
      // 'basic-auth',
      // 'ext-auth',
      // 'hmac-auth',
      // 'jwt-auth',
      // 'key-auth',
      // 'oauth',
      // 'oidc',
      // 'opa',
    ],
    [PluginVisibilityCategory.SECURITY]: [
      // 'bot-detect',
      // 'cors',
      // 'ip-restriction',
      // 'request-block',
      // 'waf',
    ],
    [PluginVisibilityCategory.TRAFFIC]: [
      // 'cluster-key-rate-limit',
      // 'key-rate-limit',
      // 'request-validation',
      // 'traffic-tag',
    ],
    [PluginVisibilityCategory.TRANSFORM]: [
      // 'cache-control',
      // 'custom-response',
      // 'de-graphql',
      // 'frontend-gray',
      // 'transformer',
    ],
    [PluginVisibilityCategory.O11Y]: [
      // 'geo-ip',
    ],
    [PluginVisibilityCategory.CUSTOM]: [
      // 自定义插件按需取消注释
    ],
  },
  [PluginVisibilityScope.ROUTE]: {
    [PluginVisibilityCategory.ROUTE]: [
      // 'rewrite',
      // 'headerModify',
      // 'cors',
      // 'retries',
    ],
    [PluginVisibilityCategory.AI]: [
      // 'ai-agent',
      // 'ai-cache',
      'ai-data-masking',
      // 'ai-history',
      // 'ai-intent',
      // 'ai-json-resp',
      // 'ai-load-balancer',
      // 'ai-prompt-decorator',
      // 'ai-prompt-template',
      // 'ai-proxy',
      'ai-quota',
      // 'ai-rag',
      // 'ai-search',
      // 'ai-security-guard',
      'ai-statistics',
      // 'ai-token-ratelimit',
      // 'model-mapper',
      // 'model-router',
    ],
    [PluginVisibilityCategory.AUTH]: [
      // 'basic-auth',
      // 'ext-auth',
      // 'hmac-auth',
      // 'jwt-auth',
      // 'key-auth',
      // 'oauth',
      // 'oidc',
      // 'opa',
    ],
    [PluginVisibilityCategory.SECURITY]: [
      // 'bot-detect',
      // 'cors',
      // 'ip-restriction',
      // 'request-block',
      // 'waf',
    ],
    [PluginVisibilityCategory.TRAFFIC]: [
      // 'cluster-key-rate-limit',
      // 'key-rate-limit',
      // 'request-validation',
      // 'traffic-tag',
    ],
    [PluginVisibilityCategory.TRANSFORM]: [
      // 'cache-control',
      // 'custom-response',
      // 'de-graphql',
      // 'frontend-gray',
      // 'transformer',
    ],
    [PluginVisibilityCategory.O11Y]: [
      // 'geo-ip',
    ],
    [PluginVisibilityCategory.CUSTOM]: [
      // 自定义插件按需取消注释
    ],
  },
  [PluginVisibilityScope.DOMAIN]: {
    [PluginVisibilityCategory.AI]: [
      // 'ai-agent',
      // 'ai-cache',
      'ai-data-masking',
      // 'ai-history',
      // 'ai-intent',
      // 'ai-json-resp',
      // 'ai-load-balancer',
      // 'ai-prompt-decorator',
      // 'ai-prompt-template',
      // 'ai-proxy',
      'ai-quota',
      // 'ai-rag',
      // 'ai-search',
      // 'ai-security-guard',
      'ai-statistics',
      // 'ai-token-ratelimit',
      // 'model-mapper',
      // 'model-router',
    ],
    [PluginVisibilityCategory.AUTH]: [
      // 'basic-auth',
      // 'ext-auth',
      // 'hmac-auth',
      // 'jwt-auth',
      // 'key-auth',
      // 'oauth',
      // 'oidc',
      // 'opa',
    ],
    [PluginVisibilityCategory.SECURITY]: [
      // 'bot-detect',
      // 'cors',
      // 'ip-restriction',
      // 'request-block',
      // 'waf',
    ],
    [PluginVisibilityCategory.TRAFFIC]: [
      // 'cluster-key-rate-limit',
      // 'key-rate-limit',
      // 'request-validation',
      // 'traffic-tag',
    ],
    [PluginVisibilityCategory.TRANSFORM]: [
      // 'cache-control',
      // 'custom-response',
      // 'de-graphql',
      // 'frontend-gray',
      // 'transformer',
    ],
    [PluginVisibilityCategory.O11Y]: [
      // 'geo-ip',
    ],
    [PluginVisibilityCategory.CUSTOM]: [
      // 自定义插件按需取消注释
    ],
  },
  [PluginVisibilityScope.AI_ROUTE]: {
    [PluginVisibilityCategory.ROUTE]: [
      // 'headerModify',
      // 'cors',
      // 'retries',
    ],
    [PluginVisibilityCategory.AI]: [
      // 'ai-agent',
      // 'ai-cache',
      'ai-data-masking',
      // 'ai-history',
      // 'ai-intent',
      // 'ai-json-resp',
      // 'ai-load-balancer',
      // 'ai-prompt-decorator',
      // 'ai-prompt-template',
      // 'ai-proxy',
      'ai-quota',
      // 'ai-rag',
      // 'ai-search',
      // 'ai-security-guard',
      'ai-statistics',
      // 'ai-token-ratelimit',
      // 'mcp-server',
      // 'model-mapper',
      // 'model-router',
    ],
    [PluginVisibilityCategory.AUTH]: [
      // 'basic-auth',
      // 'ext-auth',
      // 'hmac-auth',
      // 'jwt-auth',
      // 'key-auth',
      // 'oauth',
      // 'oidc',
      // 'opa',
    ],
    [PluginVisibilityCategory.SECURITY]: [
      // 'bot-detect',
      // 'cors',
      // 'ip-restriction',
      // 'request-block',
      // 'waf',
    ],
    [PluginVisibilityCategory.TRAFFIC]: [
      // 'cluster-key-rate-limit',
      // 'key-rate-limit',
      // 'request-validation',
      // 'traffic-tag',
    ],
    [PluginVisibilityCategory.TRANSFORM]: [
      // 'cache-control',
      // 'custom-response',
      // 'de-graphql',
      // 'frontend-gray',
      // 'transformer',
    ],
    [PluginVisibilityCategory.O11Y]: [
      // 'geo-ip',
    ],
    [PluginVisibilityCategory.CUSTOM]: [
      // 自定义插件按需取消注释
    ],
  },
};

export function resolvePluginVisibilityScope(queryType?: string): PluginVisibilityScopeKey {
  if (queryType === QueryType.ROUTE) {
    return PluginVisibilityScope.ROUTE;
  }
  if (queryType === QueryType.DOMAIN) {
    return PluginVisibilityScope.DOMAIN;
  }
  if (queryType === QueryType.AI_ROUTE) {
    return PluginVisibilityScope.AI_ROUTE;
  }
  return PluginVisibilityScope.GLOBAL;
}

export function normalizePluginVisibilityCategory(category?: string): PluginVisibilityCategoryKey | null {
  if (!category) {
    return null;
  }
  return PLUGIN_VISIBILITY_CATEGORIES.includes(category as PluginVisibilityCategoryKey)
    ? category as PluginVisibilityCategoryKey
    : null;
}

export function getVisiblePluginNames(
  scope?: PluginVisibilityScopeKey,
  category?: PluginVisibilityCategoryKey | null,
): readonly string[] {
  if (!scope || !category) {
    return [];
  }
  return VISIBLE_PLUGIN_NAMES_BY_SCOPE[scope]?.[category] || [];
}

export function getVisiblePluginNamesByScope(scope?: PluginVisibilityScopeKey): readonly string[] {
  if (!scope) {
    return [];
  }
  const scopeConfig = VISIBLE_PLUGIN_NAMES_BY_SCOPE[scope];
  if (!scopeConfig) {
    return [];
  }
  const pluginNames: string[] = [];
  PLUGIN_VISIBILITY_CATEGORIES.forEach((category) => {
    const categoryPluginNames = scopeConfig[category] || [];
    categoryPluginNames.forEach((pluginName) => {
      if (!pluginNames.includes(pluginName)) {
        pluginNames.push(pluginName);
      }
    });
  });
  return pluginNames;
}

export function isPluginVisible(plugin: PluginIdentity, scope?: PluginVisibilityScopeKey): boolean {
  const pluginNames = [plugin.name, plugin.key].filter((pluginName): pluginName is string => Boolean(pluginName));
  if (!pluginNames.length) {
    return false;
  }
  const category = normalizePluginVisibilityCategory(plugin.category);
  const visiblePluginNames = category
    ? getVisiblePluginNames(scope, category)
    : getVisiblePluginNamesByScope(scope);
  if (!visiblePluginNames.length) {
    return false;
  }
  return pluginNames.some((pluginName) => visiblePluginNames.includes(pluginName));
}

export function filterVisiblePlugins<T extends PluginIdentity>(plugins: T[], scope?: PluginVisibilityScopeKey): T[] {
  return plugins.filter((plugin) => isPluginVisible(plugin, scope));
}
