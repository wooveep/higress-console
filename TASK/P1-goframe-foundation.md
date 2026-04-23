# P1 GoFrame 基座

- 状态：`done`
- 依赖：`P0`

## 目标

- 用 GoFrame 接管 `backend/`。
- 建立后续迁移共用的基础结构、启动入口、配置、中间件、统一响应和测试框架。

## 已完成

- [x] 原 `backend/` 已迁移到 `backend-java-legacy/`。
- [x] 已用 `gf init` 初始化新的 `backend/`。
- [x] 已建立目标目录：
  - `api/`
  - `internal/controller/`
  - `internal/service/`
  - `internal/dao/`
  - `internal/model/do/`
  - `internal/model/entity/`
  - `internal/middleware/`
  - `internal/job/`
  - `utility/clients/*`
  - `test/contracts/`
- [x] 已移除默认 `hello` 示例与 `internal/logic` 目录。
- [x] 已建立基础 `healthz` 和 `system` 接口骨架。
- [x] 已建立基础 Docker / build / start 入口。
- [x] 已建立首批单元测试骨架。
- [x] 已补 `session` 管理与管理员鉴权基础设施。
- [x] 已建立统一 `success/data/message` 响应语义与基础错误返回结构。
- [x] 已补 OpenAPI/Swagger 暴露与基础路由检查。
- [x] 已补静态 landing 页面与前端静态资源 / MCP 模板挂载规范。
- [x] 已补 Portal DB / K8s / Grafana 客户端真实配置结构与 fake 默认实现。

## 待完成

- [ ] 在真实环境 client 接入后，把错误码枚举和外部依赖错误分层进一步收口。

## 验收点

- [x] `backend/` 已成为独立 GoFrame 项目。
- [x] 旧 Java 代码仍可在 `backend-java-legacy/` 中对照。
- [x] 新后端可以执行 `go test ./...`。
- [x] 新后端可以启动并通过基础 smoke 检查。
- [x] 配置、日志、统一响应、中间件结构满足后续迁移使用。

## 单元测试

- [x] 基础平台服务测试。
- [x] 中间件测试。
- [x] 配置加载测试。
- [x] 路由启动 smoke 测试。
