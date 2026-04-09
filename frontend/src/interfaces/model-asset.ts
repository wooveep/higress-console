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

export interface ProviderModelOption {
  modelId: string;
  targetModel: string;
  label: string;
}

export interface ProviderModelCatalog {
  providerName: string;
  models: ProviderModelOption[];
}

export interface ModelAssetOptions {
  capabilities: {
    modalities: string[];
    features: string[];
    requestKinds: string[];
  };
  providerModels: ProviderModelCatalog[];
}

export interface ModelAssetCapabilities {
  modalities?: string[];
  features?: string[];
  requestKinds?: string[];
}

export interface ModelBindingPricing {
  currency?: 'CNY' | string;
  inputCostPerToken?: number;
  outputCostPerToken?: number;
  inputCostPerRequest?: number;
  cacheCreationInputTokenCost?: number;
  cacheCreationInputTokenCostAbove1hr?: number;
  cacheReadInputTokenCost?: number;
  inputCostPerTokenAbove200kTokens?: number;
  outputCostPerTokenAbove200kTokens?: number;
  cacheCreationInputTokenCostAbove200kTokens?: number;
  cacheReadInputTokenCostAbove200kTokens?: number;
  outputCostPerImage?: number;
  outputCostPerImageToken?: number;
  inputCostPerImage?: number;
  inputCostPerImageToken?: number;
  supportsPromptCaching?: boolean;
}

export interface ModelBindingLimits {
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
  createdAt?: string;
  updatedAt?: string;
  tags?: string[];
  capabilities?: ModelAssetCapabilities;
  bindings?: ModelAssetBinding[];
}
