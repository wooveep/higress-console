import yaml from 'js-yaml';
import type {
  HigressGlobalConfigFormState,
  HigressGlobalConfigParseResult,
  TracingBackendKind,
} from '@/interfaces/system';

export const HIGRESS_COMPRESSION_LEVELS = [
  'BEST_COMPRESSION',
  'BEST_SPEED',
  'COMPRESSION_LEVEL_1',
  'COMPRESSION_LEVEL_2',
  'COMPRESSION_LEVEL_3',
  'COMPRESSION_LEVEL_4',
  'COMPRESSION_LEVEL_5',
  'COMPRESSION_LEVEL_6',
  'COMPRESSION_LEVEL_7',
  'COMPRESSION_LEVEL_8',
  'COMPRESSION_LEVEL_9',
] as const;

export const HIGRESS_COMPRESSION_STRATEGIES = [
  'DEFAULT_STRATEGY',
  'FILTERED',
  'HUFFMAN_ONLY',
  'RLE',
  'FIXED',
] as const;

const TRACING_BACKEND_KINDS: TracingBackendKind[] = ['skywalking', 'zipkin', 'opentelemetry'];

export function createDefaultHigressGlobalConfigFormState(): HigressGlobalConfigFormState {
  return {
    addXRealIpHeader: false,
    disableXEnvoyHeaders: false,
    tracing: {
      enable: false,
      sampling: 100,
      timeout: 500,
      backendKind: 'none',
      skywalking: {
        service: '',
        port: '11800',
        accessToken: '',
      },
      zipkin: {
        service: '',
        port: '9411',
        accessToken: '',
      },
      opentelemetry: {
        service: '',
        port: '',
        accessToken: '',
      },
    },
    gzip: {
      enable: true,
      minContentLength: 1024,
      contentType: [
        'text/html',
        'text/css',
        'text/plain',
        'text/xml',
        'application/json',
        'application/javascript',
        'application/xhtml+xml',
        'image/svg+xml',
      ],
      disableOnEtagHeader: true,
      memoryLevel: 5,
      windowBits: 12,
      chunkSize: 4096,
      compressionLevel: 'BEST_COMPRESSION',
      compressionStrategy: 'DEFAULT_STRATEGY',
    },
    downstream: {
      connectionBufferLimits: 32768,
      idleTimeout: 180,
      maxRequestHeadersKb: 60,
      routeTimeout: 0,
      http2: {
        initialConnectionWindowSize: 1048576,
        initialStreamWindowSize: 65535,
        maxConcurrentStreams: 100,
      },
    },
    upstream: {
      connectionBufferLimits: 10485760,
      idleTimeout: 10,
    },
  };
}

export function parseHigressGlobalConfig(rawYaml: string): HigressGlobalConfigParseResult {
  const root = parseHigressYamlRoot(rawYaml);
  return {
    knownConfig: extractKnownConfig(root),
    rawYaml: serializeHigressRoot(root),
    unknownTree: extractUnknownTree(root),
  };
}

export function parseHigressYamlRoot(rawYaml: string): Record<string, any> {
  if (!rawYaml.trim()) {
    throw new Error('YAML 不能为空');
  }
  const parsed = yaml.load(rawYaml);
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('顶层配置必须是 YAML 对象');
  }
  return parsed as Record<string, any>;
}

export function serializeHigressRoot(root: Record<string, any>): string {
  const dumped = yaml.dump(root ?? {}, {
    noRefs: true,
    lineWidth: 120,
    sortKeys: false,
  });
  return dumped.endsWith('\n') ? dumped : `${dumped}\n`;
}

export function mergeKnownConfigIntoRoot(baseRoot: Record<string, any>, formState: HigressGlobalConfigFormState) {
  const next = cloneRoot(baseRoot);

  next.addXRealIpHeader = formState.addXRealIpHeader;
  next.disableXEnvoyHeaders = formState.disableXEnvoyHeaders;

  const tracing = ensureObject(next, 'tracing');
  tracing.enable = formState.tracing.enable;
  tracing.sampling = sanitizeInteger(formState.tracing.sampling);
  tracing.timeout = sanitizeInteger(formState.tracing.timeout);
  TRACING_BACKEND_KINDS.forEach((kind) => {
    if (formState.tracing.backendKind !== kind) {
      delete tracing[kind];
      return;
    }
    const backend = ensureObject(tracing, kind);
    const source = formState.tracing[kind];
    backend.service = source.service.trim();
    backend.port = source.port.trim();
    if (kind === 'skywalking') {
      if (source.accessToken.trim()) {
        backend.access_token = source.accessToken.trim();
      } else {
        delete backend.access_token;
      }
    } else {
      delete backend.access_token;
    }
  });
  if (formState.tracing.backendKind === 'none') {
    TRACING_BACKEND_KINDS.forEach((kind) => {
      delete tracing[kind];
    });
  }

  const gzip = ensureObject(next, 'gzip');
  gzip.enable = formState.gzip.enable;
  gzip.minContentLength = sanitizeInteger(formState.gzip.minContentLength);
  gzip.contentType = sanitizeStringArray(formState.gzip.contentType);
  gzip.disableOnEtagHeader = formState.gzip.disableOnEtagHeader;
  gzip.memoryLevel = sanitizeInteger(formState.gzip.memoryLevel);
  gzip.windowBits = sanitizeInteger(formState.gzip.windowBits);
  gzip.chunkSize = sanitizeInteger(formState.gzip.chunkSize);
  gzip.compressionLevel = formState.gzip.compressionLevel;
  gzip.compressionStrategy = formState.gzip.compressionStrategy;

  const downstream = ensureObject(next, 'downstream');
  downstream.connectionBufferLimits = sanitizeInteger(formState.downstream.connectionBufferLimits);
  downstream.idleTimeout = sanitizeInteger(formState.downstream.idleTimeout);
  downstream.maxRequestHeadersKb = sanitizeInteger(formState.downstream.maxRequestHeadersKb);
  downstream.routeTimeout = sanitizeInteger(formState.downstream.routeTimeout);

  const http2 = ensureObject(downstream, 'http2');
  http2.initialConnectionWindowSize = sanitizeInteger(formState.downstream.http2.initialConnectionWindowSize);
  http2.initialStreamWindowSize = sanitizeInteger(formState.downstream.http2.initialStreamWindowSize);
  http2.maxConcurrentStreams = sanitizeInteger(formState.downstream.http2.maxConcurrentStreams);

  const upstream = ensureObject(next, 'upstream');
  upstream.connectionBufferLimits = sanitizeInteger(formState.upstream.connectionBufferLimits);
  upstream.idleTimeout = sanitizeInteger(formState.upstream.idleTimeout);

  return next;
}

export function validateHigressConfig(root: Record<string, any>, formState: HigressGlobalConfigFormState): string[] {
  const errors: string[] = [];

  if (typeof root.addXRealIpHeader !== 'undefined' && typeof root.addXRealIpHeader !== 'boolean') {
    errors.push('addXRealIpHeader 必须是布尔值');
  }
  if (typeof root.disableXEnvoyHeaders !== 'undefined' && typeof root.disableXEnvoyHeaders !== 'boolean') {
    errors.push('disableXEnvoyHeaders 必须是布尔值');
  }

  const tracing = asObject(root.tracing);
  if (root.tracing && !tracing) {
    errors.push('tracing 必须是对象');
  }
  if (tracing) {
    if (!Number.isFinite(formState.tracing.sampling) || formState.tracing.sampling < 0 || formState.tracing.sampling > 100) {
      errors.push('tracing.sampling 必须在 0 到 100 之间');
    }
    if (!Number.isFinite(formState.tracing.timeout) || formState.tracing.timeout <= 0) {
      errors.push('tracing.timeout 必须大于 0');
    }
    const backendCount = TRACING_BACKEND_KINDS.filter((kind) => asObject(tracing[kind])).length;
    if (formState.tracing.enable && backendCount !== 1) {
      errors.push('tracing 启用时必须且只能配置一个后端');
    }
    TRACING_BACKEND_KINDS.forEach((kind) => {
      const backend = asObject(tracing[kind]);
      if (!backend) {
        return;
      }
      const service = String(backend.service || '').trim();
      const port = String(backend.port || '').trim();
      if (!service) {
        errors.push(`tracing.${kind}.service 不能为空`);
      }
      if (!port) {
        errors.push(`tracing.${kind}.port 不能为空`);
      }
    });
  }

  const gzip = asObject(root.gzip);
  if (root.gzip && !gzip) {
    errors.push('gzip 必须是对象');
  }
  if (gzip) {
    if (!Number.isFinite(formState.gzip.minContentLength) || formState.gzip.minContentLength <= 0) {
      errors.push('gzip.minContentLength 必须大于 0');
    }
    if (sanitizeStringArray(formState.gzip.contentType).length === 0) {
      errors.push('gzip.contentType 至少保留一个 content-type');
    }
    if (!Number.isFinite(formState.gzip.memoryLevel) || formState.gzip.memoryLevel < 1 || formState.gzip.memoryLevel > 9) {
      errors.push('gzip.memoryLevel 必须在 1 到 9 之间');
    }
    if (!Number.isFinite(formState.gzip.windowBits) || formState.gzip.windowBits < 9 || formState.gzip.windowBits > 15) {
      errors.push('gzip.windowBits 必须在 9 到 15 之间');
    }
    if (!Number.isFinite(formState.gzip.chunkSize) || formState.gzip.chunkSize <= 0) {
      errors.push('gzip.chunkSize 必须大于 0');
    }
    if (!HIGRESS_COMPRESSION_LEVELS.includes(formState.gzip.compressionLevel as (typeof HIGRESS_COMPRESSION_LEVELS)[number])) {
      errors.push(`gzip.compressionLevel 必须是 ${HIGRESS_COMPRESSION_LEVELS.join(', ')} 之一`);
    }
    if (!HIGRESS_COMPRESSION_STRATEGIES.includes(formState.gzip.compressionStrategy as (typeof HIGRESS_COMPRESSION_STRATEGIES)[number])) {
      errors.push(`gzip.compressionStrategy 必须是 ${HIGRESS_COMPRESSION_STRATEGIES.join(', ')} 之一`);
    }
  }

  const downstream = asObject(root.downstream);
  if (root.downstream && !downstream) {
    errors.push('downstream 必须是对象');
  }
  if (downstream) {
    if (formState.downstream.connectionBufferLimits < 0) {
      errors.push('downstream.connectionBufferLimits 不能小于 0');
    }
    if (formState.downstream.idleTimeout < 0) {
      errors.push('downstream.idleTimeout 不能小于 0');
    }
    if (formState.downstream.maxRequestHeadersKb < 0 || formState.downstream.maxRequestHeadersKb > 8192) {
      errors.push('downstream.maxRequestHeadersKb 必须在 0 到 8192 之间');
    }
    if (formState.downstream.routeTimeout < 0) {
      errors.push('downstream.routeTimeout 不能小于 0');
    }
    if (formState.downstream.http2.maxConcurrentStreams < 1 || formState.downstream.http2.maxConcurrentStreams > 2147483647) {
      errors.push('downstream.http2.maxConcurrentStreams 必须在 1 到 2147483647 之间');
    }
    if (formState.downstream.http2.initialStreamWindowSize < 65535 || formState.downstream.http2.initialStreamWindowSize > 2147483647) {
      errors.push('downstream.http2.initialStreamWindowSize 必须在 65535 到 2147483647 之间');
    }
    if (formState.downstream.http2.initialConnectionWindowSize < 65535 || formState.downstream.http2.initialConnectionWindowSize > 2147483647) {
      errors.push('downstream.http2.initialConnectionWindowSize 必须在 65535 到 2147483647 之间');
    }
  }

  const upstream = asObject(root.upstream);
  if (root.upstream && !upstream) {
    errors.push('upstream 必须是对象');
  }
  if (upstream) {
    if (formState.upstream.connectionBufferLimits < 0) {
      errors.push('upstream.connectionBufferLimits 不能小于 0');
    }
    if (formState.upstream.idleTimeout < 0) {
      errors.push('upstream.idleTimeout 不能小于 0');
    }
  }

  return Array.from(new Set(errors));
}

function extractKnownConfig(root: Record<string, any>) {
  const next = createDefaultHigressGlobalConfigFormState();

  next.addXRealIpHeader = readBoolean(root.addXRealIpHeader, next.addXRealIpHeader);
  next.disableXEnvoyHeaders = readBoolean(root.disableXEnvoyHeaders, next.disableXEnvoyHeaders);

  const tracing = asObject(root.tracing);
  if (tracing) {
    next.tracing.enable = readBoolean(tracing.enable, next.tracing.enable);
    next.tracing.sampling = readNumber(tracing.sampling, next.tracing.sampling);
    next.tracing.timeout = readNumber(tracing.timeout, next.tracing.timeout);
    next.tracing.backendKind = resolveTracingBackendKind(tracing);
    TRACING_BACKEND_KINDS.forEach((kind) => {
      const backend = asObject(tracing[kind]);
      if (!backend) {
        return;
      }
      next.tracing[kind].service = readString(backend.service, '');
      next.tracing[kind].port = readString(backend.port, next.tracing[kind].port);
      next.tracing[kind].accessToken = readString(backend.access_token, '');
    });
  }

  const gzip = asObject(root.gzip);
  if (gzip) {
    next.gzip.enable = readBoolean(gzip.enable, next.gzip.enable);
    next.gzip.minContentLength = readNumber(gzip.minContentLength, next.gzip.minContentLength);
    next.gzip.contentType = readStringArray(gzip.contentType, next.gzip.contentType);
    next.gzip.disableOnEtagHeader = readBoolean(gzip.disableOnEtagHeader, next.gzip.disableOnEtagHeader);
    next.gzip.memoryLevel = readNumber(gzip.memoryLevel, next.gzip.memoryLevel);
    next.gzip.windowBits = readNumber(gzip.windowBits, next.gzip.windowBits);
    next.gzip.chunkSize = readNumber(gzip.chunkSize, next.gzip.chunkSize);
    next.gzip.compressionLevel = readString(gzip.compressionLevel, next.gzip.compressionLevel);
    next.gzip.compressionStrategy = readString(gzip.compressionStrategy, next.gzip.compressionStrategy);
  }

  const downstream = asObject(root.downstream);
  if (downstream) {
    next.downstream.connectionBufferLimits = readNumber(downstream.connectionBufferLimits, next.downstream.connectionBufferLimits);
    next.downstream.idleTimeout = readNumber(downstream.idleTimeout, next.downstream.idleTimeout);
    next.downstream.maxRequestHeadersKb = readNumber(downstream.maxRequestHeadersKb, next.downstream.maxRequestHeadersKb);
    next.downstream.routeTimeout = readNumber(downstream.routeTimeout, next.downstream.routeTimeout);
    const http2 = asObject(downstream.http2);
    if (http2) {
      next.downstream.http2.initialConnectionWindowSize = readNumber(http2.initialConnectionWindowSize, next.downstream.http2.initialConnectionWindowSize);
      next.downstream.http2.initialStreamWindowSize = readNumber(http2.initialStreamWindowSize, next.downstream.http2.initialStreamWindowSize);
      next.downstream.http2.maxConcurrentStreams = readNumber(http2.maxConcurrentStreams, next.downstream.http2.maxConcurrentStreams);
    }
  }

  const upstream = asObject(root.upstream);
  if (upstream) {
    next.upstream.connectionBufferLimits = readNumber(upstream.connectionBufferLimits, next.upstream.connectionBufferLimits);
    next.upstream.idleTimeout = readNumber(upstream.idleTimeout, next.upstream.idleTimeout);
  }

  return next;
}

function extractUnknownTree(root: Record<string, any>) {
  const next = cloneRoot(root);

  delete next.addXRealIpHeader;
  delete next.disableXEnvoyHeaders;

  pruneTracingKnownFields(asObject(next.tracing));
  pruneGzipKnownFields(asObject(next.gzip));
  pruneDownstreamKnownFields(asObject(next.downstream));
  pruneUpstreamKnownFields(asObject(next.upstream));
  cleanupEmptyObjects(next);

  return next as Record<string, unknown>;
}

function pruneTracingKnownFields(section?: Record<string, any> | null) {
  if (!section) {
    return;
  }
  delete section.enable;
  delete section.sampling;
  delete section.timeout;
  TRACING_BACKEND_KINDS.forEach((kind) => {
    const backend = asObject(section[kind]);
    if (!backend) {
      return;
    }
    delete backend.service;
    delete backend.port;
    delete backend.access_token;
    cleanupEmptyObjects(backend);
    if (Object.keys(backend).length === 0) {
      delete section[kind];
    }
  });
}

function pruneGzipKnownFields(section?: Record<string, any> | null) {
  if (!section) {
    return;
  }
  [
    'enable',
    'minContentLength',
    'contentType',
    'disableOnEtagHeader',
    'memoryLevel',
    'windowBits',
    'chunkSize',
    'compressionLevel',
    'compressionStrategy',
  ].forEach((key) => {
    delete section[key];
  });
}

function pruneDownstreamKnownFields(section?: Record<string, any> | null) {
  if (!section) {
    return;
  }
  delete section.connectionBufferLimits;
  delete section.idleTimeout;
  delete section.maxRequestHeadersKb;
  delete section.routeTimeout;
  const http2 = asObject(section.http2);
  if (http2) {
    delete http2.initialConnectionWindowSize;
    delete http2.initialStreamWindowSize;
    delete http2.maxConcurrentStreams;
    cleanupEmptyObjects(http2);
    if (Object.keys(http2).length === 0) {
      delete section.http2;
    }
  }
}

function pruneUpstreamKnownFields(section?: Record<string, any> | null) {
  if (!section) {
    return;
  }
  delete section.connectionBufferLimits;
  delete section.idleTimeout;
}

function cleanupEmptyObjects(target: Record<string, any>) {
  Object.entries(target).forEach(([key, value]) => {
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      cleanupEmptyObjects(value as Record<string, any>);
      if (Object.keys(value as Record<string, any>).length === 0) {
        delete target[key];
      }
    }
  });
}

function resolveTracingBackendKind(section: Record<string, any>): TracingBackendKind {
  for (const kind of TRACING_BACKEND_KINDS) {
    if (asObject(section[kind])) {
      return kind;
    }
  }
  return 'none';
}

function ensureObject(root: Record<string, any>, key: string) {
  const current = asObject(root[key]);
  if (current) {
    return current;
  }
  const next: Record<string, any> = {};
  root[key] = next;
  return next;
}

function asObject(value: unknown) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, any>;
}

function readBoolean(value: unknown, fallback: boolean) {
  return typeof value === 'boolean' ? value : fallback;
}

function readNumber(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
}

function readString(value: unknown, fallback: string) {
  return typeof value === 'string' ? value : fallback;
}

function readStringArray(value: unknown, fallback: string[]) {
  if (!Array.isArray(value)) {
    return fallback;
  }
  return value.filter((item): item is string => typeof item === 'string');
}

function sanitizeStringArray(value: string[]) {
  return value.map((item) => item.trim()).filter(Boolean);
}

function sanitizeInteger(value: number) {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.trunc(value);
}

function cloneRoot<T>(value: T): T {
  return JSON.parse(JSON.stringify(value ?? {})) as T;
}
