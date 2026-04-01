import {
  AiQuotaConsumerQuota,
  AiQuotaMenuState,
  AiQuotaRouteSummary,
  AiQuotaScheduleRule,
  AiQuotaScheduleRuleRequest,
  AiQuotaUserPolicy,
  AiQuotaUserPolicyRequest,
} from '@/interfaces/ai-quota';
import request, { RequestOptions } from './request';

const BASE_URL = '/v1/ai/quotas';
const QUIET_MENU_REQUEST_OPTIONS: RequestOptions = {
  skipAuthRedirect: true,
  skipErrorModal: true,
};

export const getAiQuotaMenuState = (): Promise<AiQuotaMenuState> => {
  return request.get<any, AiQuotaMenuState>(`${BASE_URL}/menu-state`, QUIET_MENU_REQUEST_OPTIONS);
};

export const getAiQuotaRoutes = (): Promise<AiQuotaRouteSummary[]> => {
  return request.get<any, AiQuotaRouteSummary[]>(`${BASE_URL}/routes`);
};

export const getAiQuotaConsumers = (routeName: string): Promise<AiQuotaConsumerQuota[]> => {
  return request.get<any, AiQuotaConsumerQuota[]>(`${BASE_URL}/routes/${routeName}/consumers`);
};

export const refreshAiQuota = (
  routeName: string,
  consumerName: string,
  value: number,
): Promise<AiQuotaConsumerQuota> => {
  return request.put<any, AiQuotaConsumerQuota>(
    `${BASE_URL}/routes/${routeName}/consumers/${consumerName}/quota`,
    { value },
  );
};

export const deltaAiQuota = (
  routeName: string,
  consumerName: string,
  value: number,
): Promise<AiQuotaConsumerQuota> => {
  return request.post<any, AiQuotaConsumerQuota>(
    `${BASE_URL}/routes/${routeName}/consumers/${consumerName}/delta`,
    { value },
  );
};

export const getAiQuotaUserPolicy = (
  routeName: string,
  consumerName: string,
): Promise<AiQuotaUserPolicy> => {
  return request.get<any, AiQuotaUserPolicy>(
    `${BASE_URL}/routes/${routeName}/consumers/${consumerName}/policy`,
  );
};

export const saveAiQuotaUserPolicy = (
  routeName: string,
  consumerName: string,
  payload: AiQuotaUserPolicyRequest,
): Promise<AiQuotaUserPolicy> => {
  return request.put<any, AiQuotaUserPolicy>(
    `${BASE_URL}/routes/${routeName}/consumers/${consumerName}/policy`,
    payload,
  );
};

export const getAiQuotaScheduleRules = (
  routeName: string,
  consumerName?: string,
): Promise<AiQuotaScheduleRule[]> => {
  return request.get<any, AiQuotaScheduleRule[]>(`${BASE_URL}/routes/${routeName}/schedules`, {
    params: {
      consumerName,
    },
  });
};

export const saveAiQuotaScheduleRule = (
  routeName: string,
  payload: AiQuotaScheduleRuleRequest,
): Promise<AiQuotaScheduleRule> => {
  return request.put<any, AiQuotaScheduleRule>(`${BASE_URL}/routes/${routeName}/schedules`, payload);
};

export const deleteAiQuotaScheduleRule = (routeName: string, ruleId: string): Promise<any> => {
  return request.delete<any, any>(`${BASE_URL}/routes/${routeName}/schedules/${ruleId}`);
};
