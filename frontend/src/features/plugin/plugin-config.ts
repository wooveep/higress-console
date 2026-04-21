import yaml from 'js-yaml';
import type { WasmPluginData } from '@/interfaces/wasm-plugin';

export const AI_DATA_MASKING_PLUGIN_NAME = 'ai-data-masking';

export const AI_DATA_MASKING_MANAGED_KEYS = [
  'deny_openai',
  'deny_jsonpath',
  'deny_raw',
  'deny_code',
  'deny_message',
  'deny_raw_message',
  'deny_content_type',
  'deny_words',
  'deny_rules',
  'replace_roles',
  'replace_rules',
  'audit_sink',
  'system_deny',
  'system_deny_words',
] as const;

export type SchemaNode = {
  type?: string;
  title?: string;
  description?: string;
  properties?: Record<string, SchemaNode>;
  items?: SchemaNode;
  required?: string[];
  enum?: Array<string | number | boolean>;
  [key: string]: any;
};

export function resolvePluginSchema(configData: any): SchemaNode | null {
  const schema = configData?.schema;
  if (!schema || typeof schema !== 'object') {
    return null;
  }
  if (schema.jsonSchema && typeof schema.jsonSchema === 'object') {
    return schema.jsonSchema as SchemaNode;
  }
  if (schema.openAPIV3Schema && typeof schema.openAPIV3Schema === 'object') {
    return schema.openAPIV3Schema as SchemaNode;
  }
  if (schema.type || schema.properties || schema.items) {
    return schema as SchemaNode;
  }
  return null;
}

export function cloneDeep<T>(value: T): T {
  return JSON.parse(JSON.stringify(value ?? null)) as T;
}

export function getLocalizedSchemaText(node: Record<string, any>, key: string, locale: string, fallback = '') {
  const i18nValue = node?.[`x-${key}-i18n`];
  return i18nValue?.[locale] || node?.[key] || fallback;
}

export function dumpYamlObject(value: unknown) {
  return yaml.dump(value ?? {}, {
    noRefs: true,
    lineWidth: 120,
    skipInvalid: true,
  });
}

export function parseYamlObject(value?: string) {
  if (!value?.trim()) {
    return {};
  }
  const parsed = yaml.load(value);
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return {};
  }
  return parsed as Record<string, any>;
}

export function getExampleRaw(configData: any, isGlobalAuthPlugin = false) {
  let raw = configData?.schema?.extensions?.['x-example-raw'] || '';
  if (isGlobalAuthPlugin && !raw) {
    raw = 'allow: []';
  }
  return raw;
}

export function omitManagedSchema(configData: any) {
  const schema = resolvePluginSchema(configData);
  if (!schema?.properties) {
    return configData;
  }
  const next = cloneDeep(configData);
  const targetSchema = resolvePluginSchema(next);
  if (!targetSchema?.properties) {
    return next;
  }
  AI_DATA_MASKING_MANAGED_KEYS.forEach((key) => {
    delete targetSchema.properties?.[key];
    if (Array.isArray(targetSchema.required)) {
      targetSchema.required = targetSchema.required.filter((item: string) => item !== key);
    }
  });
  return next;
}

export function omitAiDataMaskingManagedKeys(value: unknown) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return value;
  }
  const next = cloneDeep(value);
  AI_DATA_MASKING_MANAGED_KEYS.forEach((key) => {
    delete (next as Record<string, unknown>)[key];
  });
  return next;
}

export function createSchemaDefaultValue(schema?: SchemaNode): any {
  if (!schema) {
    return '';
  }
  if ('default' in schema) {
    return cloneDeep(schema.default);
  }
  if (Array.isArray(schema.enum) && schema.enum.length) {
    return schema.enum[0];
  }
  if (schema.type === 'object') {
    const result: Record<string, any> = {};
    Object.entries(schema.properties || {}).forEach(([key, child]) => {
      if ((schema.required || []).includes(key)) {
        result[key] = createSchemaDefaultValue(child);
      }
    });
    return result;
  }
  if (schema.type === 'array') {
    return [];
  }
  if (schema.type === 'boolean') {
    return false;
  }
  if (schema.type === 'integer' || schema.type === 'number') {
    return undefined;
  }
  return '';
}

export function sanitizeSchemaValue(value: any): any {
  if (Array.isArray(value)) {
    return value
      .map((item) => sanitizeSchemaValue(item))
      .filter((item) => !isSchemaValueEmpty(item));
  }
  if (value && typeof value === 'object') {
    const next: Record<string, any> = {};
    Object.entries(value).forEach(([key, child]) => {
      const sanitized = sanitizeSchemaValue(child);
      if (!isSchemaValueEmpty(sanitized)) {
        next[key] = sanitized;
      }
    });
    return next;
  }
  return value;
}

export function isSchemaValueEmpty(value: any) {
  if (value === undefined || value === null) {
    return true;
  }
  if (typeof value === 'string') {
    return !value.trim();
  }
  if (Array.isArray(value)) {
    return value.length === 0;
  }
  if (typeof value === 'object') {
    return Object.keys(value).length === 0;
  }
  return false;
}

export function validateSchemaValue(schema: SchemaNode | undefined, value: any, locale: string, path = ''): string[] {
  if (!schema?.properties) {
    return [];
  }
  const errors: string[] = [];
  const current = value && typeof value === 'object' ? value : {};
  (schema.required || []).forEach((key) => {
    if (isSchemaValueEmpty(current[key])) {
      const label = getLocalizedSchemaText(schema.properties?.[key] || {}, 'title', locale, key);
      errors.push(path ? `${path} / ${label}` : label);
    }
  });
  Object.entries(schema.properties).forEach(([key, child]) => {
    if (child.type === 'object' && current[key] && typeof current[key] === 'object') {
      const label = getLocalizedSchemaText(child, 'title', locale, key);
      errors.push(...validateSchemaValue(child, current[key], locale, path ? `${path} / ${label}` : label));
    }
  });
  return errors;
}

export function buildPluginImageUrl(record?: Partial<WasmPluginData>) {
  if (!record?.imageRepository) {
    return '';
  }
  return record.imageVersion ? `${record.imageRepository}:${record.imageVersion}` : record.imageRepository;
}

export function splitPluginImageUrl(imageUrl: string) {
  const trimmed = imageUrl.trim();
  const protocolIndex = trimmed.indexOf('://');
  const lastColonIndex = trimmed.lastIndexOf(':');
  const isOciImage = protocolIndex === -1 || trimmed.startsWith('oci://');
  if (trimmed && isOciImage && lastColonIndex > protocolIndex) {
    return {
      imageRepository: trimmed.substring(0, lastColonIndex),
      imageVersion: trimmed.substring(lastColonIndex + 1),
    };
  }
  return {
    imageRepository: trimmed,
    imageVersion: '',
  };
}
