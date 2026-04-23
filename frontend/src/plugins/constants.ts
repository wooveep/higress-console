export const BUILTIN_ROUTE_PLUGIN_LIST = [
  {
    key: 'rewrite',
    category: 'route',
    title: '重写',
    description: '修改请求的 Host 与 Path。',
    builtIn: true,
    enabledInAiRoute: false,
  },
  {
    key: 'headerModify',
    category: 'route',
    title: 'Header 设置',
    description: '修改请求头和响应头。',
    builtIn: true,
    enabledInAiRoute: true,
  },
  {
    key: 'cors',
    category: 'route',
    title: '跨域',
    description: '配置跨域访问规则。',
    builtIn: true,
    enabledInAiRoute: true,
  },
  {
    key: 'retries',
    category: 'route',
    title: '重试',
    description: '配置后端重试策略。',
    builtIn: true,
    enabledInAiRoute: true,
  },
] as const;

export const DEFAULT_PLUGIN_ICON = 'https://dummyimage.com/80x80/f0f5ff/9db7d9&text=PLG';
