export interface UserInfo {
  name: string;
  displayName: string;
  password?: string;
  type?: 'user' | 'admin' | 'guest';
  avatarUrl?: string;
}

export interface InitParams {
  adminUser: UserInfo;
  configs?: object;
}

export type TracingBackendKind = 'none' | 'skywalking' | 'zipkin' | 'opentelemetry';

export interface HigressTracingBackendFormState {
  service: string;
  port: string;
  accessToken: string;
}

export interface HigressTracingFormState {
  enable: boolean;
  sampling: number;
  timeout: number;
  backendKind: TracingBackendKind;
  skywalking: HigressTracingBackendFormState;
  zipkin: HigressTracingBackendFormState;
  opentelemetry: HigressTracingBackendFormState;
}

export interface HigressGzipFormState {
  enable: boolean;
  minContentLength: number;
  contentType: string[];
  disableOnEtagHeader: boolean;
  memoryLevel: number;
  windowBits: number;
  chunkSize: number;
  compressionLevel: string;
  compressionStrategy: string;
}

export interface HigressHTTP2FormState {
  initialConnectionWindowSize: number;
  initialStreamWindowSize: number;
  maxConcurrentStreams: number;
}

export interface HigressDownstreamFormState {
  connectionBufferLimits: number;
  idleTimeout: number;
  maxRequestHeadersKb: number;
  routeTimeout: number;
  http2: HigressHTTP2FormState;
}

export interface HigressUpstreamFormState {
  connectionBufferLimits: number;
  idleTimeout: number;
}

export interface HigressGlobalConfigFormState {
  addXRealIpHeader: boolean;
  disableXEnvoyHeaders: boolean;
  tracing: HigressTracingFormState;
  gzip: HigressGzipFormState;
  downstream: HigressDownstreamFormState;
  upstream: HigressUpstreamFormState;
}

export interface HigressGlobalConfigParseResult {
  knownConfig: HigressGlobalConfigFormState;
  rawYaml: string;
  unknownTree: Record<string, unknown>;
}
