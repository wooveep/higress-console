export interface AgentCatalogOptionServer {
  mcpServerName: string;
  description?: string;
  type?: string;
  domains?: string[];
  authEnabled?: boolean;
  authType?: string;
}

export interface AgentCatalogOptions {
  servers: AgentCatalogOptionServer[];
}

export interface AgentCatalogRecord {
  agentId: string;
  canonicalName?: string;
  displayName?: string;
  intro?: string;
  description?: string;
  iconUrl?: string;
  tags?: string[];
  mcpServerName?: string;
  status?: 'draft' | 'published' | 'unpublished' | string;
  toolCount?: number;
  transportTypes?: string[];
  resourceSummary?: string;
  promptSummary?: string;
  publishedAt?: string;
  unpublishedAt?: string;
  createdAt?: string;
  updatedAt?: string;
}
