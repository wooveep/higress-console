import request, { RequestOptions } from './request';
import { Consumer, ConsumerDetail, InviteCodeRecord, ResetPasswordResponse } from '@/interfaces/consumer';

const QUIET_PORTAL_REQUEST_OPTIONS: RequestOptions = {
  skipErrorModal: true,
};

export const getConsumers = (): Promise<Consumer[]> => {
  return request.get<any, Consumer[]>('/v1/consumers');
};

export const getConsumerDetail = (name: string): Promise<ConsumerDetail> => {
  return request.get<any, ConsumerDetail>(`/v1/consumers/${name}`, QUIET_PORTAL_REQUEST_OPTIONS);
};

export const getConsumerDepartments = (): Promise<string[]> => {
  return request.get<any, string[]>('/v1/consumers/departments');
};

export const addConsumerDepartment = (name: string): Promise<any> => {
  return request.post<any, any>('/v1/consumers/departments', { name });
};

export const addConsumer = (payload: Consumer): Promise<any> => {
  return request.post<any, any>('/v1/consumers', payload);
};

export const deleteConsumer = (name: string): Promise<any> => {
  return request.delete<any, any>(`/v1/consumers/${name}`);
};

export const updateConsumer = (payload: Consumer): Promise<any> => {
  return request.put<any, any>(`/v1/consumers/${payload.name}`, payload);
};

export const updateConsumerStatus = (name: string, status: 'active' | 'disabled' | 'pending'): Promise<any> => {
  return request.patch<any, any>(`/v1/consumers/${name}/status`, { status });
};

export const resetConsumerPassword = (name: string): Promise<ResetPasswordResponse> => {
  return request.post<any, ResetPasswordResponse>(`/v1/consumers/${name}/password/reset`);
};

export const createInviteCode = (expiresInDays?: number): Promise<InviteCodeRecord> => {
  return request.post<any, InviteCodeRecord>('/v1/portal/invite-codes', { expiresInDays });
};

export const listInviteCodes = (params?: {
  pageNum?: number;
  pageSize?: number;
  status?: string;
}): Promise<InviteCodeRecord[]> => {
  return request.get<any, InviteCodeRecord[]>('/v1/portal/invite-codes', {
    ...QUIET_PORTAL_REQUEST_OPTIONS,
    params,
  });
};

export const updateInviteCodeStatus = (inviteCode: string, status: 'active' | 'disabled'): Promise<InviteCodeRecord> => {
  return request.patch<any, InviteCodeRecord>(`/v1/portal/invite-codes/${inviteCode}`, { status });
};

export const disableInviteCode = (inviteCode: string): Promise<InviteCodeRecord> => {
  return updateInviteCodeStatus(inviteCode, 'disabled');
};

export const enableInviteCode = (inviteCode: string): Promise<InviteCodeRecord> => {
  return updateInviteCodeStatus(inviteCode, 'active');
};
