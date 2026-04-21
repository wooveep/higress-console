# AIGateway Console 后端业务逻辑线梳理

## 1. 业务线总览

为便于按 Go + GoFrame 的方式重写，后端不再按“原目录”理解，而按 4 条业务主线理解：

1. 系统与会话线
2. 网关资源线
3. Portal 业务线
4. 投影与调和线

这 4 条线共享两类底座：

- 外部系统：Kubernetes、Portal PostgreSQL、Grafana
- 通用能力：统一响应、鉴权、配置、日志、trace、任务调度

## 2. 系统与会话线

### 2.1 核心目标

- 让控制台能启动、登录、初始化、读取系统信息。
- 提供全局配置读取与默认资源初始化。

### 2.2 关键入口

- `SessionController`
- `SystemController`
- `UserController`
- `HealthzController`
- `LandingController`
- `DashboardController`

### 2.3 关键服务

- `SessionServiceImpl`
- `SystemServiceImpl`
- `ConfigServiceImpl`
- `DashboardServiceImpl`

### 2.4 逻辑链

```text
登录请求
  -> SessionController
  -> SessionServiceImpl
  -> Kubernetes Secret 读取管理员配置
  -> Cookie / Basic Auth 校验
```

```text
系统初始化
  -> SystemController
  -> SystemServiceImpl
  -> SessionService / ConfigService
  -> TlsCertificateService + DomainService + RouteService
  -> 默认证书 / 默认域名 / 默认路由
```

### 2.5 GoFrame 重写思路

- 这一条线先不追求业务全量，而先搭骨架。
- 在 GoFrame 中拆为：
  - `internal/controller/system`
  - `internal/controller/session`
  - `internal/service/platform`
  - `internal/middleware`
- 先完成：
  - 统一响应
  - trace / request id
  - 健康检查
  - 系统信息接口
  - 静态 landing 托管
- 再迁移登录态、管理员 Secret、初始化逻辑。

## 3. 网关资源线

### 3.1 核心目标

- 管理 K8s 中的网关配置资源。
- 包括 Route、Domain、TLS、Service、Wasm、MCP、AI Route、Provider 等。

### 3.2 关键入口

- `RoutesController`
- `DomainsController`
- `ServicesController`
- `ServiceSourceController`
- `ProxyServerController`
- `TlsCertificatesController`
- `WasmPluginsController`
- `WasmPluginInstancesController`
- `controller/ai/AiRoutesController`
- `controller/ai/LlmProvidersController`
- `controller/mcp/McpServerController`

### 3.3 关键服务链

```text
Controller
  -> Java SDK Service
     -> KubernetesClientService
     -> KubernetesModelConverter
     -> ConsumerService / WasmPluginService / McpServerService / ...
```

### 3.4 典型逻辑

#### Route

```text
RoutesController
  -> RouteService
  -> ingress2Route / route2Ingress
  -> allow-list / consumer-level auth 同步
  -> K8s Ingress 增删改查
```

#### MCP

```text
McpServerController
  -> McpServerService
  -> ConfigMap / route / consumerAuthInfo
  -> MCP 访问配置拼装
```

#### AI Route / Provider

```text
AiRoutesController / LlmProvidersController
  -> AiRouteService / LlmProviderService
  -> ConfigMap / Wasm 插件 / K8s 资源转换
```

### 3.5 GoFrame 重写思路

- 这一条线不能先“抄 Controller”，要先抽象客户端与领域接口。
- 建议结构：
  - `utility/clients/k8s`
  - `internal/service/gateway`
  - `internal/controller/{routes,domains,plugins,mcp,ai}`
- 先做：
  - K8s client interface
  - Route/Domain/Plugin/MCP 的资源模型映射接口
  - fake client 测试
- 再迁业务控制器。

## 4. Portal 业务线

### 4.1 核心目标

- 控制面直接管理 Portal 共享 PostgreSQL 中的用户、组织、邀请码、模型资产、账务和统计。

### 4.2 关键入口

- `ConsumersController`
- `OrganizationController`
- `PortalInviteCodeController`
- `PortalStatsController`
- `AssetGrantController`
- `AiQuotaController`
- `AiSensitiveWordController`
- `controller/ai/AgentCatalogController`
- `controller/ai/ModelAssetsController`

### 4.3 关键 DB 服务

- `PortalUserJdbcService`
- `PortalOrganizationJdbcService`
- `PortalInviteCodeJdbcService`
- `PortalModelAssetJdbcService`
- `PortalBillingQuotaJdbcService`
- `PortalUsageStatsService`
- `AiSensitiveWordJdbcService`

### 4.4 典型逻辑

#### Consumer

```text
ConsumersController
  -> PortalConsumerService
     -> PortalUserJdbcService
     -> PortalConsumerProjectionService
     -> PortalConsumerLevelAuthReconcileService
     -> ConsumerService(key-auth projection)
```

#### 组织

```text
OrganizationController
  -> PortalOrganizationService
     -> PortalOrganizationJdbcService
     -> PortalUserJdbcService
     -> projection/reconcile
```

#### 模型资产

```text
ModelAssetsController
  -> PortalModelAssetJdbcService
  -> LlmProviderService
  -> Asset grant / published binding / pricing version
```

### 4.5 GoFrame 重写思路

- Portal 业务线是最适合 GoFrame 的一条线，因为其真相源主要是 PostgreSQL。
- 建议结构：
  - `internal/dao`
  - `internal/model/do`
  - `internal/model/entity`
  - `utility/clients/portaldb`
  - `internal/service/portal`
- 迁移原则：
  - 不继续沿用原生 JDBC。
  - 统一改成 GoFrame ORM + DO。
  - 所有业务规则保留在 `service/`，不堆进 Controller。

## 5. 投影与调和线

### 5.1 核心目标

- 把 DB 真相源与网关运行态收敛一致。
- 把 AI/插件/等级授权等策略周期性调和。

### 5.2 关键任务

- `PortalConsumerProjectionService`
- `PortalConsumerLevelAuthReconcileService`
- `AiSensitiveWordProjectionService`
- `AiPluginExecutionOrderReconcileService`

### 5.3 典型逻辑

```text
Portal DB 用户/Key
  -> projection service
  -> key-auth Consumer 投影
  -> disabled / revoked 处理
```

```text
组织/用户等级变更
  -> reconcile service
  -> Route / AI Route / MCP allow-list 重建
```

```text
AI 脱敏 DB 规则
  -> projection service
  -> WasmPluginInstance 配置更新
```

### 5.4 GoFrame 重写思路

- 统一迁到 `internal/job`，每个任务暴露两类入口：
  - cron 调度入口
  - service 手动触发入口
- 要求：
  - 幂等
  - 可手工触发
  - 带 trace / trigger 来源
  - 能写指标和日志

## 6. 按 Go / GoFrame 的开发理解项目

### 6.1 不再按“Controller 对 Controller”重写

正确方式是先分层：

- API 请求对象：`api/.../v1`
- 领域服务：`internal/service/...`
- 数据访问：`internal/dao + do + entity`
- 外部系统：`utility/clients/...`
- 调度任务：`internal/job`

### 6.2 先画依赖，再写代码

建议每个迁移子模块都先回答 5 个问题：

1. 真相源是 DB、K8s 还是 ConfigMap？
2. 读写路径是否相同？
3. 是否有投影/调和？
4. 是否有前端契约必须保持？
5. 是否需要契约测试样本？

### 6.3 领域切分优先于技术切分

- `system/session` 是平台底座。
- `gateway` 是 K8s 资源域。
- `portal` 是 DB 业务域。
- `jobs` 是收敛域。

这样拆后，后续 GoFrame 开发会更接近可维护的 Go 代码，而不是 Java 包结构翻译。
