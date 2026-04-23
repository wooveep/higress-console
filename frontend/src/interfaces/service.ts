export interface Service {
  name: string;
  namespace: string;
  port?: number;
  endpoints: string[];
  [propName: string]: any;
}

export interface ServiceResponse {
  data: Service[];
  pageNum: number;
  pageSize: number;
  total: number;
}

// Keep raw service names unchanged in API payloads and configs.
const SERVICE_DISPLAY_NAME_MAP: Record<string, string> = {
  'aigateway-console.dns': 'aigateway-console.dns',
};

export function getServiceDisplayName(name?: string): string {
  if (!name) {
    return '';
  }
  return SERVICE_DISPLAY_NAME_MAP[name] || name;
}

export function serviceToString(service: Service): string {
  if (!service) {
    return '-';
  }
  const name = service.name || '-';
  return service.port != null ? `${name}:${service.port}` : name;
}
