import { WasmPluginData } from '@/interfaces/wasm-plugin';

type PluginIdentity = Partial<Pick<WasmPluginData, 'name' | 'key' | 'category'>>;

const DEDICATED_PLUGIN_PATHS: Record<string, string> = {
  'ai-quota': '/ai/quota',
  'ai-data-masking': '/ai/sensitive',
};

export const QueryType = {
  ROUTE: 'route',
  DOMAIN: 'domain',
  SERVICE: 'service',
  AI_ROUTE: 'aiRoute',
} as const;

export const PluginVisibilityScope = {
  GLOBAL: 'global',
  ROUTE: QueryType.ROUTE,
  DOMAIN: QueryType.DOMAIN,
  SERVICE: QueryType.SERVICE,
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

const VISIBLE_PLUGIN_NAMES_BY_SCOPE: PluginVisibilityConfig = {
  [PluginVisibilityScope.GLOBAL]: {
    [PluginVisibilityCategory.AI]: ['ai-statistics'],
    [PluginVisibilityCategory.AUTH]: [],
    [PluginVisibilityCategory.SECURITY]: [],
    [PluginVisibilityCategory.TRAFFIC]: [],
    [PluginVisibilityCategory.TRANSFORM]: [],
    [PluginVisibilityCategory.O11Y]: [],
    [PluginVisibilityCategory.CUSTOM]: [],
  },
  [PluginVisibilityScope.ROUTE]: {
    [PluginVisibilityCategory.ROUTE]: [],
    [PluginVisibilityCategory.AI]: ['ai-statistics'],
    [PluginVisibilityCategory.AUTH]: [],
    [PluginVisibilityCategory.SECURITY]: [],
    [PluginVisibilityCategory.TRAFFIC]: [],
    [PluginVisibilityCategory.TRANSFORM]: [],
    [PluginVisibilityCategory.O11Y]: [],
    [PluginVisibilityCategory.CUSTOM]: [],
  },
  [PluginVisibilityScope.DOMAIN]: {
    [PluginVisibilityCategory.AI]: ['ai-statistics'],
    [PluginVisibilityCategory.AUTH]: [],
    [PluginVisibilityCategory.SECURITY]: [],
    [PluginVisibilityCategory.TRAFFIC]: [],
    [PluginVisibilityCategory.TRANSFORM]: [],
    [PluginVisibilityCategory.O11Y]: [],
    [PluginVisibilityCategory.CUSTOM]: [],
  },
  [PluginVisibilityScope.SERVICE]: {
    [PluginVisibilityCategory.AI]: ['ai-statistics'],
    [PluginVisibilityCategory.AUTH]: [],
    [PluginVisibilityCategory.SECURITY]: [],
    [PluginVisibilityCategory.TRAFFIC]: [],
    [PluginVisibilityCategory.TRANSFORM]: [],
    [PluginVisibilityCategory.O11Y]: [],
    [PluginVisibilityCategory.CUSTOM]: [],
  },
  [PluginVisibilityScope.AI_ROUTE]: {
    [PluginVisibilityCategory.ROUTE]: [],
    [PluginVisibilityCategory.AI]: ['ai-statistics', 'ai-data-masking'],
    [PluginVisibilityCategory.AUTH]: [],
    [PluginVisibilityCategory.SECURITY]: [],
    [PluginVisibilityCategory.TRAFFIC]: [],
    [PluginVisibilityCategory.TRANSFORM]: [],
    [PluginVisibilityCategory.O11Y]: [],
    [PluginVisibilityCategory.CUSTOM]: [],
  },
};

export function resolveDedicatedPluginPath(pluginName?: string | null): string {
  const key = String(pluginName || '').trim();
  return key ? (DEDICATED_PLUGIN_PATHS[key] || '') : '';
}

export function buildPluginTargetPath(type?: string | null, name?: string | null) {
  if (!type || !name) {
    return '/plugin';
  }
  if (type === QueryType.ROUTE) {
    return `/route/config?type=route&name=${encodeURIComponent(name)}`;
  }
  if (type === QueryType.DOMAIN) {
    return `/domain/config?type=domain&name=${encodeURIComponent(name)}`;
  }
  if (type === QueryType.SERVICE) {
    return `/service/config?type=service&name=${encodeURIComponent(name)}`;
  }
  if (type === QueryType.AI_ROUTE) {
    return `/ai/route/config?type=aiRoute&name=${encodeURIComponent(name)}`;
  }
  return '/plugin';
}

export function resolvePluginVisibilityScope(queryType?: string): PluginVisibilityScopeKey {
  if (queryType === QueryType.ROUTE) {
    return PluginVisibilityScope.ROUTE;
  }
  if (queryType === QueryType.DOMAIN) {
    return PluginVisibilityScope.DOMAIN;
  }
  if (queryType === QueryType.SERVICE) {
    return PluginVisibilityScope.SERVICE;
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
