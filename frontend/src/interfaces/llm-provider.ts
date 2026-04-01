export interface LlmProvider {
  key?: string;
  name: string;
  type: string;
  protocol?: string;
  proxyName?: string;
  tokens: string[];
  tokenFailoverConfig?: TokeFailoverConfig;
  rawConfigs?: LlmProviderRawConfigs;
}

export interface TokeFailoverConfig {
  enabled?: boolean;
  failureThreshold?: number;
  successThreshold?: number;
  healthCheckInterval?: number;
  healthCheckTimeout?: number;
  healthCheckModel?: string;
}

export enum LlmProviderProtocol {
  OPENAI_V1 = 'openai/v1',
}

export interface LlmProviderRawConfigs {
  portalModelMeta?: PortalModelMeta;
  [prop: string]: any;
}

export interface PortalModelMeta {
  intro?: string;
  tags?: string[];
  capabilities?: {
    modalities?: string[];
    features?: string[];
  };
  pricing?: {
    currency?: 'CNY';
    input_cost_per_token?: number;
    output_cost_per_token?: number;
    input_cost_per_request?: number;
    cache_creation_input_token_cost?: number;
    cache_creation_input_token_cost_above_1hr?: number;
    cache_read_input_token_cost?: number;
    input_cost_per_token_above_200k_tokens?: number;
    output_cost_per_token_above_200k_tokens?: number;
    cache_creation_input_token_cost_above_200k_tokens?: number;
    cache_read_input_token_cost_above_200k_tokens?: number;
    output_cost_per_image?: number;
    output_cost_per_image_token?: number;
    input_cost_per_image?: number;
    input_cost_per_image_token?: number;
    supports_prompt_caching?: boolean;
    inputPer1K?: number;
    outputPer1K?: number;
  };
  limits?: {
    rpm?: number;
    tpm?: number;
    contextWindow?: number;
  };
}
