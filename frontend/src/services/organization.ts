import {
  AssetGrantRecord,
  OrgAccountMutation,
  OrgAccountRecord,
  OrgDepartmentMoveRequest,
  OrgImportResult,
  OrgDepartmentMutation,
  OrgDepartmentNode,
} from '@/interfaces/org';
import request, { RequestOptions } from './request';

const QUIET_PORTAL_REQUEST_OPTIONS: RequestOptions = {
  skipErrorModal: true,
};

export const listOrgDepartmentsTree = (): Promise<OrgDepartmentNode[]> => {
  return request.get<any, OrgDepartmentNode[]>('/v1/org/departments/tree', QUIET_PORTAL_REQUEST_OPTIONS);
};

export const createOrgDepartment = (payload: OrgDepartmentMutation): Promise<OrgDepartmentNode> => {
  return request.post<any, OrgDepartmentNode>('/v1/org/departments', payload);
};

export const updateOrgDepartment = (departmentId: string, payload: Pick<OrgDepartmentMutation, 'name'>): Promise<OrgDepartmentNode> => {
  return request.put<any, OrgDepartmentNode>(`/v1/org/departments/${departmentId}`, payload);
};

export const moveOrgDepartment = (departmentId: string, payload: OrgDepartmentMoveRequest): Promise<OrgDepartmentNode> => {
  return request.patch<any, OrgDepartmentNode>(`/v1/org/departments/${departmentId}/move`, payload);
};

export const deleteOrgDepartment = (departmentId: string): Promise<void> => {
  return request.delete<any, any>(`/v1/org/departments/${departmentId}`);
};

export const listOrgAccounts = (): Promise<OrgAccountRecord[]> => {
  return request.get<any, OrgAccountRecord[]>('/v1/org/accounts', QUIET_PORTAL_REQUEST_OPTIONS);
};

export const createOrgAccount = (payload: OrgAccountMutation): Promise<OrgAccountRecord> => {
  return request.post<any, OrgAccountRecord>('/v1/org/accounts', payload);
};

export const updateOrgAccount = (consumerName: string, payload: OrgAccountMutation): Promise<OrgAccountRecord> => {
  return request.put<any, OrgAccountRecord>(`/v1/org/accounts/${consumerName}`, payload);
};

export const updateOrgAccountAssignment = (
  consumerName: string,
  payload: Pick<OrgAccountMutation, 'departmentId' | 'parentConsumerName'>,
): Promise<OrgAccountRecord> => {
  return request.patch<any, OrgAccountRecord>(`/v1/org/accounts/${consumerName}/assignment`, payload);
};

export const updateOrgAccountStatus = (
  consumerName: string,
  status: 'active' | 'disabled' | 'pending',
): Promise<OrgAccountRecord> => {
  return request.patch<any, OrgAccountRecord>(`/v1/org/accounts/${consumerName}/status`, { status });
};

export const downloadOrgTemplate = (): Promise<ArrayBuffer> => {
  return request.get<any, ArrayBuffer>('/v1/org/template', {
    responseType: 'arraybuffer',
    skipErrorModal: false,
  });
};

export const exportOrgWorkbook = (): Promise<ArrayBuffer> => {
  return request.get<any, ArrayBuffer>('/v1/org/export', {
    responseType: 'arraybuffer',
    skipErrorModal: false,
  });
};

export const importOrgWorkbook = (file: File): Promise<OrgImportResult> => {
  const formData = new FormData();
  formData.append('file', file);
  return request.post<any, OrgImportResult>('/v1/org/import', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
};

export const listAssetGrants = (assetType: string, assetId: string): Promise<AssetGrantRecord[]> => {
  return request.get<any, AssetGrantRecord[]>(`/v1/assets/${assetType}/${assetId}/grants`);
};

export const replaceAssetGrants = (
  assetType: string,
  assetId: string,
  grants: AssetGrantRecord[],
): Promise<AssetGrantRecord[]> => {
  return request.put<any, AssetGrantRecord[]>(`/v1/assets/${assetType}/${assetId}/grants`, { grants });
};
