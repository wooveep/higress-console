export interface OrgDepartmentNode {
  departmentId: string;
  name: string;
  parentDepartmentId?: string;
  adminConsumerName?: string;
  adminDisplayName?: string;
  level?: number;
  memberCount?: number;
  children?: OrgDepartmentNode[];
}

export interface OrgAccountRecord {
  consumerName: string;
  displayName?: string;
  email?: string;
  status?: 'active' | 'disabled' | 'pending' | string;
  userLevel?: 'normal' | 'plus' | 'pro' | 'ultra' | string;
  source?: string;
  departmentId?: string;
  departmentName?: string;
  departmentPath?: string;
  parentConsumerName?: string;
  isDepartmentAdmin?: boolean;
  lastLoginAt?: string;
  tempPassword?: string;
}

export interface OrgAccountMutation {
  consumerName: string;
  displayName?: string;
  email?: string;
  userLevel?: string;
  password?: string;
  status?: string;
  departmentId?: string;
  parentConsumerName?: string;
}

export interface OrgDepartmentMutation {
  name?: string;
  parentDepartmentId?: string;
  adminConsumerName?: string;
  admin?: OrgDepartmentAdminMutation;
}

export interface OrgDepartmentMoveRequest {
  parentDepartmentId?: string;
}

export interface AssetGrantRecord {
  assetType?: string;
  assetId?: string;
  subjectType?: 'consumer' | 'department' | 'user_level' | string;
  subjectId?: string;
}

export interface OrgDepartmentAdminMutation {
  consumerName: string;
  displayName?: string;
  email?: string;
  userLevel?: string;
  password?: string;
}

export interface OrgImportResult {
  createdDepartments: number;
  updatedDepartments: number;
  createdAccounts: number;
  updatedAccounts: number;
}
