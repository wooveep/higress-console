export function splitLines(value?: string | null) {
  return (value || '')
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean);
}

export function joinLines(value?: Array<string | number | null | undefined>) {
  return (value || [])
    .filter((item) => item !== null && item !== undefined && String(item).trim())
    .map((item) => String(item))
    .join('\n');
}

export function formatDateTime(value?: string | number | null) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  return date.toLocaleString('zh-CN', { hour12: false });
}

export function safeParseJson<T>(value: string, fallback: T): T {
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
}

export function stringifyPretty(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}
