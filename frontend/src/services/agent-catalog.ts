import { AgentCatalogOptions, AgentCatalogRecord } from '@/interfaces/agent-catalog';
import request, { RequestOptions } from './request';

const QUIET_PORTAL_REQUEST_OPTIONS: RequestOptions = {
  skipErrorModal: true,
};

export const getAgentCatalogs = (): Promise<AgentCatalogRecord[]> => {
  return request.get<any, AgentCatalogRecord[]>('/v1/ai/agent-catalog', QUIET_PORTAL_REQUEST_OPTIONS);
};

export const getAgentCatalog = (agentId: string): Promise<AgentCatalogRecord> => {
  return request.get<any, AgentCatalogRecord>(`/v1/ai/agent-catalog/${agentId}`);
};

export const getAgentCatalogOptions = (): Promise<AgentCatalogOptions> => {
  return request.get<any, AgentCatalogOptions>('/v1/ai/agent-catalog/options', {
    skipErrorModal: true,
  });
};

export const createAgentCatalog = (payload: AgentCatalogRecord): Promise<AgentCatalogRecord> => {
  return request.post<any, AgentCatalogRecord>('/v1/ai/agent-catalog', payload);
};

export const updateAgentCatalog = (agentId: string, payload: AgentCatalogRecord): Promise<AgentCatalogRecord> => {
  return request.put<any, AgentCatalogRecord>(`/v1/ai/agent-catalog/${agentId}`, payload);
};

export const publishAgentCatalog = (agentId: string): Promise<AgentCatalogRecord> => {
  return request.post<any, AgentCatalogRecord>(`/v1/ai/agent-catalog/${agentId}/publish`);
};

export const unpublishAgentCatalog = (agentId: string): Promise<AgentCatalogRecord> => {
  return request.post<any, AgentCatalogRecord>(`/v1/ai/agent-catalog/${agentId}/unpublish`);
};
