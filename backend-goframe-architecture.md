# AIGateway Console GoFrame 目标架构

## 1. 目标

- 用 Go + GoFrame 接管 `aigateway-console/backend`。
- 旧 Java 后端保留在 `backend-java-legacy/` 作为迁移参考，不再作为主实现。
- 新后端按业务线组织，不复制 Java 的 `console + sdk` 包结构。

## 2. 目标目录

```text
backend/
  api/
  internal/
    cmd/
    consts/
    controller/
    middleware/
    job/
    dao/
    model/
      do/
      entity/
      response/
    service/
      platform/
      gateway/
      portal/
      jobs/
  utility/
    clients/
      k8s/
      portaldb/
      grafana/
  manifest/config/
  resource/public/
  test/contracts/
```

## 3. 分层原则

### 3.1 API 层

- 路径、方法、请求/响应对象都定义在 `api/.../v1`。
- 只做参数绑定与接口说明，不放业务。

### 3.2 Controller 层

- 负责：
  - 参数校验进入点
  - 调用领域服务
  - 把服务返回值交给 GoFrame 统一响应中间件
- 不负责：
  - SQL
  - K8s 资源转换
  - 跨域编排细节

### 3.3 Service 层

- 所有业务逻辑直接放在 `internal/service/` 下的子域中。
- 不新增 `logic/` 层。
- 服务职责：
  - 业务规则
  - 事务边界
  - 外部客户端组合调用
  - 任务触发与幂等

### 3.4 DAO / DO / Entity 层

- 所有 Portal DB 访问统一收口到 GoFrame ORM。
- 约束：
  - 查询和持久化统一通过 DAO。
  - Update/Insert 数据统一走 DO 对象。
  - 不在 service 中直接拼裸 SQL，除非明确为复杂报表查询并附说明。

### 3.5 外部系统客户端层

- `utility/clients/k8s`
  - 对接 K8s 资源。
  - 提供接口和 fake 实现入口。
- `utility/clients/portaldb`
  - 留给跨库访问或复杂查询辅助。
- `utility/clients/grafana`
  - 承接 Grafana 代理能力。

### 3.6 任务层

- `internal/job`
  - 统一放 cron 任务与调和入口。
- 原则：
  - 每个 job 都要能手工触发。
  - 每个 job 都要写明真相源、目标态、幂等策略。

## 4. 模块边界

### 4.1 `platform`

- 负责：
  - healthz
  - system info/config
  - dashboard
  - admin session
  - 静态 landing
  - 通用配置和启动信息

### 4.2 `gateway`

- 负责：
  - route
  - domain
  - service/service-source
  - proxy server
  - tls certificate
  - wasm plugin / instance
  - ai route / provider
  - mcp server

### 4.3 `portal`

- 负责：
  - consumer
  - 组织与账号
  - invite code
  - asset grant
  - model assets
  - usage stats
  - ai quota
  - ai sensitive words

### 4.4 `jobs`

- 负责：
  - consumer projection
  - level auth reconcile
  - ai sensitive projection
  - ai plugin execution order reconcile

## 5. 命名规范

- 新 Go 包、服务名、配置名、文档名优先使用 `aigateway`，避免继续扩大 `higress` 显式命名。
- 对外接口策略：
  - 保留 `/v1`、`/session`、`/system` 等通用路径前缀。
  - 显式包含 `higress` 的命名，在切换阶段统一改成 `aigateway`。
  - 示例：`/system/aigateway-config` -> `/system/aigateway-config`
- 兼容约束：
  - Portal 共库表名、Higress CRD、插件运行时协议在 v1 不动。

## 6. 中间件与通用能力

- `internal/middleware`
  - `trace`：生成或透传 request id
  - `auth`：管理员会话 / 匿名放行策略
  - `access_log`：记录请求耗时和关键信息
  - 统一响应：使用 GoFrame handler response 中间件
- 错误策略：
  - 业务错误要可追踪、可分层。
  - Controller 不手工包装每个响应。

## 7. 测试策略

- `go test ./...` 是每阶段最低门槛。
- 分类：
  - service 单元测试
  - DAO + sqlmock
  - PostgreSQL 集成测试
  - K8s fake client 测试
  - contract fixtures
  - job 幂等测试

## 8. 迁移顺序

### P0

- 冻结分析、业务线、迁移边界、任务拆分。

### P1

- 初始化 GoFrame 基座。
- 创建新的目录、配置、Docker/启动脚本、统一响应、中间件、基础接口和测试框架。

### P2

- 迁移系统、会话、配置、健康检查、Dashboard、Landing。

### P3

- 迁移 K8s 网关资源域及其适配层。

### P4

- 迁移 Portal DB 域和 DAO。

### P5

- 迁移所有定时任务和调和逻辑。

### P6

- 前端调用切换、显式 `higress` 命名切换、冒烟验证与发布收口。

## 9. 当前已落地基线

- `backend-java-legacy/`：旧 Java 后端基线已保留。
- `backend/`：新的 GoFrame 工程已初始化。
- 当前阶段先完成 P0 文档和 P1 基座，不在本次提交中强行迁移完整业务。
