# AIGateway Console 项目图谱（2026-03 版）

## 1. 领域边界
- 网关配置领域（K8s CRD）：`Route`、`Domain`、`WasmPlugin`、`MCP`、Service Source 等。
- 用户身份领域（Portal DB）：`portal_user`、`portal_api_key`、`portal_invite_code`。

## 2. 关键后端组件
- `ConsumersController`：`/v1/consumers`，组织架构用户管理主入口。
- `PortalConsumerService`：Consumer 主业务服务（DB 真源读写、状态、密码重置）。
- `PortalUserJdbcService`：直接 JDBC 访问 Portal 库。
- `PortalConsumerProjectionService`：DB -> `key-auth` consumers 周期投影 + 历史回填。
- `PortalInviteCodeController` + `PortalInviteCodeJdbcService`：邀请码管理接口。
- `AiQuotaServiceImpl`：消费者校验改为查 DB 真源（不再依赖 key-auth 列表）。

## 3. Consumer API（Console）
- `GET /v1/consumers`：分页用户列表（来自 `portal_user`）。
- `POST /v1/consumers`：创建用户；兼容接收 `credentials` 但不再持久化写入。
- `PUT /v1/consumers/{name}`：更新用户；`credentials` 输入忽略（日志告警 deprecate）。
- `PATCH /v1/consumers/{name}/status`：`active|pending|disabled` 状态切换。
- `DELETE /v1/consumers/{name}`：软删除语义（置 disabled + 禁用 key）。
- `POST /v1/consumers/{name}/password/reset`：生成并返回 16 位临时密码（仅回显一次）。
- `GET /v1/consumers/departments`：来自 `portal_user` 去重部门。
- `POST /v1/consumers/departments`：兼容保留（废弃）。

## 4. Invite Code API（Console）
- `POST /v1/portal/invite-codes`：生成邀请码（默认 16 位，默认 7 天）。
- `GET /v1/portal/invite-codes`：分页查询。
- `PATCH /v1/portal/invite-codes/{code}`：状态切换（`active` / `disabled`）。

## 5. 运行时一致性策略
- DB 为真源，`key-auth` 为投影缓存。
- 启动时可回填历史 key-auth 数据（用户回填为 `pending`）。
- 投影周期默认 30 秒，可配置 orphan cleanup（默认关闭）。
- `disabled` 用户会触发 API Key 全部禁用并在投影侧撤销。

## 6. 前端核心页面
- 组织架构页：用户树、启用/禁用、密码重置、邀请码管理、只读 key 摘要。
- Route/AI Route/MCP 表单：消费者下拉读取 `/v1/consumers`，并集保留历史值。
- MCP 创建：前端按 RFC1123 小写规则校验名称。

## 7. 当前待修问题（2026-04-18）
- 本地开发环境链路仍有隐式运行时修复：
  - console 本地镜像构建默认依赖的 `backend/resource/public/plugin/plugins.properties` 缺失；
  - `higress-config` 下发的 `mcpServer.redis` 配置未自动包含密码；
  - 内置 `ai-proxy.internal` WasmPlugin 资源没有稳定的模块 URL / 版本映射，`gateway` 冷启动可能卡在 Wasm 初始化。
- 模型资产“发布绑定”接口存在 PostgreSQL 兼容问题：
  - `POST /v1/ai/model-assets/:assetId/bindings/:bindingId/publish`
  - 当前报错：`operator does not exist: boolean = integer (SQLSTATE 42883)`。
  - 同类问题也存在于 AI 脱敏、AI 配额的布尔字段读写链路，需要一起收口。
- Console 前端缺少黑盒自动化点击验证，当前页面级回归主要依赖人工打开页面确认。
