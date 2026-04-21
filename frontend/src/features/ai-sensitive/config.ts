import type {
  AiSensitiveDetectRule,
  AiSensitiveReplaceRule,
  AiSensitiveRuntimeConfig,
} from '@/interfaces/ai-sensitive';

export type AiSensitiveRuntimeSettings = {
  denyOpenai: boolean;
  denyJsonpathText: string;
  denyRaw: boolean;
  denyCode: number;
  denyMessage: string;
  denyRawMessage: string;
  denyContentType: string;
  auditSinkServiceName: string;
  auditSinkNamespace: string;
  auditSinkPort?: number;
  auditSinkPath: string;
  auditSinkTimeoutMs: number;
};

export type AiSensitiveDetectPreset = {
  key: string;
  labelKey: string;
  descriptionKey: string;
  pattern: string;
  matchType: 'contains' | 'exact' | 'regex';
  priority?: number;
};

export type AiSensitiveReplacePreset = {
  key: string;
  labelKey: string;
  descriptionKey: string;
  pattern: string;
  replaceType: 'replace' | 'hash';
  replaceValue?: string;
  restore?: boolean;
  priority?: number;
};

export type AiSensitiveJsonpathPreset = {
  key: string;
  labelKey: string;
  descriptionKey: string;
  values: string[];
};

export const AI_SENSITIVE_DETECT_PRESETS: AiSensitiveDetectPreset[] = [
  {
    key: 'apiKey',
    labelKey: 'aiSensitive.detectPreset.apiKey.label',
    descriptionKey: 'aiSensitive.detectPreset.apiKey.description',
    pattern: 'sk-[0-9a-zA-Z]*',
    matchType: 'regex',
    priority: 100,
  },
  {
    key: 'mobilePhone',
    labelKey: 'aiSensitive.detectPreset.mobilePhone.label',
    descriptionKey: 'aiSensitive.detectPreset.mobilePhone.description',
    pattern: '%{MOBILE}',
    matchType: 'regex',
    priority: 90,
  },
  {
    key: 'email',
    labelKey: 'aiSensitive.detectPreset.email.label',
    descriptionKey: 'aiSensitive.detectPreset.email.description',
    pattern: '%{EMAILLOCALPART}@%{HOSTNAME:domain}',
    matchType: 'regex',
    priority: 80,
  },
  {
    key: 'idCard',
    labelKey: 'aiSensitive.detectPreset.idCard.label',
    descriptionKey: 'aiSensitive.detectPreset.idCard.description',
    pattern: '%{IDCARD}',
    matchType: 'regex',
    priority: 80,
  },
];

export const AI_SENSITIVE_REPLACE_PRESETS: AiSensitiveReplacePreset[] = [
  {
    key: 'mobilePhone',
    labelKey: 'aiSensitive.replacePreset.mobilePhone.label',
    descriptionKey: 'aiSensitive.replacePreset.mobilePhone.description',
    pattern: '%{MOBILE}',
    replaceType: 'replace',
    replaceValue: '****',
    priority: 90,
  },
  {
    key: 'email',
    labelKey: 'aiSensitive.replacePreset.email.label',
    descriptionKey: 'aiSensitive.replacePreset.email.description',
    pattern: '%{EMAILLOCALPART}@%{HOSTNAME:domain}',
    replaceType: 'replace',
    replaceValue: '****@$domain',
    restore: true,
    priority: 80,
  },
  {
    key: 'bankCard',
    labelKey: 'aiSensitive.replacePreset.bankCard.label',
    descriptionKey: 'aiSensitive.replacePreset.bankCard.description',
    pattern: '%{BANKCARD}',
    replaceType: 'replace',
    replaceValue: '****',
    priority: 80,
  },
  {
    key: 'ip',
    labelKey: 'aiSensitive.replacePreset.ip.label',
    descriptionKey: 'aiSensitive.replacePreset.ip.description',
    pattern: '%{IP}',
    replaceType: 'replace',
    replaceValue: '***.***.***.***',
    restore: true,
    priority: 80,
  },
  {
    key: 'idCard',
    labelKey: 'aiSensitive.replacePreset.idCard.label',
    descriptionKey: 'aiSensitive.replacePreset.idCard.description',
    pattern: '%{IDCARD}',
    replaceType: 'replace',
    replaceValue: '****',
    priority: 80,
  },
  {
    key: 'apiKey',
    labelKey: 'aiSensitive.replacePreset.apiKey.label',
    descriptionKey: 'aiSensitive.replacePreset.apiKey.description',
    pattern: 'sk-[0-9a-zA-Z]*',
    replaceType: 'hash',
    restore: true,
    priority: 100,
  },
];

export const AI_SENSITIVE_JSONPATH_PRESETS: AiSensitiveJsonpathPreset[] = [
  {
    key: 'messagesContent',
    labelKey: 'aiSensitive.jsonpathPreset.messagesContent.label',
    descriptionKey: 'aiSensitive.jsonpathPreset.messagesContent.description',
    values: ['$.messages[*].content'],
  },
  {
    key: 'userMessages',
    labelKey: 'aiSensitive.jsonpathPreset.userMessages.label',
    descriptionKey: 'aiSensitive.jsonpathPreset.userMessages.description',
    values: ['$.messages[?(@.role=="user")].content'],
  },
  {
    key: 'systemPrompt',
    labelKey: 'aiSensitive.jsonpathPreset.systemPrompt.label',
    descriptionKey: 'aiSensitive.jsonpathPreset.systemPrompt.description',
    values: ['$.messages[?(@.role=="system")].content', '$.system'],
  },
  {
    key: 'responsesInput',
    labelKey: 'aiSensitive.jsonpathPreset.responsesInput.label',
    descriptionKey: 'aiSensitive.jsonpathPreset.responsesInput.description',
    values: ['$.input[*].content[*].text'],
  },
];

export function createDefaultAiSensitiveRuntimeSettings(): AiSensitiveRuntimeSettings {
  return {
    denyOpenai: true,
    denyJsonpathText: '$.messages[*].content',
    denyRaw: false,
    denyCode: 200,
    denyMessage: '提问或回答中包含敏感词，已被屏蔽',
    denyRawMessage: '{"errmsg":"提问或回答中包含敏感词，已被屏蔽"}',
    denyContentType: 'application/json',
    auditSinkServiceName: '',
    auditSinkNamespace: '',
    auditSinkPort: undefined,
    auditSinkPath: '',
    auditSinkTimeoutMs: 2000,
  };
}

export function splitMultilineValues(value?: string) {
  const seen = new Set<string>();
  return String(value || '')
    .split('\n')
    .map((item) => item.trim())
    .filter((item) => {
      if (!item || seen.has(item)) {
        return false;
      }
      seen.add(item);
      return true;
    });
}

export function parseAiSensitiveRuntimeSettings(
  config?: AiSensitiveRuntimeConfig | null,
): AiSensitiveRuntimeSettings {
  const defaults = createDefaultAiSensitiveRuntimeSettings();
  return {
    denyOpenai: config?.denyOpenai ?? defaults.denyOpenai,
    denyJsonpathText: normalizeStringArray(config?.denyJsonpath).join('\n') || defaults.denyJsonpathText,
    denyRaw: config?.denyRaw ?? defaults.denyRaw,
    denyCode: Number(config?.denyCode ?? defaults.denyCode),
    denyMessage: String(config?.denyMessage || defaults.denyMessage),
    denyRawMessage: String(config?.denyRawMessage || defaults.denyRawMessage),
    denyContentType: String(config?.denyContentType || defaults.denyContentType),
    auditSinkServiceName: String(config?.auditSink?.serviceName || ''),
    auditSinkNamespace: String(config?.auditSink?.namespace || ''),
    auditSinkPort: config?.auditSink?.port ? Number(config.auditSink.port) : undefined,
    auditSinkPath: String(config?.auditSink?.path || ''),
    auditSinkTimeoutMs: Number(config?.auditSink?.timeoutMs ?? defaults.auditSinkTimeoutMs),
  };
}

export function buildAiSensitiveRuntimeConfig(
  settings: AiSensitiveRuntimeSettings,
): AiSensitiveRuntimeConfig {
  const auditSink = sanitizeAuditSink({
    serviceName: settings.auditSinkServiceName,
    namespace: settings.auditSinkNamespace,
    port: settings.auditSinkPort,
    path: settings.auditSinkPath,
    timeoutMs: settings.auditSinkTimeoutMs,
  });

  return sanitizeObject({
    denyOpenai: settings.denyOpenai,
    denyJsonpath: splitMultilineValues(settings.denyJsonpathText),
    denyRaw: settings.denyRaw,
    denyCode: settings.denyCode,
    denyMessage: settings.denyMessage,
    denyRawMessage: settings.denyRawMessage,
    denyContentType: settings.denyContentType,
    auditSink,
  }) as AiSensitiveRuntimeConfig;
}

export function listSelectedJsonpathPresetKeys(value?: string) {
  const selected = new Set(splitMultilineValues(value));
  return AI_SENSITIVE_JSONPATH_PRESETS
    .filter((item) => item.values.every((entry) => selected.has(entry)))
    .map((item) => item.key);
}

export function applyJsonpathPresetKeys(keys: string[]) {
  const values = keys.flatMap((key) => (
    AI_SENSITIVE_JSONPATH_PRESETS.find((item) => item.key === key)?.values || []
  ));
  return splitMultilineValues(values.join('\n')).join('\n');
}

export function serializeDetectRules(rules: AiSensitiveDetectRule[]) {
  return rules
    .filter((item) => String(item.pattern || '').trim())
    .map((item) => ({
      pattern: item.pattern.trim(),
      match_type: item.matchType || 'contains',
      description: item.description || undefined,
      priority: Number(item.priority || 0),
      enabled: item.enabled !== false,
    }));
}

export function serializeReplaceRules(rules: AiSensitiveReplaceRule[]) {
  const normalized = rules
    .filter((item) => String(item.pattern || '').trim())
    .map((item) => ({
      pattern: item.pattern.trim(),
      replace_type: item.replaceType || 'replace',
      replace_value: item.replaceValue || undefined,
      restore: Boolean(item.restore),
      description: item.description || undefined,
      priority: Number(item.priority || 0),
      enabled: item.enabled !== false,
    }));
  return {
    replaceRoles: normalized.map((item) => ({
      regex: item.pattern,
      type: item.replace_type,
      restore: item.restore,
      value: item.replace_value,
    })),
    replaceRules: normalized,
  };
}

function normalizeStringArray(value: unknown) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item || '').trim()).filter(Boolean);
  }
  if (typeof value === 'string') {
    return splitMultilineValues(value);
  }
  return [];
}

function sanitizeAuditSink(value: Record<string, any>) {
  const next = sanitizeObject(value);
  return Object.keys(next).length ? next : undefined;
}

function sanitizeObject(value: Record<string, any>) {
  const next: Record<string, any> = {};
  Object.entries(value).forEach(([key, current]) => {
    if (Array.isArray(current)) {
      if (current.length) {
        next[key] = current;
      }
      return;
    }
    if (current && typeof current === 'object') {
      const nested = sanitizeObject(current);
      if (Object.keys(nested).length) {
        next[key] = nested;
      }
      return;
    }
    if (current === undefined || current === null || current === '') {
      return;
    }
    next[key] = current;
  });
  return next;
}
