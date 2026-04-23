import i18n from '@/i18n';
import {
  addGatewayDomain,
  deleteGatewayDomain,
  getGatewayDomains,
  updateGatewayDomain,
} from '@/services/domain';
import {
  addServiceSource as addServiceSourceRecord,
  deleteServiceSource,
  getServiceSources,
  updateServiceSource,
} from '@/services/service-source';
import { getGatewayServices } from '@/services/service';
import {
  addTlsCertificate as addTlsCertificateRecord,
  deleteTlsCertificate,
  getTlsCertificates,
  updateTlsCertificate,
} from '@/services/tls-certificate';
import {
  addLlmProvider,
  deleteLlmProvider,
  getLlmProviders,
  updateLlmProvider,
} from '@/services/llm-provider';
import {
  createAgentCatalog,
  getAgentCatalogs,
  publishAgentCatalog,
  unpublishAgentCatalog,
  updateAgentCatalog,
} from '@/services/agent-catalog';
import {
  createModelAsset,
  getModelAssets,
  updateModelAsset,
} from '@/services/model-asset';
import {
  createOrUpdateMcpServer,
  deleteMcpServer,
  listMcpServers,
} from '@/services/mcp';
import {
  createWasmPlugin,
  deleteWasmPlugin,
  getWasmPlugins,
  updateWasmPlugin,
} from '@/services/plugin';
import {
  addAiRoute,
  deleteAiRoute,
  getAiRoutes,
  updateAiRoute,
} from '@/services/ai-route';
import {
  addGatewayRouteCompat,
  deleteGatewayRouteCompat,
  getGatewayRouteDetailCompat,
  getGatewayRoutesCompat,
  updateGatewayRouteCompat,
} from '@/services/route-compat';

export interface ResourceAction {
  key: string;
  labelKey: string;
  visible?: (record: any) => boolean;
  run: (record: any) => Promise<any>;
}

export interface ResourceDefinition {
  keyField: string;
  list: () => Promise<any>;
  create?: (payload: any) => Promise<any>;
  update?: (record: any, payload: any) => Promise<any>;
  remove?: (record: any) => Promise<any>;
  detail?: (record: any) => Promise<any>;
  fields?: string[];
  searchFields?: string[];
  normalizeList?: (payload: any) => any[];
  createTemplate?: () => any;
  readonly?: boolean;
  actions?: ResourceAction[];
}

function normalizeRows(payload: any) {
  if (Array.isArray(payload)) {
    return payload;
  }
  if (payload && Array.isArray(payload.data)) {
    return payload.data;
  }
  return [];
}

export const resourceRegistry: Record<string, ResourceDefinition> = {
  'service-source': {
    keyField: 'name',
    list: () => getServiceSources({} as any),
    create: addServiceSourceRecord,
    update: (_, payload) => updateServiceSource(payload),
    remove: (record) => deleteServiceSource(record.name),
    fields: ['name', 'type', 'domain', 'port', 'version'],
    searchFields: ['name', 'type', 'domain'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ name: '', type: 'static', domain: '', port: 80 }),
  },
  service: {
    keyField: 'name',
    list: getGatewayServices,
    readonly: true,
    fields: ['name', 'namespace', 'port', 'endpoints'],
    searchFields: ['name', 'namespace'],
    normalizeList: normalizeRows,
  },
  route: {
    keyField: 'name',
    list: getGatewayRoutesCompat,
    detail: (record) => getGatewayRouteDetailCompat(record.name),
    create: addGatewayRouteCompat,
    update: (_, payload) => updateGatewayRouteCompat(payload),
    remove: (record) => deleteGatewayRouteCompat(record.name),
    fields: ['name', 'domains', 'methods', 'services', 'version'],
    searchFields: ['name'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ name: '', domains: [], methods: ['GET'], services: [] }),
  },
  domain: {
    keyField: 'name',
    list: getGatewayDomains,
    create: addGatewayDomain,
    update: (_, payload) => updateGatewayDomain(payload),
    remove: (record) => deleteGatewayDomain(record.name),
    fields: ['name', 'enableHttps', 'certIdentifier', 'version'],
    searchFields: ['name', 'certIdentifier'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ name: '', enableHttps: 'off' }),
  },
  'tls-certificate': {
    keyField: 'name',
    list: getTlsCertificates,
    create: addTlsCertificateRecord,
    update: (_, payload) => updateTlsCertificate(payload),
    remove: (record) => deleteTlsCertificate(record.name),
    fields: ['name', 'domains', 'validityStart', 'validityEnd'],
    searchFields: ['name'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ name: '', cert: '', key: '', domains: [] }),
  },
  plugin: {
    keyField: 'name',
    list: () => getWasmPlugins(i18n.global.locale.value),
    create: createWasmPlugin,
    update: (record, payload) => updateWasmPlugin(record.name, payload),
    remove: (record) => deleteWasmPlugin(record.name),
    fields: ['name', 'title', 'category', 'phase', 'imageRepository', 'imageVersion'],
    searchFields: ['name', 'title', 'category'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ name: '', title: '', category: 'custom', imageRepository: '', imageVersion: 'latest' }),
  },
  'ai-provider': {
    keyField: 'name',
    list: getLlmProviders,
    create: addLlmProvider,
    update: (_, payload) => updateLlmProvider(payload),
    remove: (record) => deleteLlmProvider(record.name),
    fields: ['name', 'type', 'protocol', 'proxyName', 'tokens'],
    searchFields: ['name', 'type', 'protocol'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ name: '', type: '', protocol: 'openai/v1', tokens: [] }),
  },
  'ai-model-assets': {
    keyField: 'assetId',
    list: getModelAssets,
    create: createModelAsset,
    update: (record, payload) => updateModelAsset(record.assetId, payload),
    fields: ['assetId', 'canonicalName', 'displayName', 'tags', 'updatedAt'],
    searchFields: ['assetId', 'canonicalName', 'displayName'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ assetId: '', canonicalName: '', displayName: '', tags: [], bindings: [] }),
  },
  'ai-agent-catalog': {
    keyField: 'agentId',
    list: getAgentCatalogs,
    create: createAgentCatalog,
    update: (record, payload) => updateAgentCatalog(record.agentId, payload),
    fields: ['agentId', 'displayName', 'status', 'mcpServerName', 'publishedAt', 'updatedAt'],
    searchFields: ['agentId', 'displayName', 'canonicalName', 'mcpServerName'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ agentId: '', canonicalName: '', displayName: '', mcpServerName: '' }),
    actions: [
      {
        key: 'publish',
        labelKey: 'misc.save',
        visible: (record) => record.status !== 'published',
        run: (record) => publishAgentCatalog(record.agentId),
      },
      {
        key: 'unpublish',
        labelKey: 'misc.cancel',
        visible: (record) => record.status === 'published',
        run: (record) => unpublishAgentCatalog(record.agentId),
      },
    ],
  },
  'ai-route': {
    keyField: 'name',
    list: getAiRoutes,
    create: addAiRoute,
    update: (_, payload) => updateAiRoute(payload),
    remove: (record) => deleteAiRoute(record.name),
    fields: ['name', 'domains', 'services', 'modelPredicates', 'updatedAt'],
    searchFields: ['name'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ name: '', domains: [], services: [] }),
  },
  'mcp-list': {
    keyField: 'name',
    list: () => listMcpServers({ pageNum: 1, pageSize: 200 }),
    create: createOrUpdateMcpServer,
    update: (_, payload) => createOrUpdateMcpServer(payload),
    remove: (record) => deleteMcpServer(record.name),
    fields: ['name', 'type', 'description', 'domains'],
    searchFields: ['name', 'type', 'description'],
    normalizeList: normalizeRows,
    createTemplate: () => ({ name: '', type: 'OPEN_API', description: '', domains: [] }),
  },
};
