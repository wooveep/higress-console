export interface Consumer {
  name: string;
  department?: string;
  credentials?: Credential[] | string[];
  portalStatus?: 'active' | 'disabled' | 'pending' | string;
  portalDisplayName?: string;
  portalEmail?: string;
  portalUserSource?: string;
  portalUserLevel?: 'normal' | 'plus' | 'pro' | 'ultra' | string;
  portalTempPassword?: string;
  portalPassword?: string;
  version?: number;
  key?: string;
}

export interface ConsumerDetail extends Consumer {
  departmentId?: string;
  departmentPath?: string;
  createdAt?: string;
  updatedAt?: string;
  lastLoginAt?: string;
}

export interface ResetPasswordResponse {
  consumerName: string;
  tempPassword: string;
  updatedAt: string;
}

export interface InviteCodeRecord {
  inviteCode: string;
  status: 'active' | 'disabled' | 'used' | string;
  expiresAt?: string;
  usedByConsumer?: string;
  usedAt?: string;
  createdAt?: string;
}

export interface Credential {
  type: string;
  [propName: string]: any;
}

export const CredentialType = {
  KEY_AUTH: { key: 'key-auth', displayName: 'Key Auth', enabled: true, displayColor: '#4095e5' },
  OAUTH2: { key: 'oauth2', displayName: 'OAuth2', enabled: false, displayColor: '#4095e5' },
  JWT_AUTH: { key: 'jwt-auth', displayName: 'JWT', enabled: false, displayColor: '#4095e5' },
};

export interface KeyAuthCredential extends Credential {
  source: string;
  key?: string;
  value: string;
}

export enum KeyAuthCredentialSource {
  BEARER = 'BEARER',
  HEADER = 'HEADER',
  QUERY = 'QUERY',
}
