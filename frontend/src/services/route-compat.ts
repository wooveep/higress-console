import request from './request';

export interface GatewayRouteRecord {
  name: string;
  version?: string;
  domains?: string[];
  methods?: string[];
  services?: Array<{
    name: string;
    port?: number;
    weight?: number;
  }>;
  [key: string]: any;
}

export interface GatewayRouteListResponse {
  data: GatewayRouteRecord[];
  pageNum?: number;
  pageSize?: number;
  total?: number;
}

export const getGatewayRoutesCompat = (): Promise<GatewayRouteListResponse> => {
  return request.get<any, GatewayRouteListResponse>('/v1/routes');
};

export const getGatewayRouteDetailCompat = (name: string): Promise<GatewayRouteRecord> => {
  return request.get<any, GatewayRouteRecord>(`/v1/routes/${name}`);
};

export const addGatewayRouteCompat = (payload: GatewayRouteRecord): Promise<any> => {
  return request.post('/v1/routes', payload);
};

export const updateGatewayRouteCompat = (payload: GatewayRouteRecord): Promise<any> => {
  return request.put(`/v1/routes/${payload.name}`, payload);
};

export const deleteGatewayRouteCompat = (name: string): Promise<any> => {
  return request.delete(`/v1/routes/${name}`);
};
