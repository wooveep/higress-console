import { WasmPluginData } from '@/interfaces/wasm-plugin';
import request, { type RequestOptions } from './request';

const QUIET_PLUGIN_REQUEST_OPTIONS: RequestOptions = {
  skipErrorModal: true,
};

// 获取全局的插件配置列表
export const getWasmPlugins = (lang: string): Promise<any> => {
  return request.get('/v1/wasm-plugins', { params: { lang } });
};

// 获取全局的指定插件配置
export const getPluginsDetail = ({ pluginName }): Promise<any> => {
  return request.get(`/v1/global/plugin-instances/${pluginName}`);
};

export const createWasmPlugin = (payload: WasmPluginData) => {
  return request.post<any, any>('/v1/wasm-plugins', payload);
};

export const updateWasmPlugin = (name: string, payload: WasmPluginData) => {
  return request.put<any, any>(`/v1/wasm-plugins/${name}`, payload);
};

export const deleteWasmPlugin = (name: string) => {
  return request.delete<any, any>(`/v1/wasm-plugins/${name}`);
};

// 获取指定插件的运行时配置数据格式
export const getWasmPluginsConfig = (name: string) => {
  return request.get<any, any>(`/v1/wasm-plugins/${name}/config`);
};

export const getWasmPluginReadme = (name: string) => {
  return request.get<any, string>(`/v1/wasm-plugins/${name}/readme`, QUIET_PLUGIN_REQUEST_OPTIONS);
};

// 获取全局的指定插件配置
export const getGlobalPluginInstance = (pluginName: string) => {
  return request.get<any, any>(`/v1/global/plugin-instances/${pluginName}`);
};

// 修改全局的指定插件配置
export const updateGlobalPluginInstance = (pluginName: string, payload) => {
  return request.put<any, any>(`/v1/global/plugin-instances/${pluginName}`, payload);
};

export const deleteGlobalPluginInstance = (pluginName: string) => {
  return request.delete<any, any>(`/v1/global/plugin-instances/${pluginName}`);
};

// 获取指定路由的插件配置列表
export const getRoutePluginInstances = (name: string) => {
  return request.get<any, any>(`/v1/routes/${name}/plugin-instances`);
};

export const getRoutePluginInstancesWithAliases = (name: string, aliases: string[] = []) => {
  return request.get<any, any>(`/v1/routes/${name}/plugin-instances`, {
    params: {
      aliases: aliases.join(','),
    },
  });
};

// 获取指定路由的指定插件配置
export const getRoutePluginInstance = (params: { name: string; pluginName: string }) => {
  const { name, pluginName } = params;
  return request.get<any, any>(`/v1/routes/${name}/plugin-instances/${pluginName}`);
};

export const getRoutePluginInstanceWithAliases = (params: { name: string; pluginName: string; aliases?: string[] }) => {
  const { name, pluginName, aliases = [] } = params;
  return request.get<any, any>(`/v1/routes/${name}/plugin-instances/${pluginName}`, {
    params: {
      aliases: aliases.join(','),
    },
  });
};

// 修改指定路由的指定插件配置
export const updateRoutePluginInstance = (params: { name: string; pluginName: string }, payload) => {
  const { name, pluginName } = params;
  return request.put<any, any>(`/v1/routes/${name}/plugin-instances/${pluginName}`, payload);
};

export const deleteRoutePluginInstance = (params: { name: string; pluginName: string }) => {
  const { name, pluginName } = params;
  return request.delete<any, any>(`/v1/routes/${name}/plugin-instances/${pluginName}`);
};

// 获取指定域名的插件配置列表
export const getDomainPluginInstances = (name: string) => {
  return request.get<any, any>(`/v1/domains/${name}/plugin-instances`);
};

// 获取指定域名的指定插件配置
export const getDomainPluginInstance = (params: { name: string; pluginName: string }) => {
  const { name, pluginName } = params;
  return request.get<any, any>(`/v1/domains/${name}/plugin-instances/${pluginName}`);
};

// 修改指定域名的指定插件配置
export const updateDomainPluginInstance = (params: { name: string; pluginName: string }, payload) => {
  const { name, pluginName } = params;
  return request.put<any, any>(`/v1/domains/${name}/plugin-instances/${pluginName}`, payload);
};

export const deleteDomainPluginInstance = (params: { name: string; pluginName: string }) => {
  const { name, pluginName } = params;
  return request.delete<any, any>(`/v1/domains/${name}/plugin-instances/${pluginName}`);
};

export const getServicePluginInstances = (name: string) => {
  return request.get<any, any>(`/v1/services/${name}/plugin-instances`);
};

export const getServicePluginInstance = (params: { name: string; pluginName: string }) => {
  const { name, pluginName } = params;
  return request.get<any, any>(`/v1/services/${name}/plugin-instances/${pluginName}`);
};

export const updateServicePluginInstance = (params: { name: string; pluginName: string }, payload) => {
  const { name, pluginName } = params;
  return request.put<any, any>(`/v1/services/${name}/plugin-instances/${pluginName}`, payload);
};

export const deleteServicePluginInstance = (params: { name: string; pluginName: string }) => {
  const { name, pluginName } = params;
  return request.delete<any, any>(`/v1/services/${name}/plugin-instances/${pluginName}`);
};
