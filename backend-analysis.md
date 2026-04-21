# AIGateway Console Backend 现状分析

## 1. 当前真实基线

- 当前 `aigateway-console/backend` 在重构前是 Java + Spring Boot + Maven 多模块项目，不是 GoFrame。
- 原始结构分为两个核心模块：
  - `console`：控制面 API、会话、Portal JDBC、页面静态资源托管、统一响应与定时任务。
  - `sdk`：K8s/CRD/插件/AI/MCP 领域模型与适配层，相当于 Java 版本的网关控制面 SDK。
- 当前仓库已经按计划完成目录切换：
  - `backend-java-legacy/`：保留旧 Java 后端作为迁移参考。
  - `backend/`：新的 GoFrame 后端基座。

## 2. 代码规模与结构特征

- `console` 主包下主要目录数量：
  - `controller`：约 30 个 Controller，约 186 个路由映射注解。
  - `service`：约 33 个服务/编排类。
  - `model`：约 42 个模型类。
  - `client`：Grafana 等外部客户端适配。
- `sdk` 主包下主要目录数量：
  - `service`：约 98 个服务类。
  - `service/kubernetes`：体量最大，负责 Ingress、Secret、ConfigMap、CRD、Plugin 等资源转换与访问。
  - `service/mcp`、`service/ai`、`service/consumer`：承接 MCP、AI、Consumer/Auth 等能力。

## 3. 原后端目录职责

### 3.1 `console`

- `controller/`
  - 直接暴露控制面 HTTP API。
  - 覆盖系统、登录、路由、域名、证书、插件、MCP、Consumer、组织、模型资产、AI 配额、AI 脱敏等。
- `service/`
  - 编排层，既有纯业务服务，也有带副作用的同步/调和任务。
- `service/portal/`
  - 直接访问 Portal 共享 PostgreSQL。
  - 这里是 Consumer、组织、邀请码、模型资产、账单、统计、AI 脱敏等 DB 真相源。
- `aop/`
  - `ApiStandardizationAspect` 负责统一异常包装、鉴权入口、traceId 与 DELETE 204 处理。
- `config/`
  - `SdkConfig` 用于注入 Java SDK 服务。
- `client/grafana/`
  - Grafana 仪表盘代理和数据源访问。
- `resources/`
  - `application.properties`、dashboard JSON、landing 页面、AI 敏感词默认字典等资源。

### 3.2 `sdk`

- `service/kubernetes/`
  - K8s 客户端与模型转换器。
  - 是 Route/Domain/Service/TLS/Wasm/AI Route/MCP 的底层资源入口。
- `service/consumer/`
  - Consumer、AllowList、key-auth 等运行时鉴权适配。
- `service/ai/`
  - AI Route、Provider、模型协议映射。
- `service/mcp/`
  - MCP Server 资源、ConfigMap/DB 配置转换、Consumer Auth 信息。
- `model/*`
  - 控制面 DTO / 资源模型 / 查询对象。

## 4. 关键依赖关系

### 4.1 顶层调用链

```text
Controller
  -> Console Service / Portal JDBC Service
     -> Java SDK Service / SQL / Grafana Client
        -> Kubernetes API / Portal PostgreSQL / Grafana
```

### 4.2 配置与注入链

```text
application.properties
  -> console/config/SdkConfig
     -> HigressServiceProvider
        -> RouteService / DomainService / WasmPluginService / McpServerService / ...
```

### 4.3 统一 HTTP 行为链

```text
ApiStandardizationAspect
  -> 登录态校验
  -> SessionUserHelper 注入当前用户
  -> 统一异常 -> ResponseEntity<Response<T>>
  -> traceId MDC
```

## 5. 关键运行时能力

### 5.1 会话与系统初始化

- `SessionServiceImpl`
  - 管理管理员登录态。
  - 依赖 K8s Secret 存储管理员用户名、密码和加密密钥。
- `SystemServiceImpl`
  - 负责系统初始化、默认证书、默认 Domain/Route 创建。
  - 强依赖 `TlsCertificateService`、`DomainService`、`RouteService`、`ConfigService`。

### 5.2 网关资源管理

- `RoutesController` -> `RouteService`
- `DomainsController` -> `DomainService`
- `TlsCertificatesController` -> `TlsCertificateService`
- `WasmPluginsController` / `WasmPluginInstancesController`
- `McpServerController` -> `McpServerService`
- `AiRoutesController` / `LlmProvidersController`

这一组能力的共性：

- HTTP 层很薄。
- 业务规则和资源转换主要在 SDK 中。
- 真正高风险点在 K8s 资源转换、注解读写、ConfigMap/CRD 结构。

### 5.3 Portal PostgreSQL 真相源

关键类：

- `PortalUserJdbcService`
- `PortalOrganizationJdbcService`
- `PortalInviteCodeJdbcService`
- `PortalModelAssetJdbcService`
- `PortalBillingQuotaJdbcService`
- `PortalUsageStatsService`
- `AiSensitiveWordJdbcService`

特点：

- 使用原生 JDBC，而不是 ORM。
- 表结构和 Portal 共库，必须保持迁移兼容。
- 当前已有一部分能力切到 DB 真相源，例如 Consumer、组织、模型资产、脱敏规则、账单统计。

### 5.4 投影与调和

关键类：

- `PortalConsumerProjectionService`
- `PortalConsumerLevelAuthReconcileService`
- `AiSensitiveWordProjectionService`
- `AiPluginExecutionOrderReconcileService`

特点：

- 既有启动时同步，也有定时 `@Scheduled` 周期收敛。
- 这些任务连接 DB 真相源与网关运行态，是重写中的高耦合风险区。

## 6. 测试现状

原 Java 后端已有部分测试，但覆盖并不均匀。

- `console/src/test`
  - 覆盖 AI 脱敏、部分 JDBC 服务、部分 Controller、插件执行顺序调和。
- `sdk/src/test`
  - 覆盖 Route/MCP/Wasm/Consumer/K8s Model Converter 等。

结论：

- 原有测试可作为行为参考。
- Go 重写后不能只依赖“照抄 Java 测试”，需要把契约测试、DAO 测试、K8s 适配测试和 cron 任务测试分层重建。

## 7. 迁移难点与优先级

### 7.1 优先迁移的骨架能力

- 统一响应、中间件、trace、错误语义。
- 系统/健康检查/静态资源托管。
- 配置加载、命名规范、Docker/构建入口。

### 7.2 高风险能力

- 基于 K8s CRD 的 Route/Domain/Wasm/MCP/AI Route 转换。
- Portal DB 真相源与 key-auth/allow-list 投影。
- 管理员会话和系统初始化。
- AI 脱敏和配额这类“DB + K8s + 定时任务”组合链路。

### 7.3 建议迁移方式

- 先用 GoFrame 建立统一基座和目录。
- 然后按“系统骨架 -> 网关资源 -> Portal DB -> 定时任务 -> 前端切换”推进。
- 每条主线保留行为对照文档，避免直接把 Java 包结构翻译成 Go 包结构。

## 8. 当前结论

- 这次重写不是语言层面的替换，而是控制面架构的再分层。
- 新 GoFrame 后端应明确分成：
  - HTTP/API 层
  - 领域服务层
  - DAO/Entity/DO 层
  - 外部系统客户端适配层
  - 定时任务层
- `backend-java-legacy/` 将在多个阶段持续作为行为、字段和运行时资源映射的对照基线。
