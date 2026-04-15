# GoFrame 重写开发记录

## 2026-04-14

### 阶段 / 任务

- `P3` aftercare 第二批
- `P3-CP-02` `mcp-servers` save strategy 字段级 parity
- `P3-CP-04` `ai-providers` service source / extra source / ai-route resync parity

### 改动摘要

- `utility/clients/k8s/controlplane_runtime.go` 继续补 `mcp-servers` 与 `ai-providers` 的 Java 字段级对齐，不改上层 `ListResources/GetResource/UpsertResource/DeleteResource` 契约。
- `mcp-servers` 保存/读取补齐 `DIRECT_ROUTE / REDIRECT_ROUTE` 的运行时语义：
  - 回写 `consumerAuthInfo.allowedConsumerLevels` 到 route annotation
  - 从 `directRouteConfig.transportType + higress-config.sse_path_suffix` 计算 rewrite path
  - 读取时恢复 `directRouteConfig.path`
  - 对 DNS registry 自动补 `rewrite.host`
  - `pathRewritePrefix` 统一收口为 Java 语义下的 `/`
  - `SSE` 保存时恢复旧 Java 的结尾校验：`directRouteConfig.path` 必须以 `sse_path_suffix` 结尾；缺失配置时默认回退 `/sse`
  - `higress-config.mcpServer` 初始化补回旧 Java 默认值：`enable=true`、`sse_path_suffix=/sse`、Redis placeholder、空 `servers[]`
  - `match_list / servers[]` 更新改为保留未知字段，避免覆盖手工或运行时附加配置
  - 保存时会主动清理历史兼容路由上的残留 `key-auth` / `default-mcp` 规则
- `ai-providers` 运行时补齐：
  - `openaiExtraCustomUrls` 多 IP 地址聚合为 static registry
  - `vertex-auth.internal` extra service source 注入与“删除时不回收”的共享语义
  - `ai-route` ingress service 引用改为跟随 provider 实际 `dns / static / custom service`
- `gateway` 入口校验与 provider endpoint 规则继续补齐：
  - `OPEN_API.rawConfigurations` 非法 YAML 在入口提前拦截
  - `DATABASE.dbType / dsn` 缺失时拒绝保存
  - `DIRECT_ROUTE / REDIRECT_ROUTE.transportType` 只接受 Java 兼容值，`path` 非法时提前失败
  - `openai / azure` 自定义 URL 改为严格要求带 scheme
  - `qwen / zhipuai` 自定义 endpoint 改为严格域名校验
  - `claude` 补默认 endpoint 推导
- `ai-providers` 保存前继续补齐 Java normalize 语义：
  - `claudeCodeMode` 布尔化，`claudeVersion` 默认补 `2023-06-01`
  - `qwenEnableSearch / qwenEnableCompatible / zhipuCodePlanMode` 默认值对齐
  - `vertexRegion` 小写化，`vertexAuthKey` 结构校验，`vertexAuthServiceName` 默认补齐
  - `bedrock / ollama` 的必填项和端口校验前移到 gateway service
- `mcp-servers` 删除清理继续补边界：
  - `default-mcp` plugin 清理覆盖默认路由和历史兼容路由
  - `DATABASE` 详情补齐 Java 侧兼容的 `rawConfigurations`
- `OPEN_API` 的 Redis 校验错误继续细化，覆盖“缺失配置 / 空地址 / 占位地址”三类失败路径
- 补充 control-plane 纯函数测试，覆盖 provider extra service source 与 MCP direct route restore/rewrite。
- 定向补充 `MCP DIRECT_ROUTE` 的回归测试，覆盖 `SSE path suffix` 非法输入报错，以及运行态未显式配置时默认 `/sse` 回退。
- 定向补充 `MCP ConfigMap` 回归测试，覆盖 `mcpServer` 默认初始化，以及 `match_list / servers[]` 更新时保留未知字段。

### 关键文件 / 接口

- `backend/utility/clients/k8s/controlplane.go`
- `backend/utility/clients/k8s/controlplane_runtime.go`
- `backend/utility/clients/k8s/controlplane_test.go`
- `/v1/mcpServer*`
- `/v1/ai/providers*`
- `/v1/ai/routes*`

### 验证结果

- `go test ./...` 通过。
- 本轮定向验证：`go test ./utility/clients/k8s ./internal/service/gateway` 通过。
- 新增验证覆盖：
  - `openaiExtraCustomUrls` 多 IP static registry/service name 生成
  - `vertex` extra service source 注入
  - `MCP DIRECT_ROUTE` rewrite/path restore 与 `sse_path_suffix` 处理
  - `OPEN_API` 非法 YAML、`DATABASE` 缺失字段、`DIRECT_ROUTE` 非法 transport 的入口校验
  - `DATABASE` 兼容 `rawConfigurations` 生成
  - `openai / azure` scheme 校验与 `qwen / zhipuai` 域名校验
  - `OPEN_API` Redis 缺失/空地址/占位地址错误
  - `DIRECT_ROUTE` 的 `SSE path suffix` 非法输入与默认 `/sse` 回退
  - `mcpServer` 默认初始化与 `match_list / servers[]` 保留未知字段
  - `claude / qwen / vertex` provider 保存前归一化

### 与 backend-java-legacy 的行为差异

- `P3-CP-02` 已完成：`OPEN_API / DATABASE / DIRECT/REDIRECT_ROUTE` 的 save/delete 副作用、Redis 与 `SSE path suffix` 校验、历史兼容路由清理，以及 `match_list / servers[]` 更新语义已与旧 Java 对齐。
- `P3-CP-04` 当前已补齐 extra service source、service ref 选择和首批 provider-specific endpoint 规则，但 provider handler 仍未把所有 Java provider-specific endpoint 细节完整枚举到 Go 侧。
- 本轮未重新执行 live cluster 删除重建 smoke；集群级回归仍以上一轮 `P3-CP-06` 结果为主。

## 2026-04-11

### 阶段 / 任务

- `P0` 契约样本补齐与记录基线建立
- `P1` GoFrame 基座收口
- `P2` 平台核心迁移
- `P3` 网关资源域首版迁移

### 改动摘要

- 新增统一开发记录文件，作为阶段文档之外的 review 汇总真相源。
- `backend/` 新增管理员鉴权中间件、会话管理、用户接口、Dashboard 接口、系统初始化与 `aigateway-config` 读写。
- `utility/clients/{k8s,grafana,portaldb}` 从占位接口补为可配置 client 抽象，默认使用内存/fake 形态支撑迁移与测试。
- `gateway` 域新增内存版资源 CRUD，覆盖 `Route / Domain / TLS / Service / ServiceSource / ProxyServer / Wasm / AI Route / Provider / MCP` 及插件实例接口。
- 补充 `resource/dashboard/*.json` 与 `resource/public/mcp-templates/*`，让 Dashboard 配置下载和 MCP 模板静态访问能在新后端工作。

### 关键文件 / 接口

- `backend/internal/cmd/cmd.go`
- `backend/internal/middleware/auth.go`
- `backend/internal/service/platform/service.go`
- `backend/internal/service/gateway/service.go`
- `backend/internal/controller/{session,user,system,dashboard,gateway}/*`
- `backend/utility/clients/{k8s,grafana,portaldb}/*`
- `/session/*`
- `/user/*`
- `/system/*`
- `/dashboard/*`
- `/v1/routes`
- `/v1/domains`
- `/v1/tls-certificates`
- `/v1/services`
- `/v1/service-sources`
- `/v1/proxy-servers`
- `/v1/wasm-plugins`
- `/v1/ai/*`
- `/v1/mcpServer*`

### 验证结果

- `go test ./...` 通过。
- 手工 smoke 已验证：
  - `GET /system/info`
  - `POST /system/init`
  - `POST /session/login`
  - `GET /user/info`
  - `GET /dashboard/info`
  - `GET /healthz/ready`
  - `GET /system/aigateway-config`
  - `GET /v1/routes`
  - `GET /v1/mcpServer`

### 与 backend-java-legacy 的行为差异

- `P2` 当前管理员配置真相源仍通过内存版 `k8s` fake client 承载，接口行为已对齐到可初始化、可登录、可改密，但未接真实 K8s Secret。
- `P2` Dashboard 目前保留接口、静态配置读取和 native 数据结构；Grafana 代理与 datasource 自动初始化暂未接真实后端。
- `P3` 当前资源域是控制面契约优先的首版实现，CRUD 和 smoke 可用，但仍未完成真实 CRD / ConfigMap / 注解映射的一致性迁移。

### Review 关注点 / 风险

- `gateway` 域当前以 in-memory store 稳定 HTTP 契约，后续接真实 K8s client 时要重点 review 资源字段映射是否与 Java 旧实现一致。
- `platform` 域已形成会话和系统初始化主链，但管理员 Secret、ConfigMap 冲突控制、Dashboard 代理仍需在真实环境 client 接入后复核。
- 当前 `success/data/message` 信封和 HTTP 状态已统一，但“错误码枚举化”仍可在后续迭代继续收口。

## 2026-04-11（第二批）

### 阶段 / 任务

- `P3` 网关资源域收口
- `P4` Portal 业务域第一批启动

### 改动摘要

- `utility/clients/k8s` 新增 `kubectl` 驱动的 real client，实现 `ConfigMap / Secret / generic resource ConfigMap` 持久化，保留 fake client 供测试和契约样本复用。
- 为 `gateway` service 补保护资源、internal 资源、plugin instance 的行为测试，收紧 P3 的基础约束。
- `utility/clients/portaldb` 从占位健康检查升级为真实 SQL client，支持 MySQL driver、schema bootstrap、测试态 `NewFromDB` 注入。
- 新增 `portal` 域 service 与 controller，打通 `Consumer / Org Accounts / Org Departments / Invite Codes` 首批接口。
- `cmd` 装配补齐 `portal` 路由绑定，并扩展 `hack/config.yaml` 的 `k8s / portaldb / grafana` 配置样例。

### 关键文件 / 接口

- `backend/utility/clients/k8s/client.go`
- `backend/utility/clients/portaldb/client.go`
- `backend/internal/service/gateway/service_test.go`
- `backend/internal/service/portal/service.go`
- `backend/internal/service/portal/service_test.go`
- `backend/internal/controller/portal/http.go`
- `/v1/consumers*`
- `/v1/org/departments*`
- `/v1/org/accounts*`
- `/v1/portal/invite-codes*`

### 验证结果

- `go test ./...` 通过。
- 新增验证覆盖：
  - P3：保护资源删除拦截、internal 资源写入拦截、plugin instance round-trip
  - P4：invite code 创建、org account 状态更新的 sqlmock 测试

### 与 backend-java-legacy 的行为差异

- P3 real client 当前把通用资源落为带标签的 `ConfigMap`，还没有直接映射到旧 Java 使用的 CRD/注解组合。
- P4 第一批采用手写 SQL service + schema bootstrap 方式先打通真相源，没有生成 GoFrame `DAO / DO / Entity` 产物。
- P4 目前只覆盖 `Consumer / Org / Invite Code`，`Asset Grant / Model Assets / Pricing / Stats / AI Quota / AI Sensitive` 仍未迁移。
- `org/template`、`org/export`、`org/import` 仍未实现，本轮明确延后。

### Review 关注点 / 风险

- `kubectl` real client 依赖运行环境中的 `kubectl` 与 kubeconfig/namespace 配置，部署环境需要额外验证二进制和权限。
- P3 资源目前是“真实 K8s 持久化 + 泛化资源模型”，不是“真实 CRD 字段级全量对齐”，review 时要特别关注这个边界。
- P4 首批 DB 接入优先保证接口闭环和可测性，后续补 `DAO / DO / Entity` 时要避免与当前 SQL schema bootstrap 产生重复定义。

## 2026-04-11（第三批）

### 阶段 / 任务

- `P4` Model Assets / Agent Catalog / Asset Grants
- `P5` Jobs / Reconcile

### 改动摘要

- `portaldb` schema bootstrap 继续扩展，补上 `portal_asset_grant / portal_model_asset / portal_model_binding / portal_model_binding_price_version / portal_agent_catalog / ai_sensitive_* / job_run_record`。
- `portal` 域新增 `Asset Grants`、`Model Assets`、`Model Bindings`、`Binding Price Versions`、`Agent Catalog` 服务与 HTTP 路由，补齐前端依赖的 `/v1/assets/*/grants`、`/v1/ai/model-assets*`、`/v1/ai/agent-catalog*`。
- `portal` 写路径统一接入 hook，给后续任务体系提供单一入口。
- 新增 `internal/service/jobs` 与 `internal/controller/jobs`，落地统一 job registry、cron 注册、手工触发、最近执行结果查询、幂等跳过与 `job_run_record` 落库。
- 首批迁移四类任务：
  - `portal-consumer-projection`
  - `portal-consumer-level-auth-reconcile`
  - `ai-sensitive-projection`
  - `ai-plugin-execution-order-reconcile`

### 关键文件 / 接口

- `backend/internal/service/portal/assets.go`
- `backend/internal/controller/portal/http.go`
- `backend/internal/service/jobs/service.go`
- `backend/internal/controller/jobs/http.go`
- `backend/internal/job/jobs.go`
- `backend/internal/cmd/cmd.go`
- `/v1/assets/{assetType}/{assetId}/grants`
- `/v1/ai/model-assets*`
- `/v1/ai/agent-catalog*`
- `/internal/jobs`
- `/internal/jobs/{name}`
- `/internal/jobs/{name}/trigger`

### 验证结果

- `go test ./...` 通过。
- 新增验证覆盖：
  - P4：`ReplaceAssetGrants` sqlmock 测试、`GetAgentCatalogOptions` K8s options 测试
  - P5：`portal-consumer-projection` 手工触发测试、同快照幂等跳过测试

### 与 backend-java-legacy 的行为差异

- P4 当前仍使用手写 SQL service，没有生成 GoFrame `DAO / DO / Entity` 产物。
- `portal-consumer-projection` 当前把 Portal 用户投影到 generic `consumers` 资源，而不是旧 Java 中的 key-auth consumer/credential 真实控制面对象。
- `ai-sensitive-projection` 当前先把 Portal DB 规则聚合为 `ai-sensitive-projections/default` 资源，尚未接入旧 Java 的 `ai-data-masking` plugin instance 配置投影细节。
- `ai-plugin-execution-order-reconcile` 当前只对 generic `wasm-plugins` 资源的 `phase/priority` 做对齐，未迁移旧 Java 的 built-in WasmPlugin CR 版本迁移与 legacy CR 合并逻辑。
- `P6` 未启动，本轮明确保持旧命名与现有前端调用路径不变。

### Review 关注点 / 风险

- `P4` 的 HTTP 契约已补齐，但 Portal 领域模型仍处于“service + schema bootstrap”阶段，后续补 `DAO / DO / Entity` 时要避免出现第二套字段命名。
- `P5` 的运行记录与幂等已经落库，但“基础指标”目前主要体现为最近执行状态和运行耗时，还没有外接 Prometheus/Grafana 指标面。
- `consumer projection` 与 `AI sensitive projection` 都是“迁移路径优先”的首版实现，真实运行时目标资源若切换为更贴近旧 Java 的对象模型，需要再次 review 任务输出契约。

## 2026-04-11（第四批）

### 阶段 / 任务

- `P6` 切换与命名收口

### 改动摘要

- 后端把 `/system/higress-config` 硬切为 `/system/aigateway-config`，并把 ConfigMap 业务键从 `higress` 改为 `aigateway`。
- `platform` 服务新增一次性迁移逻辑，读取到旧 `higress` 键时会自动改写为 `aigateway` 并回写到 `ConfigMap`。
- 前端同步切换 `system` service、系统页标题、语言存储键、默认域名、AI Route 示例文案和页面标题到 `aigateway` 命名。
- `backend/go.mod` 与全部 Go import path 已切换到 `github.com/wooveep/aigateway-console/backend`。
- 当前项目目录已从 `higress-console` 改名为 `aigateway-console`，并同步更新父仓库活跃脚本、Helm 依赖路径和根级协作文档引用。
- Helm chart、安装脚本、镜像默认名、MCP 模板值和公开 README/CONTRIBUTING 文档已收口到 `aigateway-console`。

### 关键文件 / 接口

- `backend/internal/controller/system/http.go`
- `backend/internal/service/platform/service.go`
- `backend/internal/service/platform/service_test.go`
- `backend/utility/clients/k8s/client.go`
- `frontend/src/services/system.ts`
- `frontend/src/i18n.ts`
- `frontend/src/views/system/SystemPage.vue`
- `frontend/src/views/gateway/DomainPage.vue`
- `frontend/src/views/ai/AiRoutePage.vue`
- `backend/go.mod`
- `helm/Chart.yaml`
- `helm/install-console.sh`
- `../scripts/aigateway-dev.py`
- `../higress/helm/higress/{Chart.yaml,Chart.lock}`
- `GET /system/aigateway-config`
- `PUT /system/aigateway-config`

### 验证结果

- `go test ./...` 通过。
- `npm run build` 通过。
- `helm template aigateway-console ./helm` 通过。
- `helm dependency update && helm template aigateway ./`（`higress/helm/higress`）通过。
- 手工 smoke 已验证：
  - `GET /system/info`
  - `GET /healthz/ready`
  - `GET /landing`
  - `POST /system/init`
  - `POST /session/login`
  - `GET /system/aigateway-config`（登录后）

### 与 backend-java-legacy 的行为差异

- 本轮按硬切策略执行，`/system/higress-config` 旧接口不再保留兼容别名。
- 运行时兼容仅保留一次性迁移：旧 `ConfigMap.data.higress` 会自动翻转为 `ConfigMap.data.aigateway`，不做长期双读双写。
- 历史归档名称未整体重写；`backend-java-legacy/`、根目录历史记忆文件与 `higress/` 子项目中的历史语义仍保持原样。

### Review 关注点 / 风险

- Helm 默认镜像仓库已切到 `.../aigateway-console`，实际发布环境需要确认对应镜像标签存在，避免部署时拉取失败。
- 父仓库目前只收口了活跃脚本和协作文档；其他历史材料中仍可能保留旧名称，这是刻意保留而非遗漏。
- `P6` 当前已完成工程内硬切，但若后续还要同步外部仓库名、发布流水线或文档站链接，需要在仓库外继续做配套迁移。

## 2026-04-12

### 阶段 / 任务

- 脚本链路刷新
- Java parity 第一批缺口补齐
- legacy TODO 收口台账建立

### 改动摘要

- `higress/helm/build-local-images.sh` 的 `console` 构建链路改为真实执行 `frontend build -> backend/resource/public/html -> backend/Dockerfile`，不再依赖旧 Java 静态目录，也不再把 `backend/build.sh` 误当成镜像构建入口。
- `aigateway-console/backend/Dockerfile` 改为支持 `TARGETARCH` 多架构构建，并以当前 Go backend 为最终运行镜像入口。
- `aigateway-console/helm/install-console.sh` 与 `upgrade-aigateway-console-image.sh` 的默认镜像名从临时 demo tag 收口到 `aigateway/console:*`。
- Portal 域补齐前端已依赖的首批缺口接口：
  - `/v1/ai/quotas*`
  - `/v1/ai/sensitive-words*`
  - `/v1/org/template`
  - `/v1/org/export`
  - `/v1/org/import`
- Gateway 域补齐 Java parity 第二批中的高频缺口：
  - `/v1/wasm-plugins/:name/readme`
  - `global/domain/route/service` plugin instance delete / service-scope 操作
- 新增 `TASK/java-go-parity-matrix.md`，把 Java controller 对齐状态与 legacy TODO 收口状态统一沉淀成台账。

### 关键文件 / 接口

- `higress/helm/build-local-images.sh`
- `aigateway-console/backend/Dockerfile`
- `aigateway-console/helm/install-console.sh`
- `aigateway-console/helm/upgrade-aigateway-console-image.sh`

## 2026-04-12（第二批）

### 阶段 / 任务

- `P4` Portal 真表收口
- Console / Portal 共享 schema 集成校验

### 改动摘要

- 在 `aigateway-portal/backend/schema/shared` 新增共享 schema 包，把 `portal_user / portal_invite_code / org_department / org_account_membership / asset_grant / quota_policy_user / portal_model_asset / portal_model_binding / portal_agent_catalog` 收口成 Portal 侧唯一 migration 真相源。
- `aigateway-portal` 自己改为复用共享 schema 包启动；`aigateway-console` 不再创建这些共享表，只在启动时校验它们存在。
- `portaldb.autoMigrate` 语义调整为“仅对 Console 自有表生效”，Console 自有表保留 `portal_model_binding_price_version / portal_ai_sensitive_* / portal_ai_quota_balance / portal_ai_quota_schedule_rule / job_run_record`。
- `portaldb` 新增 `MigrateLegacyData`，支持把 `portal_users / portal_departments / portal_asset_grant / portal_ai_quota_user_policy` 一次性幂等迁移到 Portal 真表。
- `portal` service 把 `Consumer / Org / Invite / Grants / Model Assets / Model Bindings / Agent Catalog / AI Quota` 主链切到 Portal 真表名，并移除稳态读接口里的 `temp_password` 暴露。
- 新增 `portal-legacy-migrate` 命令，供运维显式执行旧表退场迁移。
- 新增 testcontainers 集成测试：
  - Console 侧验证共享 schema + Console 自有表 + legacy 迁移
  - Portal 侧验证对 Console 写入共享表后的用户、模型、Agent、Quota 读取兼容性

### 关键文件 / 接口

- `aigateway-portal/backend/schema/shared/shared.go`
- `aigateway-portal/backend/internal/service/portal/{service.go,organization.go}`
- `aigateway-console/backend/utility/clients/portaldb/client.go`
- `aigateway-console/backend/internal/cmd/portal_legacy_migrate.go`
- `aigateway-console/backend/internal/service/portal/{service.go,assets.go,ai_quota.go}`
- `aigateway-console/backend/internal/dao/schema.go`
- `aigateway-console/backend/internal/model/{do,entity}/schema_models.go`
- `aigateway-console/backend/utility/clients/portaldb/client_integration_test.go`
- `aigateway-console/backend/internal/service/portal/shared_schema_integration_test.go`
- `aigateway-portal/backend/internal/service/portal/shared_console_compat_integration_test.go`
- `portal-legacy-migrate`

### 验证结果

- `cd aigateway-console/backend && go test ./...` 通过。
- `cd aigateway-portal/backend && go test ./...` 通过。
- 新增 MySQL testcontainers 集成验证通过：
  - 共享 schema 初始化
  - Console 自有表初始化
  - legacy 表到真表迁移
  - Console `consumer/org/invite/model/agent/quota/ai-sensitive/jobs` 主链
  - Portal 对 Console 写入共享表后的读取兼容

### 与 backend-java-legacy 的行为差异

- 共享表命名与 Portal 当前真表对齐后，Console 不再维持旧 `portal_users / portal_departments / portal_asset_grant / portal_ai_quota_user_policy` 运行时兼容层；这些旧表只保留给一次性迁移工具读取。
- `DAO / DO / Entity` 已补齐共享表和 Console 自有表的首版模型，但 `portal service` 还未完全消除原生 SQL，当前属于“真表名/真 schema 已切换，ORM 化正在收口”的中间态。
- `portal_model_binding_price_version`、`portal_ai_sensitive_*`、`portal_ai_quota_balance`、`portal_ai_quota_schedule_rule`、`job_run_record` 仍由 Console 自己持有，Portal 侧不直接消费。

### Review 关注点 / 风险

- `portaldb` 现在默认不会帮共享表兜底建表，部署前必须确保 `aigateway-portal` migration 已先执行。
- 共享表真相源已统一，但 `portal service` 仍保留部分原生 SQL 访问；后续继续向 GoFrame DAO/DO 收口时，要避免在同一字段上出现双套映射。
- legacy 迁移命令是硬切方案的一部分，正式切换时需要在运维步骤里明确执行顺序：`Portal migration -> Console owned schema -> legacy migrate -> smoke`。
- `aigateway-console/backend/internal/controller/portal/http.go`
- `aigateway-console/backend/internal/service/portal/{ai_quota,ai_sensitive,org_workbook}.go`
- `aigateway-console/backend/internal/controller/gateway/http.go`
- `aigateway-console/backend/internal/service/gateway/service.go`
- `aigateway-console/TASK/java-go-parity-matrix.md`

### 验证结果

- `cd aigateway-console/backend && go test ./...` 通过。
- `cd aigateway-console/frontend && npm run build` 通过。
- `python3 ./scripts/aigateway-dev.py show` 通过。
- `python3 ./scripts/aigateway-dev.py sync --check` 通过。
- `bash higress/helm/build-local-images.sh --dry-run --components console` 通过。
- `helm template aigateway-console ./aigateway-console/helm` 通过。
- `helm template aigateway ./higress/helm/higress` 通过。

### 与 backend-java-legacy 的行为差异

- `AI Quota` 当前采用 `Portal DB + generic ai-routes` 的首版实现，优先满足前端菜单和管理页面，不等同于旧 Java 的完整运行时计费/Redis 语义。
- `AI Sensitive` 当前 reconcile 到 `ai-sensitive-projections/default` generic 资源，还不是旧 Java 的完整 `ai-data-masking` plugin instance 投影。
- `org/template/export/import` 已恢复为真实 XLSX 接口，但当前模板字段仍以 Go 首版组织模型为主，没有完全对齐旧 Java 的扩展列。
- `Wasm readme/config` 已补接口，但 custom plugin schema/readme 仍取决于资源元数据是否完善。
- MCP 显式路由关联元数据、ingressClass、Route/Service/Wasm 深校验等旧 Java TODO 仍未完全收口，已在 parity 台账单独列出。

### Review 关注点 / 风险

- `build-local-images.sh` 现在会在构建 `console` 时执行 `npm install`，CI/开发机要保证 Node 依赖拉取可用；后续可以再按 lockfile/缓存策略继续优化。
- 新增 `AI Quota / AI Sensitive` API 目前重点是“前端可用 + review 可跟踪”，不是与旧 Java 运行时完全等价的最终形态。
- `Portal DB` 仍然没有收口到 GoFrame `DAO / DO / Entity`，当前新增表和 SQL 逻辑后续仍需继续治理。

## 2026-04-12

### 阶段 / 任务

- P3/P4 aftercare：`higress / portal / plugin-server` 对接面收口第一批

### 改动摘要

- `k8s` client 增加 `ingressClass` 配置与回显能力，`cmd` 同时兼容现有 Helm/Higress 环境变量。
- `gateway` service 为 `Route / Service / MCP / Wasm` 增加首批深校验，并补显式 `routeMetadata / mcpRouteMetadata`，不再只依赖旧 `customLabels` 思路。
- `wasm-plugins` 列表、`/config`、`/readme` 增加 legacy builtin plugin spec/readme 回退链路，前端不再只能依赖运行时资源里是否刚好带了 schema/readme。
- `internal/dao`、`internal/model/do`、`internal/model/entity` 新增首批 Portal/Job 共享表基础模型，结束空目录状态。
- `portaldb.autoMigrate` 默认值改为关闭，降低 Console 继续主动扩散共享库 schema 的风险。

### 关键文件 / 接口

- `aigateway-console/backend/internal/cmd/cmd.go`
- `aigateway-console/backend/utility/clients/k8s/client.go`
- `aigateway-console/backend/internal/service/gateway/{service,validation,plugin_metadata}.go`
- `aigateway-console/backend/internal/{dao,model/do,model/entity}/*`
- `aigateway-console/TASK/java-go-parity-matrix.md`

### 验证结果

- `cd aigateway-console/backend && go test ./...` 通过。
- `python3 ./scripts/aigateway-dev.py show` 通过。
- `helm template aigateway-console ./aigateway-console/helm` 通过。

### 与 backend-java-legacy 的行为差异

- `MCP route metadata` 已从“只靠标签推断”推进到显式元数据，但还不是旧 Java/K8s CRD 的完整关系模型。
- `Wasm` 已可从 builtin plugin 资源回退加载 schema/readme，但 custom plugin 元数据仍主要依赖运行时资源本身。
- `DAO / DO / Entity` 目前完成的是基础模型落盘，不是 Portal service 全量替换 raw SQL 的终态。

### Review 关注点 / 风险

- 当前 `portal` service 仍主要使用原生 SQL；虽然表模型已补齐，但真正的 service/DAO 收口还需要后续分批推进。
- `plugin instance` 的 schema 校验现在是首批结构校验，已经能兜底明显错误，但还不是完整 JSON Schema 实现。

## 2026-04-14

### 阶段 / 任务

- `P4 aftercare` 收口：`P4-AF-01 ~ P4-AF-03`

### 改动摘要

- `portal` 域新增统一持久化 helper，把 `Invite / Asset Grant / AI Quota / AI Sensitive` 的高频写读路径从散落 raw SQL 收到 `DAO / DO / Entity` 风格入口。
- 新增 `Consumer / Org / Invite / Model Assets / Agent Catalog / AI Quota / AI Sensitive` 的 HTTP 契约测试与 fixture，形成 Portal 域回归基线。
- `Portal Stats` 新增 `/v1/portal/stats/usage-trend`，Dashboard 面板补上趋势图、汇总卡片和 CSV 导出。
- `TASK/README.md` 与 `P4-portal-domain.md` 已同步更新，`P4` 状态切为 `done`。

### 关键文件 / 接口

- `backend/internal/service/portal/{store.go,service.go,assets.go,ai_quota.go,ai_sensitive.go,stats.go,contracts_test.go}`
- `backend/internal/controller/portal/http.go`
- `backend/test/contracts/{consumers,org,portal-invite,model-assets,agent-catalog,ai-quota,ai-sensitive}`
- `frontend/src/features/dashboard/{PortalStatsPanel.vue,portal-stats/PortalStatsTrendChart.vue}`
- `frontend/src/{interfaces/services}/portal-stats.ts`
- `/v1/portal/stats/usage-trend`

### 验证结果

- `cd aigateway-console/backend && go test ./internal/service/portal ./internal/service/platform` 通过。
- `cd aigateway-console/frontend && npm run build` 通过。

### 阶段 / 任务

- `P3 / P4 / P5` 功能闭环补齐

### 改动摘要

- `Plugin Management` 补齐 `README` 展示、`global/route/domain/service` 绑定删除，以及 `service-scope plugin instances` 整组前端接线。
- `Service List` 新增跳转到 service-scope 插件配置页，`PluginPage` 同步支持 `service` 作用域。
- `Consumer Management` 新增只读详情抽屉，并接上 `GET /v1/consumers/:name`；后端 `GetConsumer` 同步补返回 `departmentId/path`、`createdAt`、`updatedAt`、`lastLoginAt` 和脱敏后的活跃凭证概要。
- `portal` 域新增 `/v1/portal/stats/{usage,usage-events,department-bills}`，并在 `AI Dashboard` 增加 Portal Stats 页签区。
- `System Settings` 新增 `/system/jobs` 运维页，支持任务列表、详情、最近运行记录和手动触发。

### 关键文件 / 接口

- `backend/internal/service/portal/{service.go,stats.go,stats_test.go}`
- `backend/internal/controller/portal/http.go`
- `frontend/src/views/plugin/PluginPage.vue`
- `frontend/src/views/consumer/ConsumerPage.vue`
- `frontend/src/features/dashboard/PortalStatsPanel.vue`
- `frontend/src/views/system/SystemJobsPage.vue`
- `/v1/portal/stats/usage`
- `/v1/portal/stats/usage-events`
- `/v1/portal/stats/department-bills`
- `/internal/jobs`
- `/internal/jobs/{name}`
- `/internal/jobs/{name}/trigger`

### 验证结果

- `cd aigateway-console/backend && go test ./...` 通过。
- `cd aigateway-console/frontend && npm run build` 通过。

### 与 backend-java-legacy 的行为差异

- `Portal Stats` 当前已完成查询接口迁移，但前端仍以表格/筛选为主，尚未恢复旧系统没有的图表化与导出体验。
- `Consumer detail` 当前优先复用 Portal DB 与现有接口，只展示可稳定返回的账户与脱敏凭证信息，不扩展新的写操作。
- `Jobs` 当前以系统页下的运维入口承载，没有提升为独立一级导航。

## 2026-04-14

### 阶段 / 任务

- `P3 aftercare` 第二批：`MCP / AI Route / Provider` 控制面真相源与写侧副作用收口

### 改动摘要

- `routes / ai-routes / ai-providers` 已从 `console.aigateway.io/type=resource` generic 资源切回真实 `Ingress / ConfigMap / WasmPlugin` 读链。
- `mcp-servers` 已纳入 real client 的 control-plane 分支，开始按 `Ingress + higress-config + McpBridge/registry + route auth/plugin instance` 联合模型读写。
- `ai-routes` 首批补上 public/internal/fallback route、`model-router`、`model-mapper`、`ai-statistics`、`EnvoyFilter` 的联动写路径。
- `ai-providers` 首批补上 `ai-proxy.internal` 之外的 registry / service-scope `ACTIVE_PROVIDER_ID` 同步与相关 `ai-routes` resync。
- live cluster smoke 期间修复两处真实问题：
  - `mutateBuiltinWasmPlugin` 更新现有 `WasmPlugin` 时不能盲删 `metadata.resourceVersion`
  - `MCP` 运行时读写 `higress-config` 时不能误指向 `aigateway-console` ConfigMap
- TASK 台账同步拆出 `P3-CP-01 ~ P3-CP-06`，后续继续按控制面真相源和 Java save strategy parity 追踪。

### 关键文件 / 接口

- `aigateway-console/backend/utility/clients/k8s/controlplane.go`
- `aigateway-console/backend/utility/clients/k8s/controlplane_runtime.go`
- `aigateway-console/backend/utility/clients/k8s/controlplane_test.go`
- `aigateway-console/TASK/{README.md,P3-gateway-resource-domain.md,java-go-parity-matrix.md,development-record.md}`

### 验证结果

- `cd aigateway-console/backend && go test ./...` 通过。
- `minikube / aigateway-system` real cluster smoke 通过：
  - 临时 `ai-provider` create/update/delete/cleanup 通过
  - 临时 `ai-route` create/update/delete/cleanup 通过
  - 临时 `mcp-server` create/update/delete/cleanup 通过

### 与 backend-java-legacy 的行为差异

- `routes / ai-routes / ai-providers` 真实读链已打通，但 `MCP` 和 `AI` 写路径仍未与 Java 运行时副作用完全对齐。
- `MCP` 当前已覆盖 route、`higress-config`、route auth、`default-mcp` plugin 的首批联动；仍需继续补历史兼容路由、更多 save strategy 细节与删除清理。
- `AI Route / Provider` 当前已覆盖 runtime 联动主链，但仍需继续对齐 provider handler 细节、extra service sources、fallback cleanup 与更多字段级语义。

### Review 关注点 / 风险

- 这一轮仍处于 aftercare 中段，重点是“真相源回归 + 关键副作用打通”，还不是 Java parity 的最终收口版本。
- real cluster smoke 和 delete cleanup 仍需继续验证，尤其要关注 `matchRules`、`EnvoyFilter`、service-scope plugin instance 是否有残留。
