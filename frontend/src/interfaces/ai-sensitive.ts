export type AiSensitiveMatchType = 'contains' | 'exact' | 'regex';
export type AiSensitiveReplaceType = 'replace' | 'hash';
export type AiSensitiveDateTimeValue = string | number[];

export interface AiSensitiveMenuState {
  enabled: boolean;
  enabledRouteCount: number;
}

export interface AiSensitiveDetectRule {
  id?: number;
  pattern: string;
  matchType: AiSensitiveMatchType;
  description?: string;
  priority?: number;
  enabled?: boolean;
  createdAt?: AiSensitiveDateTimeValue;
  updatedAt?: AiSensitiveDateTimeValue;
}

export interface AiSensitiveReplaceRule {
  id?: number;
  pattern: string;
  replaceType: AiSensitiveReplaceType;
  replaceValue?: string;
  restore?: boolean;
  description?: string;
  priority?: number;
  enabled?: boolean;
  createdAt?: AiSensitiveDateTimeValue;
  updatedAt?: AiSensitiveDateTimeValue;
}

export interface AiSensitiveReplacePreset {
  key: string;
  label: string;
  description?: string;
  pattern: string;
  replaceType: AiSensitiveReplaceType;
  replaceValue?: string;
  restore?: boolean;
  priority?: number;
}

export interface AiSensitiveBlockAudit {
  id: number;
  requestId?: string;
  routeName?: string;
  consumerName?: string;
  displayName?: string;
  blockedAt?: AiSensitiveDateTimeValue;
  blockedBy?: string;
  requestPhase?: string;
  blockedReasonJson?: string;
  guardCode?: number;
  blockedDetails?: Array<{
    type?: string;
    level?: string;
    suggestion?: string;
  }>;
  matchType?: AiSensitiveMatchType;
  matchedRule?: string;
  matchedExcerpt?: string;
  providerId?: number;
  costUsd?: string;
}

export interface AiSensitiveStatus {
  dbEnabled: boolean;
  detectRuleCount: number;
  replaceRuleCount: number;
  auditRecordCount: number;
  systemDenyEnabled?: boolean;
  systemDictionaryWordCount?: number;
  systemDictionaryUpdatedAt?: AiSensitiveDateTimeValue;
  projectedInstanceCount: number;
  lastReconciledAt?: AiSensitiveDateTimeValue;
  lastMigratedAt?: AiSensitiveDateTimeValue;
  lastError?: string;
}

export interface AiSensitiveSystemConfig {
  systemDenyEnabled: boolean;
  dictionaryText: string;
  updatedAt?: AiSensitiveDateTimeValue;
  updatedBy?: string;
}

export interface AiSensitiveAuditQuery {
  consumerName?: string;
  displayName?: string;
  routeName?: string;
  matchType?: AiSensitiveMatchType;
  startTime?: string;
  endTime?: string;
}
