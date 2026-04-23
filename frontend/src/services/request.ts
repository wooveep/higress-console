import { h } from 'vue';
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import i18n from '@/i18n';
import { showWarningModal } from '@/lib/feedback';

export interface RequestOptions extends AxiosRequestConfig {
  skipAuthRedirect?: boolean;
  skipErrorModal?: boolean;
}

interface RequestClient {
  get<T = any, R = T>(url: string, config?: RequestOptions): Promise<R>;
  post<T = any, R = T>(url: string, data?: T, config?: RequestOptions): Promise<R>;
  put<T = any, R = T>(url: string, data?: T, config?: RequestOptions): Promise<R>;
  patch<T = any, R = T>(url: string, data?: T, config?: RequestOptions): Promise<R>;
  delete<T = any, R = T>(url: string, config?: RequestOptions): Promise<R>;
}

const rawClient: AxiosInstance = axios.create({
  timeout: 15 * 1000,
  baseURL: import.meta.env.DEV ? '/api' : '',
  headers: {
    'Content-Type': 'application/json',
  },
});

rawClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    (config.headers as any) = {
      ...(config.headers || {}),
      Authorization: token,
    };
  }

  if (config.method?.toUpperCase() === 'GET' && config.url) {
    config.url = `${config.url}${config.url.includes('?') ? '&' : '?'}ts=${Date.now()}`;
  }

  return config;
});

rawClient.interceptors.response.use(
  (response) => {
    const { data } = response;
    if (data && typeof data === 'object' && 'data' in data) {
      return data.data;
    }
    return data;
  },
  (error) => {
    const requestConfig = (error?.config ?? {}) as RequestOptions;
    const requestUrl = requestConfig.url || '';
    let message = error?.message || 'Request failed';
    let code = error?.code;

    if (error.response) {
      const { status, data } = error.response;
      code = status;

      if (status === 401) {
        if (requestUrl.includes('/login')) {
          return Promise.reject(error);
        }
        if (!requestConfig.skipAuthRedirect && !location.pathname.startsWith('/login') && !location.pathname.startsWith('/init')) {
          window.location.href = `/login?redirect=${encodeURIComponent(location.pathname + location.search)}`;
        }
        return Promise.reject(error);
      }

      const method = requestConfig.method?.toLowerCase();
      const messageKeys = [
        method ? `request.error.${status}_${method}` : '',
        `request.error.${status}`,
      ].filter(Boolean);

      for (const key of messageKeys) {
        const localized = i18n.global.t(key);
        if (localized !== key) {
          message = localized;
          break;
        }
      }

      if (data) {
        requestConfig.data = typeof data === 'string' ? data : JSON.stringify(data, null, 2);
      }
    }

    if (!requestConfig.skipErrorModal) {
      showErrorModal(message, requestConfig, code);
    }

    return Promise.reject(error);
  },
);

function showErrorModal(message: string, config: RequestOptions, code?: number) {
  showWarningModal({
    title: i18n.global.t('misc.error'),
    okText: i18n.global.t('misc.close'),
    width: 640,
    content: h('div', { class: 'portal-error-modal' }, [
      h('div', { class: 'portal-error-modal__message' }, `${code ? `${code}: ` : ''}${message}`),
      h('pre', { class: 'portal-error-modal__detail' }, JSON.stringify({
        url: config.url,
        method: config.method,
        params: config.params,
        data: config.data,
      }, null, 2)),
    ]),
  });
}

const request: RequestClient = {
  get(url, config) {
    return rawClient.get(url, config);
  },
  post(url, data, config) {
    return rawClient.post(url, data, config);
  },
  put(url, data, config) {
    return rawClient.put(url, data, config);
  },
  patch(url, data, config) {
    return rawClient.patch(url, data, config);
  },
  delete(url, config) {
    return rawClient.delete(url, config);
  },
};

export default request;
