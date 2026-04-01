import {
  AiSensitiveAuditQuery,
  AiSensitiveBlockAudit,
  AiSensitiveDetectRule,
  AiSensitiveMenuState,
  AiSensitiveReplaceRule,
  AiSensitiveStatus,
  AiSensitiveSystemConfig,
} from '@/interfaces/ai-sensitive';
import request, { RequestOptions } from './request';

const BASE_URL = '/v1/ai/sensitive-words';
const QUIET_MENU_REQUEST_OPTIONS: RequestOptions = {
  skipAuthRedirect: true,
  skipErrorModal: true,
};

export const getAiSensitiveMenuState = (): Promise<AiSensitiveMenuState> => {
  return request.get<any, AiSensitiveMenuState>(`${BASE_URL}/menu-state`, QUIET_MENU_REQUEST_OPTIONS);
};

export const getAiSensitiveStatus = (): Promise<AiSensitiveStatus> => {
  return request.get<any, AiSensitiveStatus>(`${BASE_URL}/status`);
};

export const reconcileAiSensitiveRules = (): Promise<AiSensitiveStatus> => {
  return request.post<any, AiSensitiveStatus>(`${BASE_URL}/reconcile`);
};

export const getAiSensitiveDetectRules = (): Promise<AiSensitiveDetectRule[]> => {
  return request.get<any, AiSensitiveDetectRule[]>(`${BASE_URL}/detect-rules`);
};

export const saveAiSensitiveDetectRule = (
  payload: AiSensitiveDetectRule,
): Promise<AiSensitiveDetectRule> => {
  if (payload.id) {
    return request.put<any, AiSensitiveDetectRule>(`${BASE_URL}/detect-rules/${payload.id}`, payload);
  }
  return request.post<any, AiSensitiveDetectRule>(`${BASE_URL}/detect-rules`, payload);
};

export const deleteAiSensitiveDetectRule = (id: number): Promise<any> => {
  return request.delete<any, any>(`${BASE_URL}/detect-rules/${id}`);
};

export const getAiSensitiveReplaceRules = (): Promise<AiSensitiveReplaceRule[]> => {
  return request.get<any, AiSensitiveReplaceRule[]>(`${BASE_URL}/replace-rules`);
};

export const saveAiSensitiveReplaceRule = (
  payload: AiSensitiveReplaceRule,
): Promise<AiSensitiveReplaceRule> => {
  if (payload.id) {
    return request.put<any, AiSensitiveReplaceRule>(`${BASE_URL}/replace-rules/${payload.id}`, payload);
  }
  return request.post<any, AiSensitiveReplaceRule>(`${BASE_URL}/replace-rules`, payload);
};

export const deleteAiSensitiveReplaceRule = (id: number): Promise<any> => {
  return request.delete<any, any>(`${BASE_URL}/replace-rules/${id}`);
};

export const getAiSensitiveAudits = (
  params: AiSensitiveAuditQuery,
): Promise<AiSensitiveBlockAudit[]> => {
  return request.get<any, AiSensitiveBlockAudit[]>(`${BASE_URL}/audits`, {
    params,
  });
};

export const getAiSensitiveSystemConfig = (): Promise<AiSensitiveSystemConfig> => {
  return request.get<any, AiSensitiveSystemConfig>(`${BASE_URL}/system-config`);
};

export const updateAiSensitiveSystemConfig = (
  payload: AiSensitiveSystemConfig,
): Promise<AiSensitiveSystemConfig> => {
  return request.put<any, AiSensitiveSystemConfig>(`${BASE_URL}/system-config`, payload);
};
