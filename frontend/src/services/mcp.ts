import request from '@/services/request';
import type { PageResult } from '@/interfaces/common';
import {
  McpServer,
  McpServerPageQuery,
  McpServerConsumerDetail,
  McpPresetTemplate,
} from '@/interfaces/mcp';

const BASE_URL = '/v1/mcpServer';
const MCP_TEMPLATE_BASE_URL = '/mcp-templates';

export const listMcpServers = (query: McpServerPageQuery): Promise<PageResult<McpServer>> => {
  return request.get<any, PageResult<McpServer>>(BASE_URL, { params: query });
};

export const getMcpServer = (name: string): Promise<McpServer> => {
  return request.get<any, McpServer>(`${BASE_URL}/${name}`);
};

export const createOrUpdateMcpServer = (payload: McpServer): Promise<McpServer> => {
  return payload.name ?
    request.put<any, McpServer>(`${BASE_URL}/${payload.name}`, payload) :
    request.post<any, McpServer>(BASE_URL, payload);
};

export const deleteMcpServer = (name: string): Promise<any> => {
  return request.delete<any, any>(`${BASE_URL}/${name}`);
};

export const listMcpConsumers = (
  query: any,
): Promise<PageResult<McpServerConsumerDetail>> => {
  return request.get<any, PageResult<McpServerConsumerDetail>>(`${BASE_URL}/consumers`, {
    params: query,
  });
};

export const swaggerToMcpConfig = (payload: { content: string }): Promise<any> => {
  return request.post<any, any>(`${BASE_URL}/swaggerToMcpConfig`, payload);
};

export const listMcpPresetTemplates = async (): Promise<McpPresetTemplate[]> => {
  const response = await fetch(`${MCP_TEMPLATE_BASE_URL}/index.json?ts=${Date.now()}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch MCP preset templates: ${response.status}`);
  }
  const templateIds: string[] = await response.json();
  return templateIds.map((id) => ({
    id,
    name: id,
  }));
};

export const getMcpPresetTemplate = async (templateId: string): Promise<string> => {
  const response = await fetch(`${MCP_TEMPLATE_BASE_URL}/${templateId}.yaml?ts=${Date.now()}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch MCP preset template ${templateId}: ${response.status}`);
  }
  return response.text();
};
