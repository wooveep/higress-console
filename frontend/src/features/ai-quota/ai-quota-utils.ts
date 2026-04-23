import type { AiQuotaScheduleRule, AiQuotaUserPolicy } from '@/interfaces/ai-quota';
import {
  dateTimeLocalInputToISOString,
  formatDateTimeDisplay,
  formatDateTimeLocalInputValue,
  getNowDateTimeLocalInputValue,
} from '@/utils/time';

export const BUILTIN_ADMIN_CONSUMER = 'administrator';
export const MICRO_YUAN_PER_RMB = 1_000_000;

export interface AiQuotaPolicyFormValues {
  limitTotal: number;
  limit5h: number;
  limitDaily: number;
  dailyResetTime: string;
  limitWeekly: number;
  limitMonthly: number;
  costResetAt: string;
}

export interface AiQuotaScheduleFormValues {
  action: 'REFRESH' | 'DELTA';
  cron: string;
  value: number;
  enabled: boolean;
}

export function toDisplayAmount(value: number) {
  return `${(value / MICRO_YUAN_PER_RMB).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 6,
  })} RMB`;
}

export function toFormAmount(value: number) {
  return (value || 0) / MICRO_YUAN_PER_RMB;
}

export function toStoredAmount(value: number) {
  return Math.round((value || 0) * MICRO_YUAN_PER_RMB);
}

export function createDefaultPolicyValues(policy?: Partial<AiQuotaUserPolicy>): AiQuotaPolicyFormValues {
  return {
    limitTotal: toFormAmount(policy?.limitTotal ?? 0),
    limit5h: toFormAmount(policy?.limit5h ?? 0),
    limitDaily: toFormAmount(policy?.limitDaily ?? 0),
    dailyResetTime: policy?.dailyResetTime || '00:00',
    limitWeekly: toFormAmount(policy?.limitWeekly ?? 0),
    limitMonthly: toFormAmount(policy?.limitMonthly ?? 0),
    costResetAt: formatDateTimeLocalInputValue(policy?.costResetAt || '', ''),
  };
}

export function createDefaultScheduleValues(currentAmount: number): AiQuotaScheduleFormValues {
  return {
    action: 'REFRESH',
    cron: '0 0 0 * * *',
    value: toFormAmount(currentAmount),
    enabled: true,
  };
}

export function createScheduleValuesFromRule(rule: AiQuotaScheduleRule): AiQuotaScheduleFormValues {
  return {
    action: rule.action,
    cron: rule.cron,
    value: toFormAmount(rule.value),
    enabled: rule.enabled,
  };
}

export function formatScheduleTime(value?: number) {
  return formatDateTimeDisplay(value);
}

export function buildPolicyPayload(values: AiQuotaPolicyFormValues) {
  return {
    limitTotal: toStoredAmount(values.limitTotal),
    limit5h: toStoredAmount(values.limit5h),
    limitDaily: toStoredAmount(values.limitDaily),
    dailyResetMode: 'fixed',
    dailyResetTime: values.dailyResetTime,
    limitWeekly: toStoredAmount(values.limitWeekly),
    limitMonthly: toStoredAmount(values.limitMonthly),
    costResetAt: dateTimeLocalInputToISOString(values.costResetAt?.trim()),
  };
}

export function buildSchedulePayload(values: AiQuotaScheduleFormValues) {
  return {
    action: values.action,
    cron: values.cron,
    value: toStoredAmount(values.value),
    enabled: values.enabled,
  };
}

export function fillPolicyResetNow() {
  return getNowDateTimeLocalInputValue();
}
