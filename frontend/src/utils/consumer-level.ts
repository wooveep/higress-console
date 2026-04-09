import { Consumer } from '@/interfaces/consumer';

export const USER_LEVELS = ['normal', 'plus', 'pro', 'ultra'] as const;

export type UserLevel = (typeof USER_LEVELS)[number];

const USER_LEVEL_RANK: Record<UserLevel, number> = {
  normal: 1,
  plus: 2,
  pro: 3,
  ultra: 4,
};

export const normalizeUserLevel = (level?: string | null): UserLevel => {
  const normalized = (level || '').trim().toLowerCase() as UserLevel;
  return USER_LEVEL_RANK[normalized] ? normalized : 'normal';
};

export const normalizeUserLevels = (levels?: string[] | null): UserLevel[] => {
  if (!Array.isArray(levels)) {
    return [];
  }
  const result: UserLevel[] = [];
  levels.forEach((level) => {
    const normalized = (level || '').trim().toLowerCase() as UserLevel;
    if (!USER_LEVEL_RANK[normalized]) {
      return;
    }
    if (!result.includes(normalized)) {
      result.push(normalized);
    }
  });
  return result;
};

export const expandConsumersByAllowedLevels = (levels: string[] | undefined, consumers: Consumer[]): string[] => {
  const normalizedLevels = normalizeUserLevels(levels);
  if (!normalizedLevels.length) {
    return [];
  }
  const minRank = normalizedLevels.reduce((acc, item) => Math.min(acc, USER_LEVEL_RANK[item]), Number.MAX_SAFE_INTEGER);
  const result: string[] = [];
  consumers.forEach((consumer) => {
    if (!consumer?.name) {
      return;
    }
    const consumerRank = USER_LEVEL_RANK[normalizeUserLevel(consumer.portalUserLevel)];
    if (consumerRank >= minRank && !result.includes(consumer.name)) {
      result.push(consumer.name);
    }
  });
  return result.sort((a, b) => a.localeCompare(b));
};
