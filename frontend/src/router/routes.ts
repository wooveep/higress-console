import type { RouteRecordRaw } from 'vue-router';
import {
  AppstoreOutlined,
  ApiOutlined,
  ApartmentOutlined,
  ClusterOutlined,
  CloudServerOutlined,
  ControlOutlined,
  DashboardOutlined,
  DeploymentUnitOutlined,
  GatewayOutlined,
  LockOutlined,
  NodeIndexOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
} from '@ant-design/icons-vue';

export interface AppRouteMeta {
  titleKey: string;
  auth?: boolean;
  blank?: boolean;
  navKey?: string;
  icon?: any;
  groupKey?: string;
  resourceKey?: string;
  dashboardType?: 'MAIN' | 'AI';
}

declare module 'vue-router' {
  interface RouteMeta extends AppRouteMeta {}
}

export interface NavItem {
  key: string;
  titleKey: string;
  icon?: any;
  path?: string;
  children?: NavItem[];
}

export const navItems: NavItem[] = [
  {
    key: 'dashboard',
    titleKey: 'menu.dashboard',
    icon: DashboardOutlined,
    path: '/dashboard',
  },
  {
    key: 'gateway',
    titleKey: 'index.title',
    icon: GatewayOutlined,
    children: [
      { key: 'service-source', titleKey: 'menu.serviceSources', icon: ClusterOutlined, path: '/service-source' },
      { key: 'service', titleKey: 'menu.serviceList', icon: DeploymentUnitOutlined, path: '/service' },
      { key: 'route', titleKey: 'menu.routeConfig', icon: ApartmentOutlined, path: '/route' },
      { key: 'domain', titleKey: 'menu.domainManagement', icon: ApiOutlined, path: '/domain' },
      { key: 'tls-certificate', titleKey: 'menu.certManagement', icon: SafetyCertificateOutlined, path: '/tls-certificate' },
      { key: 'consumer', titleKey: 'menu.consumerManagement', icon: ClusterOutlined, path: '/consumer' },
      { key: 'plugin', titleKey: 'menu.pluginManagement', icon: AppstoreOutlined, path: '/plugin' },
      { key: 'system', titleKey: 'menu.systemSettings', icon: SettingOutlined, path: '/system' },
    ],
  },
  {
    key: 'ai',
    titleKey: 'menu.aiServiceManagement',
    icon: RobotOutlined,
    children: [
      { key: 'ai-provider', titleKey: 'menu.llmProviderManagement', icon: CloudServerOutlined, path: '/ai/provider' },
      { key: 'ai-model-assets', titleKey: 'menu.modelAssetManagement', icon: NodeIndexOutlined, path: '/ai/model-assets' },
      { key: 'ai-agent-catalog', titleKey: 'menu.agentCatalogManagement', icon: AppstoreOutlined, path: '/ai/agent-catalog' },
      { key: 'ai-route', titleKey: 'menu.aiRouteManagement', icon: ControlOutlined, path: '/ai/route' },
      { key: 'ai-quota', titleKey: 'menu.aiQuotaManagement', icon: ClusterOutlined, path: '/ai/quota' },
      { key: 'ai-sensitive', titleKey: 'menu.aiSensitiveManagement', icon: LockOutlined, path: '/ai/sensitive' },
      { key: 'ai-dashboard', titleKey: 'menu.aiDashboard', icon: DashboardOutlined, path: '/ai/dashboard' },
      { key: 'mcp-list', titleKey: 'menu.mcpConfigurations', icon: ApiOutlined, path: '/mcp/list' },
    ],
  },
];

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/login',
    component: () => import('@/views/auth/LoginPage.vue'),
    meta: { titleKey: 'login.title', blank: true },
  },
  {
    path: '/init',
    component: () => import('@/views/auth/InitPage.vue'),
    meta: { titleKey: 'init.title', blank: true },
  },
  {
    path: '/dashboard',
    component: () => import('@/views/dashboard/DashboardPage.vue'),
    meta: { titleKey: 'menu.dashboard', auth: true, navKey: 'dashboard', dashboardType: 'MAIN' },
  },
  {
    path: '/service-source',
    component: () => import('@/views/gateway/ServiceSourcePage.vue'),
    meta: { titleKey: 'menu.serviceSources', auth: true, navKey: 'service-source' },
  },
  {
    path: '/service',
    component: () => import('@/views/gateway/ServiceListPage.vue'),
    meta: { titleKey: 'menu.serviceList', auth: true, navKey: 'service' },
  },
  {
    path: '/service/config',
    component: () => import('@/views/plugin/PluginPage.vue'),
    meta: { titleKey: 'menu.pluginManagement', auth: true, navKey: 'service' },
  },
  {
    path: '/route',
    component: () => import('@/views/gateway/RoutePage.vue'),
    meta: { titleKey: 'menu.routeConfig', auth: true, navKey: 'route' },
  },
  {
    path: '/route/config',
    component: () => import('@/views/plugin/PluginPage.vue'),
    meta: { titleKey: 'menu.pluginManagement', auth: true, navKey: 'route' },
  },
  {
    path: '/domain',
    component: () => import('@/views/gateway/DomainPage.vue'),
    meta: { titleKey: 'menu.domainManagement', auth: true, navKey: 'domain' },
  },
  {
    path: '/domain/config',
    component: () => import('@/views/plugin/PluginPage.vue'),
    meta: { titleKey: 'menu.pluginManagement', auth: true, navKey: 'domain' },
  },
  {
    path: '/tls-certificate',
    component: () => import('@/views/gateway/TlsCertificatePage.vue'),
    meta: { titleKey: 'menu.certManagement', auth: true, navKey: 'tls-certificate' },
  },
  {
    path: '/consumer',
    component: () => import('@/views/consumer/ConsumerPage.vue'),
    meta: { titleKey: 'menu.consumerManagement', auth: true, navKey: 'consumer' },
  },
  {
    path: '/plugin',
    component: () => import('@/views/plugin/PluginPage.vue'),
    meta: { titleKey: 'menu.pluginManagement', auth: true, navKey: 'plugin' },
  },
  {
    path: '/system',
    component: () => import('@/views/system/SystemPage.vue'),
    meta: { titleKey: 'menu.systemSettings', auth: true, navKey: 'system' },
  },
  {
    path: '/system/jobs',
    component: () => import('@/views/system/SystemJobsPage.vue'),
    meta: { titleKey: 'menu.systemSettings', auth: true, navKey: 'system' },
  },
  {
    path: '/user/changePassword',
    component: () => import('@/views/system/ChangePasswordPage.vue'),
    meta: { titleKey: 'user.changePassword.title', auth: true },
  },
  {
    path: '/ai/provider',
    component: () => import('@/views/ai/ProviderPage.vue'),
    meta: { titleKey: 'menu.llmProviderManagement', auth: true, navKey: 'ai-provider' },
  },
  {
    path: '/ai/model-assets',
    component: () => import('@/views/ai/ModelAssetsPage.vue'),
    meta: { titleKey: 'menu.modelAssetManagement', auth: true, navKey: 'ai-model-assets' },
  },
  {
    path: '/ai/agent-catalog',
    component: () => import('@/views/ai/AgentCatalogPage.vue'),
    meta: { titleKey: 'menu.agentCatalogManagement', auth: true, navKey: 'ai-agent-catalog' },
  },
  {
    path: '/ai/route',
    component: () => import('@/views/ai/AiRoutePage.vue'),
    meta: { titleKey: 'menu.aiRouteManagement', auth: true, navKey: 'ai-route' },
  },
  {
    path: '/ai/route/config',
    component: () => import('@/views/plugin/PluginPage.vue'),
    meta: { titleKey: 'menu.pluginManagement', auth: true, navKey: 'ai-route' },
  },
  {
    path: '/ai/quota',
    component: () => import('@/views/ai/AiQuotaPage.vue'),
    meta: { titleKey: 'menu.aiQuotaManagement', auth: true, navKey: 'ai-quota' },
  },
  {
    path: '/ai/sensitive',
    component: () => import('@/views/ai/AiSensitivePage.vue'),
    meta: { titleKey: 'menu.aiSensitiveManagement', auth: true, navKey: 'ai-sensitive' },
  },
  {
    path: '/ai/dashboard',
    component: () => import('@/views/dashboard/DashboardPage.vue'),
    meta: { titleKey: 'menu.aiDashboard', auth: true, navKey: 'ai-dashboard', dashboardType: 'AI' },
  },
  {
    path: '/mcp/list',
    component: () => import('@/views/mcp/McpListPage.vue'),
    meta: { titleKey: 'menu.mcpConfigurations', auth: true, navKey: 'mcp-list' },
  },
  {
    path: '/mcp/detail/:name?',
    component: () => import('@/views/mcp/McpDetailPage.vue'),
    meta: { titleKey: 'menu.mcpConfigurations', auth: true, navKey: 'mcp-list' },
  },
  {
    path: '/:pathMatch(.*)*',
    component: () => import('@/views/common/NotFoundPage.vue'),
    meta: { titleKey: 'misc.error', blank: true },
  },
];
