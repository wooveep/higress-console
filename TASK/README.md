# GoFrame 重写任务台账

## 状态约定

- `todo`：尚未开始
- `doing`：正在进行
- `done`：已完成
- `blocked`：被外部条件阻塞

## 阶段顺序

1. `P0-baseline-analysis.md`
2. `P1-goframe-foundation.md`
3. `P2-platform-core.md`
4. `P3-gateway-resource-domain.md`
5. `P4-portal-domain.md`
6. `P5-jobs-and-reconcile.md`
7. `P6-cutover-and-rename.md`
8. `P7-dashboard-split-and-metrics.md`

## 统一要求

- 每阶段都要补充“验收点”和“单元测试/集成测试”。
- 每阶段完成后都要执行：
  - `go test ./...`
  - 阶段内 smoke 验证
  - 与 `backend-java-legacy/` 的行为差异记录
- 变更真相源、表结构、K8s 资源映射或前端接口时，需要同步更新阶段文档。

## 当前进度

- `P0`：done
- `P1`：done
- `P2`：done
- `P3`：doing
- `P4`：done
- `P5`：doing（主链已完成，aftercare 中）
- `P6`：done
- `P7`：todo

## 当前焦点

- 主阶段：`P3 / P5 aftercare`
- 当前重点：
  - 继续把“控制面真相源回归”作为第一优先级，收口 `MCP / AI Route / Provider` 到真实 Higress 运行态对象，而不是 generic ConfigMap。
  - 继续收口 `gateway` 域与旧 Java 在 `CRD / 注解 / 字段级语义` 上的差异。
  - 收口 `modelPredicates -> x-higress-llm-model` 的 AI Route 运行时匹配，并把模型资产发布的 `RPM/TPM` 投影为每用户每模型的 runtime 限流规则。
  - 稳定 `dev-redeploy / minikube-dev` 本地联调链路，保证 Go console / Portal / plugin-server 可持续冷启动。
  - 启动 `P7`：把 `/dashboard` 与 `/ai/dashboard` 做强分层，统一页面时间范围，并收口主看板与 AI 看板的指标边界。

## 最近更新

- `P6` 已完成：显式 `higress -> aigateway` 命名、接口、镜像、脚本、Helm 与文档收口已经落地。
- `P3` 已补首批 Java parity 高频缺口：
  - `/v1/wasm-plugins/:name/readme`
  - `global/domain/route/service` plugin instance delete / service-scope 操作
  - `MCP routeMetadata / ingressClass / Route-Service-Wasm` 首批深校验
  - `routes / ai-routes / ai-providers` 已切到真实 `Ingress / ConfigMap / WasmPlugin` 读链，不再只依赖 `console.aigateway.io/type=resource`
  - `MCP save strategy` 已完成收口：`DIRECT/REDIRECT_ROUTE` 的 `transportType / rewrite host / sse_path_suffix`、`consumer level` 注解回写、`OPEN_API / DATABASE` 异常分支、`default-mcp / route auth` 历史兼容清理，以及 `match_list / servers[]` 保留未知字段的 Java 更新语义都已对齐
  - `provider runtime` 已补第二批字段级 parity：`openaiExtraCustomUrls` 多 IP static registry、`vertex-auth.internal` extra service source、`ai-route` service 引用跟随 provider 实际 service source
  - `provider-specific endpoint` 本轮继续补齐：`openai / azure / qwen / zhipuai / claude` 的 endpoint 推导与 URL/域名严格校验，`claude/qwen/zhipuai/vertex/bedrock/ollama` 的保存前归一化也已前移到 gateway service
  - `P3-CP-03` 已完成：`ai-route` 保存已按前端契约接收 `pathPredicate / headerPredicates / urlParamPredicates / upstreams / fallbackConfig`，并补齐 provider 校验、fallback response code / strategy 校验、以及 fallback 关闭时的 `key-auth` 清理
  - `P3-CP-04` 已完成：`ai-proxy / ServiceSource / service-scope ACTIVE_PROVIDER_ID / ai-route resync` 已完成收口
  - `P3` 资源转换测试已补齐：`AI Route` 的 `public/internal/fallback ingress` 映射、`model routing / fallback header` 注解传递、以及 `Provider <-> ai-proxy wasm payload` roundtrip 已有单测覆盖
  - `P3-CP-01` 已完成：`mcp-servers` 真实读链已切到 `Ingress + higress-config + McpBridge/registry + route auth/plugin instance`
  - `P3-CP-02` 已完成：`mcp-servers` 保存/删除副作用已对齐 Java save strategies
  - `P3-CP-05` 已完成：Portal/Jobs 已切到消费真实 `mcp-servers / ai-routes / ai-providers` 聚合结果
  - `P3-CP-06` 已完成 real cluster smoke：`provider / ai-route / mcp-server` create-update-delete-cleanup 已在 `minikube / aigateway-system` 验证通过
- `P3/P4/P5` 本轮功能闭环已补齐：
  - `Plugin Management` 已接上 README、绑定删除、service-scope plugin instances
  - `Consumer Management` 已接上 `/v1/consumers/:name` 只读详情
  - `AI Dashboard` 已接上 `/v1/portal/stats/*`
  - `System Settings` 已接上 `/system/jobs`
- `P4` 已推进到 Portal 真表收口：
  - 共享 schema 真相源已迁到 `aigateway-portal/backend/schema/shared`
  - Console 只维护自有表，不再兜底创建共享表
  - `Consumer / Org / Invite / Grants / Model Assets / Agent Catalog / AI Quota` 主链已切到 Portal 真表
  - `P4-AF-01 ~ P4-AF-03` 已完成：Portal 持久化 helper、契约测试与 `Portal Stats` 趋势图/CSV 导出均已补齐
- `P5` 任务体系已可用：
  - `portal-consumer-projection`
  - `portal-consumer-level-auth-reconcile`
  - `ai-sensitive-projection`
  - `ai-model-rate-limit-reconcile`
  - `ai-plugin-execution-order-reconcile`
  - 当前处于 `aftercare`：继续补失败重试、运行态观测与失败排查体验
  - `P3/P5 aftercare` 本轮新增：`modelPredicates` 已真正投影到 `x-higress-llm-model` 运行时匹配；模型资产已开始通过 projection -> builtin `cluster-key-rate-limit / ai-token-ratelimit` 生成每用户每模型规则，并记录 skip reason。
- 本地开发链路最近已补齐：
  - `console` 镜像改为 monorepo 构建入口，兼容对 `aigateway-portal/backend` 的本地 `replace`
  - `portal` 镜像构建 Go 版本已对齐 `go.mod` 的 `1.25`
  - `console` readiness probe 已从 `/` 调整到 `/healthz/ready`，避免因匿名访问策略返回 `403`

## 主要待办

- `P3`
  - 继续补 `K8s CRD / 注解 / 字段级` 对齐，而不只是 generic resource 可用。
  - 继续收口 builtin/custom Wasm plugin metadata 与 schema/readme 的统一来源，减少对 `backend-java-legacy/` 的回退依赖。
  - `MCP Server` 继续收口到 `Ingress + higress-config + McpBridge/registry + auth/plugin instance` 联合真相源。
  - `MCP Server` 继续收口到更完整的 `Ingress + higress-config + McpBridge/registry + auth/plugin instance` 字段级语义，而不再只是 save strategy parity。
  - 继续补 `AI Route / Provider` 余下的字段级 Java parity，而不只是当前 save / delete runtime 副作用闭环。
  - 继续补 `model asset RPM/TPM -> AI Route runtime rate-limit rules` 的 live smoke 与 cluster 验证。
- `P5`
  - 补失败重试与更完整的任务运行验证。
  - 继续完善 jobs 的运行态观测和失败排查体验。
- `P6` 后续收尾
  - 历史归档名称仍保留 `backend-java-legacy/`，待 builtin plugin 资源正式迁入新目录后，再评估 legacy 目录退场顺序。
