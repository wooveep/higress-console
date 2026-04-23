# P4 Portal 业务域

- 状态：`done`
- 依赖：`P1`, `P2`

## 目标

- 迁移 Portal PostgreSQL 真相源相关能力，并统一成 GoFrame DAO/DO/Entity 模式。

## 任务

- [x] 建立 Portal DB client、Schema bootstrap 与 PostgreSQL 接入骨架。
- [x] 建立 Portal DB DAO、DO、Entity 首批基础模型。
- [x] 迁移 Consumer 首批接口。
- [x] 迁移组织树与账号管理首批 CRUD。
- [x] 迁移 Invite Code。
- [x] 迁移 Asset Grant。
- [x] 迁移 Model Assets / Agent Catalog / Pricing / Stats 首批主链。
- [x] 迁移 AI Quota / AI Sensitive Word 首批 DB 能力与 HTTP 接口。

## 验收点

- [x] Portal DB 访问入口已不再依赖原生 JDBC。
- [x] Consumer/Org/Model Assets/Agent Catalog/Grant 首批接口可工作。
- [x] 与旧前端契约保持可迁移。

## 测试

- [x] Portal service sqlmock 测试。
- [x] Model Assets / Grants / Agent Catalog 服务测试。
- [x] PostgreSQL testcontainers 集成测试。
- [x] Consumer / Org / Invite Code / Model Assets / Agent Catalog / AI Quota / AI Sensitive 契约测试。

## Aftercare

- [x] `P4-AF-01`：继续把 `portal service` 从 raw SQL 向 `DAO / DO / Entity` 收口。
- [x] `P4-AF-02`：补齐 `Consumer / Org / Invite / Model Assets / Agent Catalog / AI Quota / AI Sensitive` 契约测试。
- [x] `P4-AF-03`：继续把 `Portal Stats` 从表格查询推进到更完整的图表/导出能力。

## 本轮说明

- 本轮先按第一批范围交付 `Consumer / Org / Invite Code`，并通过 `portaldb` real client + schema bootstrap 打通最小真相源链路。
- 本轮继续补上 `Asset Grant / Model Assets / Model Binding / Binding Price Version / Agent Catalog`，并补齐前端依赖的 `/v1/assets/*/grants`、`/v1/ai/model-assets*`、`/v1/ai/agent-catalog*` 接口。
- `portaldb` schema bootstrap 现已覆盖 `portal_asset_grant / portal_model_asset / portal_model_binding / portal_model_binding_price_version / portal_agent_catalog / ai_sensitive_* / ai_quota_* / job_run_record`，为 P5 和后续 AI 域迁移预留真相源表。
- 本轮补齐了 `/v1/ai/quotas*`、`/v1/ai/sensitive-words*`、`/v1/org/template`、`/v1/org/export`、`/v1/org/import`，优先满足前端现有接线与 Java parity 第一批缺口。
- 当前已补齐 GoFrame `DAO / DO / Entity` 首批基础模型，但 Portal service 还没有完成从原生 SQL 到 DAO 的实用化替换；`AI Quota / AI Sensitive Word` 也仍属于“Portal DB + generic route/runtime projection”的首版实现，不是旧 Java 的完整运行时语义复刻。
- 本轮已补齐 `internal/dao`、`internal/model/do`、`internal/model/entity` 首批共享表模型，并把 `portaldb.autoMigrate` 默认值收紧为关闭；后续仍需继续推进 service 层实用化替换和共库边界治理。
- 本轮进一步把共享表真相源收口到 `aigateway-portal/backend/schema/shared`，`aigateway-console` 只校验共享表存在并仅对 Console 自有表做 `autoMigrate`。
- `Consumer / Org / Invite / Grant / Model Assets / Agent Catalog / AI Quota` 已切到 `portal_user / org_department / org_account_membership / asset_grant / quota_policy_user / portal_model_* / portal_agent_catalog` 这组 Portal 真表名。
- 新增一次性 legacy 数据迁移命令 `backend/main.go portal-legacy-migrate`，用于把 `portal_users / portal_departments / portal_asset_grant / portal_ai_quota_user_policy` 搬迁到真表。
- 已补共享 PostgreSQL testcontainers 集成测试，并新增 Portal 侧兼容校验，验证 Console 写入共享表后 Portal 的用户、模型、Agent、Quota 读取链路可用。
- 本轮完成 `P4-AF-01 ~ P4-AF-03`：
  - `Invite / Asset Grant / AI Quota / AI Sensitive` 的高频写读路径已收口到统一持久化 helper，减少 service 层直接散落的表名和 SQL 片段。
  - 新增 `Consumer / Org / Invite / Model Assets / Agent Catalog / AI Quota / AI Sensitive` HTTP 契约测试与 fixture，补齐 Portal 域回归基线。
  - `Portal Stats` 新增 `usage-trend` 接口，并在 Dashboard 面板补上趋势图和 CSV 导出能力。
- 当前判断：`P4` 已完成并进入维护态，后续仅保留常规缺陷修复与跨阶段联调配合。
