export interface PortalUsageStatRecord {
  consumerName: string;
  modelName: string;
  requestCount: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  cacheCreationInputTokens: number;
  cacheCreation5mInputTokens: number;
  cacheCreation1hInputTokens: number;
  cacheReadInputTokens: number;
  inputImageTokens: number;
  outputImageTokens: number;
  inputImageCount: number;
  outputImageCount: number;
}

export interface PortalUsageEventRecord {
  eventId?: string;
  requestId?: string;
  traceId?: string;
  consumerName?: string;
  departmentId?: string;
  departmentPath?: string;
  apiKeyId?: string;
  modelId?: string;
  priceVersionId?: number;
  routeName?: string;
  requestPath?: string;
  requestKind?: string;
  requestStatus?: string;
  usageStatus?: string;
  httpStatus?: number;
  errorCode?: string;
  errorMessage?: string;
  inputTokens?: number;
  outputTokens?: number;
  totalTokens?: number;
  requestCount?: number;
  costMicroYuan?: number;
  startedAt?: string;
  finishedAt?: string;
  serviceDurationMs?: number;
  occurredAt?: string;
}

export interface PortalUsageTrendPoint {
  bucketLabel: string;
  requestCount: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  costMicroYuan: number;
  activeConsumers: number;
}

export interface PortalStatsSelectOption {
  label: string;
  value: string;
}

export interface PortalUsageEventFilterOptions {
  consumers: PortalStatsSelectOption[];
  departments: PortalStatsSelectOption[];
  apiKeys: PortalStatsSelectOption[];
  models: PortalStatsSelectOption[];
  routes: PortalStatsSelectOption[];
  requestStatuses: PortalStatsSelectOption[];
  usageStatuses: PortalStatsSelectOption[];
}

export interface PortalDepartmentBillRecord {
  departmentId?: string;
  departmentName?: string;
  departmentPath?: string;
  requestCount?: number;
  totalTokens?: number;
  totalCost?: number;
  activeConsumers?: number;
}

export interface PortalUsageEventsQuery {
  from?: number;
  to?: number;
  consumerNames?: string[];
  departmentIds?: string[];
  includeChildren?: boolean;
  apiKeyIds?: string[];
  modelIds?: string[];
  routeNames?: string[];
  requestStatuses?: string[];
  usageStatuses?: string[];
  pageNum?: number;
  pageSize?: number;
}

export interface PortalDepartmentBillsQuery {
  from?: number;
  to?: number;
  departmentIds?: string[];
  includeChildren?: boolean;
}

export interface PortalUsageTrendQuery {
  from?: number;
  to?: number;
  bucket?: '5m' | 'hour' | 'day';
  consumerName?: string;
  departmentId?: string;
  includeChildren?: boolean;
  modelId?: string;
  routeName?: string;
}
