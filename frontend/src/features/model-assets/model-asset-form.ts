import type {
  ModelAsset,
  ModelAssetBinding,
  ModelAssetOptions,
  ModelBindingPricing,
  ModelType,
} from '@/interfaces/model-asset';
import type { AssetGrantRecord, OrgDepartmentNode } from '@/interfaces/org';
import { MODEL_ASSET_PRESET_TAGS } from '@/interfaces/model-asset';

export type AssetFormState = {
  assetId: string;
  canonicalName: string;
  displayName: string;
  intro: string;
  modelType: ModelType | '';
  tags: string[];
  inputModalities: string[];
  outputModalities: string[];
  featureFlags: string[];
};

export type BindingFormState = {
  selectedAssetId: string;
  bindingId: string;
  modelId: string;
  providerName: string;
  targetModel: string;
  protocol: string;
  endpoint: string;
  maxInputTokens?: number;
  maxOutputTokens?: number;
  contextWindowTokens?: number;
  maxReasoningTokens?: number;
  maxInputTokensInReasoningMode?: number;
  maxOutputTokensInReasoningMode?: number;
  rpm?: number;
  tpm?: number;
  currency: string;
  inputCostPerMillionTokens?: number;
  outputCostPerMillionTokens?: number;
  cacheCreationInputTokenCostPerMillionTokens?: number;
  cacheReadInputTokenCostPerMillionTokens?: number;
  pricePerImage?: number;
  pricePerSecond?: number;
  pricePerSecond720p?: number;
  pricePerSecond1080p?: number;
  pricePer10kChars?: number;
};

export const MODEL_TYPE_LABELS: Record<string, string> = {
  text: '文本模型',
  multimodal: '全模态模型',
  image_generation: '图片生成',
  video_generation: '视频生成',
  speech_recognition: '语音识别',
  speech_synthesis: '语音合成',
  embedding: '向量模型',
};

export const MODALITY_LABELS: Record<string, string> = {
  text: '文本',
  image: '图片',
  video: '视频',
  audio: '音频',
  embedding: '向量',
};

export const FEATURE_FLAG_LABELS: Record<string, string> = {
  reasoning: '深度思考',
  vision: '视觉理解',
  function_calling: 'Function Calling',
  structured_output: '结构化输出',
  web_search: '联网搜索',
  prefix_completion: '前缀续写',
  prompt_cache: 'Cache 缓存',
  batch_inference: '批量推理',
  fine_tuning: '模型调优',
  long_context: '长上下文',
  model_experience: '模型体验',
};

type PricingFieldName =
  | 'inputCostPerMillionTokens'
  | 'outputCostPerMillionTokens'
  | 'cacheCreationInputTokenCostPerMillionTokens'
  | 'cacheReadInputTokenCostPerMillionTokens'
  | 'pricePerImage'
  | 'pricePerSecond'
  | 'pricePerSecond720p'
  | 'pricePerSecond1080p'
  | 'pricePer10kChars';

type LimitFieldName =
  | 'maxInputTokens'
  | 'maxOutputTokens'
  | 'contextWindowTokens'
  | 'maxReasoningTokens'
  | 'maxInputTokensInReasoningMode'
  | 'maxOutputTokensInReasoningMode'
  | 'rpm'
  | 'tpm';

export const statusColorMap: Record<string, string> = {
  draft: 'default',
  published: 'green',
  unpublished: 'orange',
  active: 'green',
  inactive: 'default',
  disabled: 'red',
};

export function toAssetFormState(asset?: ModelAsset): AssetFormState {
  return {
    assetId: asset?.assetId || '',
    canonicalName: asset?.canonicalName || '',
    displayName: asset?.displayName || '',
    intro: asset?.intro || '',
    modelType: (asset?.modelType as ModelType | '') || '',
    tags: asset?.tags || [],
    inputModalities: asset?.capabilities?.inputModalities || asset?.capabilities?.modalities || [],
    outputModalities: asset?.capabilities?.outputModalities || asset?.capabilities?.modalities || [],
    featureFlags: asset?.capabilities?.featureFlags || asset?.capabilities?.features || [],
  };
}

export function toBindingFormState(binding?: ModelAssetBinding, selectedAssetId = ''): BindingFormState {
  const pricing = binding?.pricing || {};
  return {
    selectedAssetId: binding?.assetId || selectedAssetId,
    bindingId: binding?.bindingId || '',
    modelId: binding?.modelId || '',
    providerName: binding?.providerName || '',
    targetModel: binding?.targetModel || '',
    protocol: binding?.protocol || 'openai/v1',
    endpoint: binding?.endpoint || '',
    maxInputTokens: binding?.limits?.maxInputTokens,
    maxOutputTokens: binding?.limits?.maxOutputTokens,
    contextWindowTokens: binding?.limits?.contextWindowTokens ?? binding?.limits?.contextWindow,
    maxReasoningTokens: binding?.limits?.maxReasoningTokens,
    maxInputTokensInReasoningMode: binding?.limits?.maxInputTokensInReasoningMode,
    maxOutputTokensInReasoningMode: binding?.limits?.maxOutputTokensInReasoningMode,
    rpm: binding?.limits?.rpm,
    tpm: binding?.limits?.tpm,
    currency: pricing.currency || 'CNY',
    inputCostPerMillionTokens: pricing.inputCostPerMillionTokens,
    outputCostPerMillionTokens: pricing.outputCostPerMillionTokens,
    cacheCreationInputTokenCostPerMillionTokens: pricing.cacheCreationInputTokenCostPerMillionTokens,
    cacheReadInputTokenCostPerMillionTokens: pricing.cacheReadInputTokenCostPerMillionTokens,
    pricePerImage: pricing.pricePerImage ?? pricing.outputCostPerImage,
    pricePerSecond: pricing.pricePerSecond,
    pricePerSecond720p: pricing.pricePerSecond720p,
    pricePerSecond1080p: pricing.pricePerSecond1080p,
    pricePer10kChars: pricing.pricePer10kChars,
  };
}

export function applyDefaultPricingByModelType(values: BindingFormState, modelType?: string) {
  const normalizedType = modelType || '';
  if (normalizedType === 'image_generation' && typeof values.pricePerImage !== 'number') {
    values.pricePerImage = 0.2;
  }
  if (normalizedType === 'video_generation') {
    if (typeof values.pricePerSecond720p !== 'number') values.pricePerSecond720p = 0.6;
    if (typeof values.pricePerSecond1080p !== 'number') values.pricePerSecond1080p = 1;
  }
  if (normalizedType === 'speech_recognition' && typeof values.pricePerSecond !== 'number') {
    values.pricePerSecond = 0.00024;
  }
}

export function clearIrrelevantPricing(values: BindingFormState, modelType?: string) {
  const allowed = new Set(getPricingFieldsForType(modelType).map((item) => item.name));
  const allFields: PricingFieldName[] = [
    'inputCostPerMillionTokens',
    'outputCostPerMillionTokens',
    'cacheCreationInputTokenCostPerMillionTokens',
    'cacheReadInputTokenCostPerMillionTokens',
    'pricePerImage',
    'pricePerSecond',
    'pricePerSecond720p',
    'pricePerSecond1080p',
    'pricePer10kChars',
  ];
  allFields.forEach((field) => {
    if (!allowed.has(field)) {
      (values as unknown as Record<string, number | undefined>)[field] = undefined;
    }
  });
}

export function buildPricing(values: BindingFormState, modelType?: string): ModelBindingPricing {
  const pricing: ModelBindingPricing = {
    currency: values.currency || 'CNY',
  };
  getPricingFieldsForType(modelType).forEach((field) => {
    const value = values[field.name];
    if (typeof value === 'number') {
      (pricing as Record<string, number | string | undefined>)[field.name] = value;
    }
  });
  return pricing;
}

export function describePricing(pricing?: ModelBindingPricing, modelType?: string) {
  if (!pricing) {
    return '-';
  }
  const parts = getPricingFieldsForType(modelType).flatMap((field) => {
    const value = (pricing as Record<string, number | undefined>)[field.name];
    if (typeof value !== 'number') {
      return [];
    }
    return `${field.label} ${value} ${field.unit}`;
  });
  return parts.length ? parts.join(' / ') : '-';
}

export function describeCapabilities(asset?: ModelAsset | null) {
  if (!asset) {
    return '-';
  }
  const featureFlags = asset.capabilities?.featureFlags || asset.capabilities?.features || [];
  const inputModalities = asset.capabilities?.inputModalities || asset.capabilities?.modalities || [];
  const outputModalities = asset.capabilities?.outputModalities || asset.capabilities?.modalities || [];
  const parts: string[] = [];
  if (inputModalities.length) {
    parts.push(`输入 ${inputModalities.map((item) => MODALITY_LABELS[item] || item).join(' / ')}`);
  }
  if (outputModalities.length) {
    parts.push(`输出 ${outputModalities.map((item) => MODALITY_LABELS[item] || item).join(' / ')}`);
  }
  if (featureFlags.length) {
    parts.push(featureFlags.map((item) => FEATURE_FLAG_LABELS[item] || item).join(' / '));
  }
  return parts.join(' | ') || '-';
}

export function getPricingFieldsForType(modelType?: string) {
  switch (modelType) {
    case 'text':
    case 'multimodal':
      return [
        { name: 'inputCostPerMillionTokens' as const, label: '输入', unit: '元 / 百万 tokens' },
        { name: 'outputCostPerMillionTokens' as const, label: '输出', unit: '元 / 百万 tokens' },
        { name: 'cacheCreationInputTokenCostPerMillionTokens' as const, label: '显式缓存创建', unit: '元 / 百万 tokens' },
        { name: 'cacheReadInputTokenCostPerMillionTokens' as const, label: '显式缓存命中', unit: '元 / 百万 tokens' },
      ];
    case 'embedding':
      return [
        { name: 'inputCostPerMillionTokens' as const, label: '输入', unit: '元 / 百万 tokens' },
      ];
    case 'image_generation':
      return [
        { name: 'pricePerImage' as const, label: '图片生成', unit: '元 / 每张' },
      ];
    case 'video_generation':
      return [
        { name: 'pricePerSecond720p' as const, label: '视频生成（720P）', unit: '元 / 每秒' },
        { name: 'pricePerSecond1080p' as const, label: '视频生成（1080P）', unit: '元 / 每秒' },
      ];
    case 'speech_recognition':
      return [
        { name: 'pricePerSecond' as const, label: '语音识别', unit: '元 / 每秒' },
      ];
    case 'speech_synthesis':
      return [
        { name: 'pricePer10kChars' as const, label: '语音合成', unit: '元 / 每万字符' },
      ];
    default:
      return [];
  }
}

export function getLimitFieldsForType(modelType?: string) {
  const common: Array<{ name: LimitFieldName; label: string }> = [
    { name: 'rpm', label: 'RPM' },
    { name: 'tpm', label: 'TPM' },
  ];
  if (modelType === 'text' || modelType === 'multimodal' || modelType === 'embedding') {
    return [
      { name: 'maxInputTokens' as const, label: '最大输入长度' },
      { name: 'maxOutputTokens' as const, label: '最大输出长度' },
      { name: 'contextWindowTokens' as const, label: '上下文长度' },
      ...common,
      ...(modelType === 'text' || modelType === 'multimodal'
        ? [
            { name: 'maxInputTokensInReasoningMode' as const, label: '最大输入长度（思考模式）' },
            { name: 'maxOutputTokensInReasoningMode' as const, label: '最大输出长度（思考模式）' },
            { name: 'maxReasoningTokens' as const, label: '最大思维链长度' },
          ]
        : []),
    ];
  }
  return common;
}

export function hasLegacyAssetValues(asset: ModelAsset | undefined, assetOptions: ModelAssetOptions) {
  if (!asset) {
    return { tags: false, capabilities: false };
  }
  const capabilitySets = {
    inputModalities: new Set(assetOptions.capabilities?.inputModalities || []),
    outputModalities: new Set(assetOptions.capabilities?.outputModalities || []),
    featureFlags: new Set(assetOptions.capabilities?.featureFlags || []),
  };
  return {
    tags: !!asset.tags?.some((tag) => !MODEL_ASSET_PRESET_TAGS.includes(tag as any)),
    capabilities:
      !!asset.capabilities?.inputModalities?.some((item) => !capabilitySets.inputModalities.has(item))
      || !!asset.capabilities?.outputModalities?.some((item) => !capabilitySets.outputModalities.has(item))
      || !!asset.capabilities?.featureFlags?.some((item) => !capabilitySets.featureFlags.has(item)),
  };
}

export function flattenDepartmentOptions(nodes: OrgDepartmentNode[], level = 0): Array<{ label: string; value: string }> {
  return (nodes || []).flatMap((node) => {
    const prefix = level > 0 ? `${'  '.repeat(level)}- ` : '';
    return [
      { label: `${prefix}${node.name}`, value: node.departmentId },
      ...flattenDepartmentOptions(node.children || [], level + 1),
    ];
  });
}

export function splitGrantAssignments(grants: AssetGrantRecord[]) {
  return {
    consumers: (grants || []).filter((item) => item.subjectType === 'consumer').map((item) => item.subjectId || ''),
    departments: (grants || []).filter((item) => item.subjectType === 'department').map((item) => item.subjectId || ''),
    userLevels: (grants || []).filter((item) => item.subjectType === 'user_level').map((item) => item.subjectId || ''),
  };
}

export function buildGrantAssignments(bindingId: string, consumers: string[], departments: string[], userLevels: string[]) {
  const next: AssetGrantRecord[] = [];
  consumers.forEach((subjectId) => next.push({ assetType: 'model_binding', assetId: bindingId, subjectType: 'consumer', subjectId }));
  departments.forEach((subjectId) => next.push({ assetType: 'model_binding', assetId: bindingId, subjectType: 'department', subjectId }));
  userLevels.forEach((subjectId) => next.push({ assetType: 'model_binding', assetId: bindingId, subjectType: 'user_level', subjectId }));
  return next;
}
