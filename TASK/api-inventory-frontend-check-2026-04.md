# Console API Inventory For Frontend Check

更新时间：`2026-04-14`

这份清单用于在 `console` 后端 GoFrame 重构后，给前端逐项核对当前 API 契约。

判定口径：

- 后端来源：`backend/internal/cmd/cmd.go` 与各 `internal/controller/**/http.go`
- 前端来源：`frontend/src/services/*.ts`
- 本文只列当前 `console` 进程实际暴露的 HTTP 接口，不展开内部 service 细节

## 1. 公共入口与匿名访问资源

这些路径当前不要求登录态，可用于页面加载、系统初始化与健康检查。

| Method | Path | 用途 | 前端现状 |
| --- | --- | --- | --- |
| `GET` | `/` | SPA 入口页 | 浏览器直接访问 |
| `GET` | `/landing` | SPA 入口别名 | 历史兼容 |
| `GET` | `/assets/*` | 前端静态资源 | 浏览器直接加载 |
| `GET` | `/mcp-templates/index.json` | MCP 预置模板索引 | `frontend/src/services/mcp.ts` 已调用 |
| `GET` | `/mcp-templates/:id.yaml` | MCP 预置模板内容 | `frontend/src/services/mcp.ts` 已调用 |
| `GET` | `/healthz` | 基础健康检查 | 探针/排障 |
| `GET` | `/healthz/ready` | readiness 检查 | K8s readinessProbe |
| `GET` | `/swagger/*` | Swagger UI | 调试用 |
| `GET` | `/api.json` | OpenAPI 文档 | 调试用 |

## 2. 会话、用户、系统、Dashboard

### 2.1 Session / User

| Method | Path | 前端 service | 说明 |
| --- | --- | --- | --- |
| `POST` | `/session/login` | `user.ts` | 登录 |
| `GET` | `/session/logout` | `user.ts` | 登出 |
| `GET` | `/user/info` | `user.ts` | 当前用户信息 |
| `POST` | `/user/changePassword` | `user.ts` | 修改密码 |

### 2.2 System

| Method | Path | 前端 service | 说明 |
| --- | --- | --- | --- |
| `GET` | `/system/info` | `system.ts` | 后端系统信息 |
| `GET` | `/system/config` | `system.ts` | 迁移阶段配置摘要 |
| `POST` | `/system/init` | `system.ts` | 初始化系统 |
| `GET` | `/system/aigateway-config` | `system.ts` | 获取 AIGateway 配置 |
| `PUT` | `/system/aigateway-config` | `system.ts` | 更新 AIGateway 配置 |

### 2.3 Dashboard

| Method | Path | 前端 service | 说明 |
| --- | --- | --- | --- |
| `GET` | `/dashboard/init` | `dashboard.ts` | 初始化 dashboard 配置 |
| `GET` | `/dashboard/info` | `dashboard.ts` | 获取 dashboard 信息 |
| `PUT` | `/dashboard/info` | `dashboard.ts` | 更新 dashboard URL |
| `GET` | `/dashboard/configData` | `dashboard.ts` | 获取 dashboard 配置数据 |
| `GET` | `/dashboard/native` | `dashboard.ts` | 获取原生 dashboard 数据 |

## 3. Gateway 域 API

### 3.1 核心资源集合

| 资源 | Method | Path | 前端 service | 备注 |
| --- | --- | --- | --- | --- |
| Routes | `GET` | `/v1/routes` | `route.ts`, `route-compat.ts` | 后端返回分页包装 |
| Routes | `GET` | `/v1/routes/:name` | `route.ts`, `route-compat.ts` | 已对齐 |
| Routes | `POST` | `/v1/routes` | `route.ts`, `route-compat.ts` | 已对齐 |
| Routes | `PUT` | `/v1/routes/:name` | `route.ts`, `route-compat.ts` | 已对齐 |
| Routes | `DELETE` | `/v1/routes/:name` | `route.ts`, `route-compat.ts` | 已对齐 |
| Domains | `GET` | `/v1/domains` | `domain.ts` | 已对齐 |
| Domains | `GET` | `/v1/domains/:name` | 前端未直接调用 | 后端已提供 |
| Domains | `POST` | `/v1/domains` | `domain.ts` | 已对齐 |
| Domains | `PUT` | `/v1/domains/:name` | `domain.ts` | 已对齐 |
| Domains | `DELETE` | `/v1/domains/:name` | `domain.ts` | 已对齐 |
| TLS Certificates | `GET` | `/v1/tls-certificates` | `tls-certificate.ts` | 已对齐 |
| TLS Certificates | `POST` | `/v1/tls-certificates` | `tls-certificate.ts` | 已对齐 |
| TLS Certificates | `PUT` | `/v1/tls-certificates/:name` | `tls-certificate.ts` | 已对齐 |
| TLS Certificates | `DELETE` | `/v1/tls-certificates/:name` | `tls-certificate.ts` | 已对齐 |
| Service Sources | `GET` | `/v1/service-sources` | `service-source.ts` | 已对齐 |
| Service Sources | `POST` | `/v1/service-sources` | `service-source.ts` | 已对齐 |
| Service Sources | `PUT` | `/v1/service-sources/:name` | `service-source.ts` | 已对齐 |
| Service Sources | `DELETE` | `/v1/service-sources/:name` | `service-source.ts` | 已对齐 |
| Proxy Servers | `GET` | `/v1/proxy-servers` | `proxy-server.ts` | 已对齐 |
| Proxy Servers | `POST` | `/v1/proxy-servers` | `proxy-server.ts` | 已对齐 |
| Proxy Servers | `PUT` | `/v1/proxy-servers/:name` | `proxy-server.ts` | 已对齐 |
| Proxy Servers | `DELETE` | `/v1/proxy-servers/:name` | `proxy-server.ts` | 已对齐 |
| Services | `GET` | `/v1/services` | `service.ts` | 只读列表 |

### 3.2 AI Routes / Providers / Wasm Plugins / MCP

| 资源 | Method | Path | 前端 service | 备注 |
| --- | --- | --- | --- | --- |
| AI Routes | `GET` | `/v1/ai/routes` | `ai-route.ts` | 后端返回分页包装 |
| AI Routes | `GET` | `/v1/ai/routes/:name` | `ai-route.ts` | 已对齐 |
| AI Routes | `POST` | `/v1/ai/routes` | `ai-route.ts` | 已对齐 |
| AI Routes | `PUT` | `/v1/ai/routes/:name` | `ai-route.ts` | 已对齐 |
| AI Routes | `DELETE` | `/v1/ai/routes/:name` | `ai-route.ts` | 已对齐 |
| AI Providers | `GET` | `/v1/ai/providers` | `llm-provider.ts` | 已对齐 |
| AI Providers | `POST` | `/v1/ai/providers` | `llm-provider.ts` | 已对齐 |
| AI Providers | `PUT` | `/v1/ai/providers/:name` | `llm-provider.ts` | 已对齐 |
| AI Providers | `DELETE` | `/v1/ai/providers/:name` | `llm-provider.ts` | 已对齐 |
| MCP Servers | `GET` | `/v1/mcpServer` | `mcp.ts` | 后端返回分页包装 |
| MCP Servers | `GET` | `/v1/mcpServer/:name` | `mcp.ts` | 已对齐 |
| MCP Servers | `POST` | `/v1/mcpServer` | `mcp.ts` | 已对齐 |
| MCP Servers | `PUT` | `/v1/mcpServer/:name` | `mcp.ts` | 前端已按当前 RESTful 契约适配 |
| MCP Servers | `DELETE` | `/v1/mcpServer/:name` | `mcp.ts` | 已对齐 |
| MCP Servers | `GET` | `/v1/mcpServer/consumers` | `mcp.ts` | 依赖 query `mcpServerName` |
| MCP Servers | `POST` | `/v1/mcpServer/swaggerToMcpConfig` | `mcp.ts` | 已对齐 |
| Wasm Plugins | `GET` | `/v1/wasm-plugins` | `plugin.ts` | 已对齐 |
| Wasm Plugins | `POST` | `/v1/wasm-plugins` | `plugin.ts` | 已对齐 |
| Wasm Plugins | `PUT` | `/v1/wasm-plugins/:name` | `plugin.ts` | 已对齐 |
| Wasm Plugins | `DELETE` | `/v1/wasm-plugins/:name` | `plugin.ts` | 已对齐 |
| Wasm Plugins | `GET` | `/v1/wasm-plugins/:name/config` | `plugin.ts` | 已对齐 |
| Wasm Plugins | `GET` | `/v1/wasm-plugins/:name/readme` | 前端当前未接 | 后端已提供，builtin metadata 优先读 `backend/resource/public/plugin` 快照 |

### 3.3 Plugin Instance 配置

| Scope | Method | Path | 前端 service | 备注 |
| --- | --- | --- | --- | --- |
| Global | `GET` | `/v1/global/plugin-instances/:pluginName` | `plugin.ts` | 已对齐 |
| Global | `PUT` | `/v1/global/plugin-instances/:pluginName` | `plugin.ts` | 已对齐 |
| Global | `DELETE` | `/v1/global/plugin-instances/:pluginName` | 前端当前未接 | 后端已提供 |
| Route | `GET` | `/v1/routes/:name/plugin-instances` | `plugin.ts` | 已对齐 |
| Route | `GET` | `/v1/routes/:name/plugin-instances/:pluginName` | `plugin.ts` | 已对齐 |
| Route | `PUT` | `/v1/routes/:name/plugin-instances/:pluginName` | `plugin.ts` | 已对齐 |
| Route | `DELETE` | `/v1/routes/:name/plugin-instances/:pluginName` | 前端当前未接 | 后端已提供 |
| Domain | `GET` | `/v1/domains/:name/plugin-instances` | `plugin.ts` | 已对齐 |
| Domain | `GET` | `/v1/domains/:name/plugin-instances/:pluginName` | `plugin.ts` | 已对齐 |
| Domain | `PUT` | `/v1/domains/:name/plugin-instances/:pluginName` | `plugin.ts` | 已对齐 |
| Domain | `DELETE` | `/v1/domains/:name/plugin-instances/:pluginName` | 前端当前未接 | 后端已提供 |
| Service | `GET` | `/v1/services/:name/plugin-instances` | 前端当前未接 | 后端已提供 |
| Service | `GET` | `/v1/services/:name/plugin-instances/:pluginName` | 前端当前未接 | 后端已提供 |
| Service | `PUT` | `/v1/services/:name/plugin-instances/:pluginName` | 前端当前未接 | 后端已提供 |
| Service | `DELETE` | `/v1/services/:name/plugin-instances/:pluginName` | 前端当前未接 | 后端已提供 |

## 4. Portal 域 API

### 4.1 Consumer / Invite / Asset Grants

| 资源 | Method | Path | 前端 service | 备注 |
| --- | --- | --- | --- | --- |
| Consumers | `GET` | `/v1/consumers` | `consumer.ts` | 已对齐 |
| Consumers | `GET` | `/v1/consumers/:name` | 前端当前未接 | 后端已提供 |
| Consumers | `POST` | `/v1/consumers` | `consumer.ts` | 已对齐 |
| Consumers | `PUT` | `/v1/consumers/:name` | `consumer.ts` | 已对齐 |
| Consumers | `DELETE` | `/v1/consumers/:name` | `consumer.ts` | 已对齐 |
| Consumer Departments | `GET` | `/v1/consumers/departments` | `consumer.ts` | 已对齐 |
| Consumer Departments | `POST` | `/v1/consumers/departments` | `consumer.ts` | 已对齐 |
| Consumer Status | `PATCH` | `/v1/consumers/:name/status` | `consumer.ts` | 已对齐 |
| Consumer Password | `POST` | `/v1/consumers/:name/password/reset` | `consumer.ts` | 已对齐 |
| Asset Grants | `GET` | `/v1/assets/:assetType/:assetId/grants` | `organization.ts` | 已对齐 |
| Asset Grants | `PUT` | `/v1/assets/:assetType/:assetId/grants` | `organization.ts` | 已对齐 |
| Invite Codes | `POST` | `/v1/portal/invite-codes` | `consumer.ts` | 已对齐 |
| Invite Codes | `GET` | `/v1/portal/invite-codes` | `consumer.ts` | 已对齐 |
| Invite Codes | `PATCH` | `/v1/portal/invite-codes/:code` | `consumer.ts` | 已对齐 |
| Portal Stats | `GET` | `/v1/portal/stats/usage` | `portal-stats.ts` | 已对齐 |
| Portal Stats | `GET` | `/v1/portal/stats/usage-events` | `portal-stats.ts` | 已对齐 |
| Portal Stats | `GET` | `/v1/portal/stats/department-bills` | `portal-stats.ts` | 已对齐 |

### 4.2 Model Assets / Agent Catalog

| 资源 | Method | Path | 前端 service | 备注 |
| --- | --- | --- | --- | --- |
| Model Assets | `GET` | `/v1/ai/model-assets` | `model-asset.ts` | 已对齐 |
| Model Assets | `GET` | `/v1/ai/model-assets/options` | `model-asset.ts` | 已对齐 |
| Model Assets | `GET` | `/v1/ai/model-assets/:assetId` | `model-asset.ts` | 已对齐 |
| Model Assets | `POST` | `/v1/ai/model-assets` | `model-asset.ts` | 已对齐 |
| Model Assets | `PUT` | `/v1/ai/model-assets/:assetId` | `model-asset.ts` | 已对齐 |
| Model Bindings | `POST` | `/v1/ai/model-assets/:assetId/bindings` | `model-asset.ts` | 已对齐 |
| Model Bindings | `PUT` | `/v1/ai/model-assets/:assetId/bindings/:bindingId` | `model-asset.ts` | 已对齐 |
| Model Bindings | `POST` | `/v1/ai/model-assets/:assetId/bindings/:bindingId/publish` | `model-asset.ts` | 已对齐 |
| Model Bindings | `POST` | `/v1/ai/model-assets/:assetId/bindings/:bindingId/unpublish` | `model-asset.ts` | 已对齐 |
| Price Versions | `GET` | `/v1/ai/model-assets/:assetId/bindings/:bindingId/price-versions` | `model-asset.ts` | 已对齐 |
| Price Versions | `POST` | `/v1/ai/model-assets/:assetId/bindings/:bindingId/price-versions/:versionId/restore` | `model-asset.ts` | 已对齐 |
| Agent Catalog | `GET` | `/v1/ai/agent-catalog` | `agent-catalog.ts` | 已对齐 |
| Agent Catalog | `GET` | `/v1/ai/agent-catalog/options` | `agent-catalog.ts` | 已对齐 |
| Agent Catalog | `GET` | `/v1/ai/agent-catalog/:agentId` | `agent-catalog.ts` | 已对齐 |
| Agent Catalog | `POST` | `/v1/ai/agent-catalog` | `agent-catalog.ts` | 已对齐 |
| Agent Catalog | `PUT` | `/v1/ai/agent-catalog/:agentId` | `agent-catalog.ts` | 已对齐 |
| Agent Catalog | `POST` | `/v1/ai/agent-catalog/:agentId/publish` | `agent-catalog.ts` | 已对齐 |
| Agent Catalog | `POST` | `/v1/ai/agent-catalog/:agentId/unpublish` | `agent-catalog.ts` | 已对齐 |

### 4.3 AI Quota / AI Sensitive

| 资源 | Method | Path | 前端 service | 备注 |
| --- | --- | --- | --- | --- |
| AI Quota | `GET` | `/v1/ai/quotas/menu-state` | `ai-quota.ts` | 已对齐 |
| AI Quota | `GET` | `/v1/ai/quotas/routes` | `ai-quota.ts` | 已对齐 |
| AI Quota | `GET` | `/v1/ai/quotas/routes/:routeName/consumers` | `ai-quota.ts` | 已对齐 |
| AI Quota | `PUT` | `/v1/ai/quotas/routes/:routeName/consumers/:consumerName/quota` | `ai-quota.ts` | 已对齐 |
| AI Quota | `POST` | `/v1/ai/quotas/routes/:routeName/consumers/:consumerName/delta` | `ai-quota.ts` | 已对齐 |
| AI Quota | `GET` | `/v1/ai/quotas/routes/:routeName/consumers/:consumerName/policy` | `ai-quota.ts` | 已对齐 |
| AI Quota | `PUT` | `/v1/ai/quotas/routes/:routeName/consumers/:consumerName/policy` | `ai-quota.ts` | 已对齐 |
| AI Quota | `GET` | `/v1/ai/quotas/routes/:routeName/schedules` | `ai-quota.ts` | 已对齐 |
| AI Quota | `PUT` | `/v1/ai/quotas/routes/:routeName/schedules` | `ai-quota.ts` | 已对齐 |
| AI Quota | `DELETE` | `/v1/ai/quotas/routes/:routeName/schedules/:ruleId` | `ai-quota.ts` | 已对齐 |
| AI Sensitive | `GET` | `/v1/ai/sensitive-words/menu-state` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `GET` | `/v1/ai/sensitive-words/status` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `POST` | `/v1/ai/sensitive-words/reconcile` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `GET` | `/v1/ai/sensitive-words/detect-rules` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `POST` | `/v1/ai/sensitive-words/detect-rules` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `PUT` | `/v1/ai/sensitive-words/detect-rules/:id` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `DELETE` | `/v1/ai/sensitive-words/detect-rules/:id` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `GET` | `/v1/ai/sensitive-words/replace-rules` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `POST` | `/v1/ai/sensitive-words/replace-rules` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `PUT` | `/v1/ai/sensitive-words/replace-rules/:id` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `DELETE` | `/v1/ai/sensitive-words/replace-rules/:id` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `GET` | `/v1/ai/sensitive-words/audits` | `ai-sensitive.ts` | 已对齐；当前响应已新增 `guardCode`、`blockedDetails[]`，并保留 `blockedReasonJson` |
| AI Sensitive | `GET` | `/v1/ai/sensitive-words/system-config` | `ai-sensitive.ts` | 已对齐 |
| AI Sensitive | `PUT` | `/v1/ai/sensitive-words/system-config` | `ai-sensitive.ts` | 已对齐 |

### 4.4 Organization

| 资源 | Method | Path | 前端 service | 备注 |
| --- | --- | --- | --- | --- |
| Departments | `GET` | `/v1/org/departments/tree` | `organization.ts` | 已对齐 |
| Departments | `POST` | `/v1/org/departments` | `organization.ts` | 已对齐 |
| Departments | `PUT` | `/v1/org/departments/:departmentId` | `organization.ts` | 已对齐 |
| Departments | `PATCH` | `/v1/org/departments/:departmentId/move` | `organization.ts` | 已对齐 |
| Departments | `DELETE` | `/v1/org/departments/:departmentId` | `organization.ts` | 已对齐 |
| Accounts | `GET` | `/v1/org/accounts` | `organization.ts` | 已对齐 |
| Accounts | `POST` | `/v1/org/accounts` | `organization.ts` | 已对齐 |
| Accounts | `PUT` | `/v1/org/accounts/:consumerName` | `organization.ts` | 已对齐 |
| Accounts | `PATCH` | `/v1/org/accounts/:consumerName/assignment` | `organization.ts` | 已对齐 |
| Accounts | `PATCH` | `/v1/org/accounts/:consumerName/status` | `organization.ts` | 已对齐 |
| Template | `GET` | `/v1/org/template` | `organization.ts` | 文件下载 |
| Export | `GET` | `/v1/org/export` | `organization.ts` | 文件下载 |
| Import | `POST` | `/v1/org/import` | `organization.ts` | `multipart/form-data` |

## 5. Jobs 内部接口

这组接口后端已经暴露，前端已补 `src/services/jobs.ts` 和 `/system/jobs` 页面。

| Method | Path | 说明 |
| --- | --- | --- |
| `GET` | `/internal/jobs` | 任务列表 |
| `GET` | `/internal/jobs/:name` | 单任务详情 |
| `POST` | `/internal/jobs/:name/trigger` | 手动触发任务 |

## 6. 当前建议前端重点核对项

### A. 本轮确认的重构差异

1. `system/config 定义变化`
   legacy `SystemController#getConfigs` 返回的是纯配置 `Map<String, Object>`，当前 Go `/system/config` 返回的是 `ConfigRes{ module, serverAddress, explicitRenameTargets, contractDirectories, properties }`；前端应只读取 `properties`，不要再把整个响应当配置字典。
2. `MCP server 更新接口改为 RESTful`
   legacy `McpServerController#addOrUpdateMcpInstance` 使用 `PUT /v1/mcpServer`；当前 Go 使用 `PUT /v1/mcpServer/:name`，前端已改为跟随当前契约。
3. `列表接口的分页包装需要显式消费`
   `/v1/ai/routes`、`/v1/mcpServer`、`/v1/mcpServer/consumers` 当前都返回分页包装对象，页面层应显式读取 `.data`，而不是把响应直接当数组。
4. `AI Provider 2.2.1 rawConfigs 已补齐`
   `/v1/ai/providers` 现支持 `providerDomain`、`providerBasePath`、`promoteThinkingOnEmpty`、`hiclawMode`、`bedrockPromptCachePointPositions`、`promptCacheRetention`，其中 `vertex` 支持 `tokens[]` 驱动的 Express Mode，不再强制 `vertexAuthKey`。
5. `Wasm builtin metadata 已切到新快照优先`
   `/v1/wasm-plugins/:name/config` 与 `/readme` 现优先读取 `backend/resource/public/plugin`，`ai-statistics` 的 `value_length_limit` 默认值已更新为 `32000`，`ai-security-guard` README 也同步到结构化拦截结果说明。
6. `AI Sensitive 审计返回结构化阻断结果`
   `/v1/ai/sensitive-words/audits` 当前除原 `blockedReasonJson` 外，还会解析返回 `requestId`、`guardCode` 与 `blockedDetails[]`，前端可以直接展示首条风险详情。

### B. 本轮已补齐的前端接线

1. `GET /v1/wasm-plugins/:name/readme`
2. `DELETE /v1/global/plugin-instances/:pluginName`
3. `DELETE /v1/routes/:name/plugin-instances/:pluginName`
4. `DELETE /v1/domains/:name/plugin-instances/:pluginName`
5. `service-scope plugin instances` 整组接口：
   `GET /v1/services/:name/plugin-instances`
   `GET /v1/services/:name/plugin-instances/:pluginName`
   `PUT /v1/services/:name/plugin-instances/:pluginName`
   `DELETE /v1/services/:name/plugin-instances/:pluginName`
6. `GET /v1/consumers/:name`
7. `/internal/jobs*` 管理接口
8. `/v1/portal/stats/*` 统计接口

### C. 建议前端复核但当前未见硬性问题

1. `/v1/routes` 的列表结果现在仍是 legacy 风格分页包装，当前页面通过 `route-compat.ts` 消费，但建议继续确认新旧两套 route 类型是否要收敛成一套。
2. `route.ts` 与 `route-compat.ts` 仍并存，建议确认页面入口当前到底以哪套类型为准，避免同路径不同类型定义继续漂移。
3. 匿名可访问页面与静态资源现在依赖后端白名单与 SPA fallback，前端若再新增新入口路径，需要同步确认是否属于匿名页面。

## 7. 核对建议

建议前端按下面顺序核对：

1. 先确认 `登录 / 初始化 / MCP / AI Route` 页面在当前 Go 后端下已经完全按新契约消费。
2. 再按页面功能核对 `Wasm Plugin README`、`plugin instance delete`、`service-scope plugin instance` 是否需要补接。
3. 最后抽样验证关键页面：
   `登录 / 初始化 / Dashboard / Route / AI Route / Model Asset / Consumer / Org / MCP`

## 8. 参考代码位置

- 后端入口：[backend/internal/cmd/cmd.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/cmd/cmd.go:1)
- Gateway 路由：[backend/internal/controller/gateway/http.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/controller/gateway/http.go:1)
- Portal 路由：[backend/internal/controller/portal/http.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/controller/portal/http.go:1)
- Session 路由：[backend/internal/controller/session/http.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/controller/session/http.go:1)
- User 路由：[backend/internal/controller/user/http.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/controller/user/http.go:1)
- System 路由：[backend/internal/controller/system/http.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/controller/system/http.go:1)
- Dashboard 路由：[backend/internal/controller/dashboard/http.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/controller/dashboard/http.go:1)
- Jobs 路由：[backend/internal/controller/jobs/http.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/controller/jobs/http.go:1)
- 匿名访问策略：[backend/internal/middleware/auth.go](/home/cloudyi/code/aigateway-group/aigateway-console/backend/internal/middleware/auth.go:1)
- 前端 services：[frontend/src/services](/home/cloudyi/code/aigateway-group/aigateway-console/frontend/src/services)
