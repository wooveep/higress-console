export const APP_TIME_ZONE =
  // eslint-disable-next-line @iceworks/best-practices/recommend-polyfill
  Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
export const APP_DATE_TIME_DISPLAY_FORMAT = 'YYYY-MM-DD HH:mm:ss';

interface StructuredDateTime {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
  second: number;
}

interface StructuredDateTimeSource {
  year?: number;
  month?: number;
  monthValue?: number;
  day?: number;
  dayOfMonth?: number;
  hour?: number;
  minute?: number;
  second?: number;
}

type DateTimeValue =
  | Date
  | number
  | string
  | number[]
  | StructuredDateTimeSource
  | null
  | undefined;

// eslint-disable-next-line @iceworks/best-practices/recommend-polyfill
const DATE_TIME_FORMATTER = new Intl.DateTimeFormat('en-CA', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
});

// eslint-disable-next-line @iceworks/best-practices/recommend-polyfill
const TIME_FORMATTER = new Intl.DateTimeFormat('en-GB', {
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
});

// eslint-disable-next-line @iceworks/best-practices/recommend-polyfill
const TIME_WITH_SECONDS_FORMATTER = new Intl.DateTimeFormat('en-GB', {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
});

export function formatDateTimeDisplay(value: DateTimeValue, fallback = '-'): string {
  const structured = parseStructuredDateTime(value);
  if (structured) {
    return formatStructuredDateTime(structured);
  }

  const timestamp = normalizeTimestamp(value);
  if (timestamp === null) {
    return fallback;
  }
  return formatDateTimeFromTimestamp(timestamp);
}

export function formatTimeDisplay(
  value: DateTimeValue,
  options?: {
    fallback?: string;
    includeSeconds?: boolean;
  },
): string {
  const fallback = options?.fallback ?? '-';
  const structured = parseStructuredDateTime(value);
  if (structured) {
    return formatStructuredTime(structured, options?.includeSeconds === true);
  }

  const timestamp = normalizeTimestamp(value);
  if (timestamp === null) {
    return fallback;
  }
  return formatTimeFromTimestamp(timestamp, options?.includeSeconds === true);
}

export function formatChartTimeLabel(value: DateTimeValue, rangeMs?: number, fallback = '-'): string {
  const timestamp = normalizeTimestamp(value);
  if (timestamp === null) {
    return fallback;
  }

  if (rangeMs && rangeMs > 24 * 60 * 60 * 1000) {
    return formatDateTimeFromTimestamp(timestamp).slice(5, 16);
  }
  if (rangeMs && rangeMs > 6 * 60 * 60 * 1000) {
    return formatTimeFromTimestamp(timestamp, true);
  }
  return formatTimeFromTimestamp(timestamp, false);
}

export function formatDateTimeLocalInputValue(value: DateTimeValue, fallback = ''): string {
  const structured = parseStructuredDateTime(value);
  if (structured) {
    return `${formatStructuredDateTime(structured).replace(' ', 'T').slice(0, 16)}`;
  }

  const timestamp = normalizeTimestamp(value);
  if (timestamp === null) {
    return fallback;
  }
  return formatDateTimeFromTimestamp(timestamp).replace(' ', 'T').slice(0, 16);
}

export function getNowDateTimeLocalInputValue(): string {
  return formatDateTimeLocalInputValue(Date.now());
}

export function dateTimeLocalInputToISOString(value: string | null | undefined): string | undefined {
  if (!value?.trim()) {
    return undefined;
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return undefined;
  }
  return parsed.toISOString();
}

export function normalizeTimestamp(value: DateTimeValue): number | null {
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? null : value.getTime();
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  if (typeof value !== 'string') {
    return null;
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  if (/^\d+$/.test(trimmed)) {
    const parsedNumber = Number(trimmed);
    return Number.isFinite(parsedNumber) ? parsedNumber : null;
  }

  if (parseStructuredDateTime(trimmed)) {
    return null;
  }

  const parsedTime = Date.parse(trimmed);
  return Number.isNaN(parsedTime) ? null : parsedTime;
}

function parseStructuredDateTime(value: DateTimeValue): StructuredDateTime | null {
  if (!value) {
    return null;
  }

  if (Array.isArray(value)) {
    const [year, month, day, hour = 0, minute = 0, second = 0] = value;
    if ([year, month, day].every((item) => typeof item === 'number' && Number.isFinite(item))) {
      return {
        year,
        month,
        day,
        hour: typeof hour === 'number' ? hour : 0,
        minute: typeof minute === 'number' ? minute : 0,
        second: typeof second === 'number' ? second : 0,
      };
    }
    return null;
  }

  if (typeof value === 'object' && !(value instanceof Date)) {
    const {
      year,
      monthValue,
      month,
      dayOfMonth,
      day,
      hour,
      minute,
      second,
    } = value;
    const resolvedMonth = monthValue ?? month;
    const resolvedDay = dayOfMonth ?? day;
    if ([year, resolvedMonth, resolvedDay].every((item) => typeof item === 'number' && Number.isFinite(item))) {
      return {
        year,
        month: resolvedMonth,
        day: resolvedDay,
        hour: typeof hour === 'number' ? hour : 0,
        minute: typeof minute === 'number' ? minute : 0,
        second: typeof second === 'number' ? second : 0,
      };
    }
    return null;
  }

  if (typeof value !== 'string') {
    return null;
  }

  const trimmed = value.trim();
  if (!trimmed || /z$/i.test(trimmed) || /[+-]\d{2}:?\d{2}$/.test(trimmed)) {
    return null;
  }

  const match = trimmed.match(
    /^(\d{4})-(\d{2})-(\d{2})(?:[T\s](\d{2})(?::(\d{2})(?::(\d{2}))?)?)?$/,
  );
  if (!match) {
    return null;
  }

  return {
    year: Number(match[1]),
    month: Number(match[2]),
    day: Number(match[3]),
    hour: Number(match[4] || 0),
    minute: Number(match[5] || 0),
    second: Number(match[6] || 0),
  };
}

function formatDateTimeFromTimestamp(timestamp: number): string {
  const parts = DATE_TIME_FORMATTER.formatToParts(new Date(timestamp));
  const partMap: Record<string, string> = {};
  parts.forEach((part) => {
    if (part.type !== 'literal') {
      partMap[part.type] = part.value;
    }
  });
  return `${partMap.year}-${partMap.month}-${partMap.day} ${partMap.hour}:${partMap.minute}:${partMap.second}`;
}

function formatTimeFromTimestamp(timestamp: number, includeSeconds: boolean): string {
  return (includeSeconds ? TIME_WITH_SECONDS_FORMATTER : TIME_FORMATTER).format(new Date(timestamp));
}

function formatStructuredDateTime(value: StructuredDateTime): string {
  return `${value.year}-${pad(value.month)}-${pad(value.day)} ${formatStructuredTime(value, true)}`;
}

function formatStructuredTime(value: StructuredDateTime, includeSeconds: boolean): string {
  const base = `${pad(value.hour)}:${pad(value.minute)}`;
  return includeSeconds ? `${base}:${pad(value.second)}` : base;
}

function pad(value: number): string {
  return value < 10 ? `0${value}` : String(value);
}
