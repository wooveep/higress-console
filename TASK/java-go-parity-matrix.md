# Java -> Go 功能对齐矩阵

## 说明

- 基线：`backend-java-legacy/console/src/main/java/com/alibaba/higress/console/controller/*`
- 当前实现：`aigateway-console/backend`
- 状态说明：
  - `done`：Go 已提供可用接口或新架构等价替代
  - `first-pass`：Go 已提供首版实现，但仍有字段级/运行时差异
  - `pending`：前端或迁移仍有缺口
  - `replaced`：不再按旧接口复刻，由新架构承担

## Controller 对齐

| Java Controller / 功能 | Go 状态 | 阻塞前端/联调 | 验证方式 | 说明 |
| --- | --- | --- | --- | --- |
| `SessionController` | `done` | 否 | `go test ./...` + 登录 smoke | `/session/*` 已迁移 |
| `SystemController` | `done` | 否 | `go test ./...` + `/system/aigateway-config` smoke | 已完成 `aigateway-config` 硬切 |
| `DashboardController` | `first-pass` | 否 | `go test ./...` | 主链路已通，Grafana 代理仍是简化版 |
| `ConsumersController` | `done` | 否 | `go test ./...` | 消费者、密码重置、状态更新已提供 |
| `OrganizationController` | `done` | 否 | `go test ./...` + workbook 接口 | `departments/accounts/template/export/import` 已补齐 |
| `InviteCodeController` | `done` | 否 | `go test ./...` | `/v1/portal/invite-codes*` 已迁移 |
| `AiQuotaController` | `first-pass` | 否 | `go test ./...` + 前端 build | `/v1/ai/quotas*` 已补齐，已切真实 ai-route 读路径，写侧运行时副作用仍待继续收口 |
| `AiSensitiveWordController` | `first-pass` | 否 | `go test ./...` + 前端 build | `/v1/ai/sensitive-words*` 已补齐，当前 reconcile 落 `ai-sensitive-projections` |
| `ModelAssetsController` | `done` | 否 | `go test ./...` | `/v1/ai/model-assets*` 已迁移 |
| `AgentCatalogController` | `done` | 否 | `go test ./...` | `/v1/ai/agent-catalog*` 已迁移 |
| `McpServerController` | `first-pass` | 否 | `go test ./...` | HTTP 契约可用，generic 读链已退出；`save strategy` parity 已完成，剩余主要是更完整的 CRD / 注解字段级语义收口 |
| `WasmPluginsController` | `first-pass` | 否 | `go test ./...` | `/config` 和 `/readme` 已补齐，并支持从 legacy builtin plugin spec/readme 回退加载 |
| `WasmPluginInstancesController` | `first-pass` | 否 | `go test ./...` | 已补 `global/domain/route/service` 和 `delete`，运行时仍非旧 Java CRD 模型 |
| `AiProxyController` | `replaced` | 否 | 文档边界 | 由新 Dashboard/Portal 架构替代 |
| `GrafanaController` | `replaced` | 否 | 文档边界 | 不再恢复旧外嵌代理链路 |
| `PortalStatsController` | `done` | 否 | `go test ./...` + 前端 build | `/v1/portal/stats/*` 已迁移并接入 AI Dashboard |

## 旧 Java TODO 收口

| Java TODO | 当前状态 | 处理结论 |
| --- | --- | --- |
| `McpServiceContextImpl` 路由绑定依赖 `customLabels` | `first-pass` | Go 已为 `mcp-servers/routes` 补显式 `routeMetadata / mcpRouteMetadata`，后续仍需继续收口到更完整的资源关系模型 |
| `DomainServiceImpl` 默认域名旧逻辑 | `done` | Go 已统一到 `aigateway-default-domain`，并完成默认资源命名收口 |
| `KeyAuthCredentialHandler` 插件升级兼容分支 | `pending` | Go 当前仍采用 generic consumer projection，未进入旧插件兼容删除阶段 |
| `WasmPluginServiceImpl` custom plugin config/readme 不支持 | `first-pass` | Go 已补 `/config`、`/readme` 接口，custom schema/readme 仍依赖资源元数据完善 |
| `WasmPluginConfig` validation TODO | `first-pass` | Go 已按 builtin/custom schema 做首批结构校验，后续仍需继续补完整 OpenAPI/JSON Schema 能力 |
| `UpstreamService` validation TODO | `first-pass` | Go 已补 service name/port/weight 与 Route services 基础校验，后续仍可继续细化高级字段 |
| `Route` validation TODO | `first-pass` | Go 已补 name/path/method/services/authConfig 基础校验，并统一默认 `ingressClass` |
| `KubernetesClientService` 异常映射 TODO | `first-pass` | Go 已有 real/fake client 和基础错误映射，但未做到旧 CRD 级别细分 |
| `KubernetesModelConverter` MCP route 判断 TODO | `first-pass` | Go 已引入显式 MCP route metadata，不再只靠 `customLabels`；CRD 级关系仍待后续继续收口 |
| `KubernetesModelConverter` ingressClass TODO | `first-pass` | Go 已补 k8s client / route / mcp 资源的 `ingressClass` 传递与回显，并兼容现有 Helm 环境变量 |
| `McpServiceContextImpl / save strategies` 运行时副作用 | `done` | Go 已完成 `match_list / servers[] / route auth / default-mcp` 的保存/删除副作用对齐，并补齐 Redis / `sse_path_suffix` / 历史兼容路由清理语义 |
| `AiRouteServiceImpl` public/internal/fallback/plugin side effects | `first-pass` | Go 已补 public/internal/fallback route、model-router/model-mapper/ai-statistics/EnvoyFilter 首版联动，仍需继续校准字段级细节与完整 cleanup |
| `LlmProviderServiceImpl` service-source / ACTIVE_PROVIDER_ID / route sync | `first-pass` | Go 已补 `ai-proxy` provider service-scope instance、registry 同步与 route resync 首版，仍需继续对齐 extra service sources 与 provider handler 细节 |

## 本轮补齐结论

- 已解决的前端阻塞接口：
  - `/v1/ai/quotas*`
  - `/v1/ai/sensitive-words*`
  - `/v1/org/template`
  - `/v1/org/export`
  - `/v1/org/import`
  - `/v1/wasm-plugins/:name/readme`
  - `/v1/portal/stats/*`
  - `/internal/jobs*`
- 已收口的脚本链路：
  - `higress/helm/build-local-images.sh --components console` 现在会真实执行 `frontend build -> backend/resource/public/html -> backend/Dockerfile`
  - `aigateway-console/helm/install-console.sh`
  - `aigateway-console/helm/upgrade-aigateway-console-image.sh`
- 仍待继续的差异：
  - K8s CRD/注解字段级对齐
  - MCP / AI Route / Provider 写侧运行时副作用与删除清理
  - DAO / DO / Entity 在 Portal service 中的实用化替换
  - Portal stats 的图表化、导出与更完整 audit 衍生能力
