export const MODEL_ASSET_PRESET_TAGS = [
  '旗舰',
  '高性价比',
  '推理',
  '长上下文',
  '视觉',
  '多模态',
  '代码',
  'Embedding',
  '图像生成',
  '语音',
  '函数调用',
  '结构化输出',
] as const;

export type ModelAssetPresetTag = (typeof MODEL_ASSET_PRESET_TAGS)[number];
export type ModelType =
  | 'text'
  | 'multimodal'
  | 'image_generation'
  | 'video_generation'
  | 'speech_recognition'
  | 'speech_synthesis'
  | 'embedding';

export interface ProviderModelOption {
  modelId: string;
  targetModel: string;
  label: string;
}

export interface ProviderModelCatalog {
  providerName: string;
  models: ProviderModelOption[];
}

export interface PublishedBindingOption {
  assetId: string;
  bindingId: string;
  modelId: string;
  targetModel: string;
  displayLabel: string;
}

export interface PublishedBindingCatalog {
  providerName: string;
  bindings: PublishedBindingOption[];
}

export interface ModelAssetOptions {
  capabilities: {
    modelTypes: string[];
    inputModalities: string[];
    outputModalities: string[];
    featureFlags: string[];
    modalities: string[];
    features: string[];
    requestKinds: string[];
  };
  providerModels: ProviderModelCatalog[];
  publishedBindings: PublishedBindingCatalog[];
}

export interface ModelAssetCapabilities {
  inputModalities?: string[];
  outputModalities?: string[];
  featureFlags?: string[];
  modalities?: string[];
  features?: string[];
  requestKinds?: string[];
}

export interface ModelBindingPricing {
  currency?: 'CNY' | string;
  inputCostPerMillionTokens?: number;
  outputCostPerMillionTokens?: number;
  pricePerImage?: number;
  pricePerSecond?: number;
  pricePerSecond720p?: number;
  pricePerSecond1080p?: number;
  pricePer10kChars?: number;
  inputCostPerRequest?: number;
  cacheCreationInputTokenCostPerMillionTokens?: number;
  cacheCreationInputTokenCostAbove1hrPerMillionTokens?: number;
  cacheReadInputTokenCostPerMillionTokens?: number;
  inputCostPerMillionTokensAbove200kTokens?: number;
  outputCostPerMillionTokensAbove200kTokens?: number;
  cacheCreationInputTokenCostPerMillionTokensAbove200kTokens?: number;
  cacheReadInputTokenCostPerMillionTokensAbove200kTokens?: number;
  outputCostPerImage?: number;
  outputImageTokenCostPerMillionTokens?: number;
  inputCostPerImage?: number;
  inputImageTokenCostPerMillionTokens?: number;
  supportsPromptCaching?: boolean;
}

export interface ModelBindingLimits {
  maxInputTokens?: number;
  maxOutputTokens?: number;
  contextWindowTokens?: number;
  maxReasoningTokens?: number;
  maxInputTokensInReasoningMode?: number;
  maxOutputTokensInReasoningMode?: number;
  rpm?: number;
  tpm?: number;
  contextWindow?: number;
}

export interface ModelBindingPriceVersion {
  versionId: number;
  modelId: string;
  currency?: string;
  status?: string;
  active?: boolean;
  effectiveFrom?: string;
  effectiveTo?: string;
  createdAt?: string;
  updatedAt?: string;
  pricing?: ModelBindingPricing;
}

export interface ModelAssetBinding {
  bindingId: string;
  assetId?: string;
  modelId?: string;
  providerName?: string;
  targetModel?: string;
  protocol?: string;
  endpoint?: string;
  status?: string;
  publishedAt?: string;
  unpublishedAt?: string;
  createdAt?: string;
  updatedAt?: string;
  pricing?: ModelBindingPricing;
  limits?: ModelBindingLimits;
}

export interface ModelAsset {
  assetId: string;
  canonicalName?: string;
  displayName?: string;
  intro?: string;
  modelType?: ModelType | string;
  createdAt?: string;
  updatedAt?: string;
  tags?: string[];
  capabilities?: ModelAssetCapabilities;
  bindings?: ModelAssetBinding[];
}
