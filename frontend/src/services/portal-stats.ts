import type {
  PortalDepartmentBillRecord,
  PortalDepartmentBillsQuery,
  PortalUsageEventRecord,
  PortalUsageEventFilterOptions,
  PortalUsageEventsQuery,
  PortalUsageStatRecord,
  PortalUsageTrendPoint,
  PortalUsageTrendQuery,
} from '@/interfaces/portal-stats';
import request, { type RequestOptions } from './request';

const QUIET_PORTAL_STATS_OPTIONS: RequestOptions = {
  skipErrorModal: true,
};

export const getPortalUsageStats = (params?: { from?: number; to?: number }) => {
  return request.get<any, PortalUsageStatRecord[]>('/v1/portal/stats/usage', {
    ...QUIET_PORTAL_STATS_OPTIONS,
    params,
  });
};

export const getPortalUsageTrend = (params?: PortalUsageTrendQuery) => {
  return request.get<any, PortalUsageTrendPoint[]>('/v1/portal/stats/usage-trend', {
    ...QUIET_PORTAL_STATS_OPTIONS,
    params,
  });
};

export const getPortalUsageEvents = (params?: PortalUsageEventsQuery) => {
  return request.get<any, PortalUsageEventRecord[]>('/v1/portal/stats/usage-events', {
    ...QUIET_PORTAL_STATS_OPTIONS,
    params: serializePortalStatsParams(params),
  });
};

export const getPortalUsageEventOptions = (params?: Omit<PortalUsageEventsQuery, 'pageNum' | 'pageSize'>) => {
  return request.get<any, PortalUsageEventFilterOptions>('/v1/portal/stats/usage-event-options', {
    ...QUIET_PORTAL_STATS_OPTIONS,
    params: serializePortalStatsParams(params),
  });
};

export const getPortalDepartmentBills = (params?: PortalDepartmentBillsQuery) => {
  return request.get<any, PortalDepartmentBillRecord[]>('/v1/portal/stats/department-bills', {
    ...QUIET_PORTAL_STATS_OPTIONS,
    params: serializePortalStatsParams(params),
  });
};

function serializePortalStatsParams(params?: object | null) {
  if (!params) {
    return undefined;
  }
  return Object.fromEntries(
    Object.entries(params as Record<string, unknown>).flatMap(([key, value]) => {
      if (value === undefined || value === null) {
        return [];
      }
      if (Array.isArray(value)) {
        const normalized = value
          .map((item) => String(item).trim())
          .filter(Boolean)
          .join(',');
        return normalized ? [[key, normalized]] : [];
      }
      return [[key, value]];
    }),
  );
}
