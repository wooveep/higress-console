import { ModelAsset, ModelAssetBinding, ModelAssetOptions, ModelBindingPriceVersion } from '@/interfaces/model-asset';
import request from './request';

export const getModelAssets = (): Promise<ModelAsset[]> => {
  return request.get<any, ModelAsset[]>('/v1/ai/model-assets');
};

export const getModelAsset = (assetId: string): Promise<ModelAsset> => {
  return request.get<any, ModelAsset>(`/v1/ai/model-assets/${assetId}`);
};

export const getModelAssetOptions = (): Promise<ModelAssetOptions> => {
  return request.get<any, ModelAssetOptions>('/v1/ai/model-assets/options');
};

export const createModelAsset = (payload: ModelAsset): Promise<ModelAsset> => {
  return request.post<any, ModelAsset>('/v1/ai/model-assets', payload);
};

export const updateModelAsset = (assetId: string, payload: ModelAsset): Promise<ModelAsset> => {
  return request.put<any, ModelAsset>(`/v1/ai/model-assets/${assetId}`, payload);
};

export const createModelBinding = (assetId: string, payload: ModelAssetBinding): Promise<ModelAssetBinding> => {
  return request.post<any, ModelAssetBinding>(`/v1/ai/model-assets/${assetId}/bindings`, payload);
};

export const updateModelBinding = (
  assetId: string,
  bindingId: string,
  payload: ModelAssetBinding,
): Promise<ModelAssetBinding> => {
  return request.put<any, ModelAssetBinding>(`/v1/ai/model-assets/${assetId}/bindings/${bindingId}`, payload);
};

export const publishModelBinding = (assetId: string, bindingId: string): Promise<ModelAssetBinding> => {
  return request.post<any, ModelAssetBinding>(`/v1/ai/model-assets/${assetId}/bindings/${bindingId}/publish`);
};

export const unpublishModelBinding = (assetId: string, bindingId: string): Promise<ModelAssetBinding> => {
  return request.post<any, ModelAssetBinding>(`/v1/ai/model-assets/${assetId}/bindings/${bindingId}/unpublish`);
};

export const getModelBindingPriceVersions = (
  assetId: string,
  bindingId: string,
): Promise<ModelBindingPriceVersion[]> => {
  return request.get<any, ModelBindingPriceVersion[]>(
    `/v1/ai/model-assets/${assetId}/bindings/${bindingId}/price-versions`,
  );
};

export const restoreModelBindingPriceVersion = (
  assetId: string,
  bindingId: string,
  versionId: number,
): Promise<ModelAssetBinding> => {
  return request.post<any, ModelAssetBinding>(
    `/v1/ai/model-assets/${assetId}/bindings/${bindingId}/price-versions/${versionId}/restore`,
  );
};
