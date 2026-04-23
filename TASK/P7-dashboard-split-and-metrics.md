# P7 Dashboard 指标分层与时间联动

- 状态：`doing`
- 依赖：`P3`, `P4`, `P5`, `P6`

## 目标

- 将 `/dashboard` 收口为平台与网关公共运行态总览，不再混入 AI 独有指标。
- 将 `/ai/dashboard` 收口为 AI 专属监控与用量页，保留并整合现有 `AI 用量统计`。
- 统一 `/ai/dashboard` 顶部原生监控与下半部分 `AI 用量统计` 的页面级时间范围。

## 最终信息架构

- `/dashboard`
  - `Platform`：CPU、内存、连接数、Gateway Pod Count
  - `Gateway Request`：Downstream Request Count、Downstream Success Rate、Downstream QPS Trend、Downstream Latency
  - `Upstream Health`：Upstream Request Count、Upstream Attempt Success Rate、Upstream QPS Trend、Upstream Latency
  - `Exceptions`：Downstream 5xx Count、Upstream 5xx/Timeout Count、Failure Route TopN、Slow Route TopN
  - `Resource Scale`：Routes、Domains、Plugins
- `/ai/dashboard`
  - 顶部原生监控
    - `AI Overview`：AI Request Count、AI Request Success Rate、Total Tokens、Estimated Cost、Upstream Attempt Success Rate
    - `Token Runtime`：Token Per Second、Cache Token Per Second、Image Token Per Second
    - `AI Request`：Downstream AI Route Request Trend、Upstream Provider/Service Request Trend、Downstream Latency、Upstream Latency
    - `AI Exceptions`：Failed Requests、Slow Requests、Error Code TopN
  - 下半部分 `AI 用量统计`
    - `用量概览`：总请求数、总 Token、用量趋势、用户+模型聚合表
    - `使用记录`：请求明细、状态、错误、时长、费用
    - `部门账单`：部门请求数、总 Token、总成本、活跃消费者

## 数据口径

- `/dashboard` 只依赖 Prometheus 与 K8s 资源统计，不依赖 `billing_usage_event`。
- `/ai/dashboard`
  - Prometheus：Token 吞吐、上下游时延、上游请求趋势、上游成功率
  - `billing_usage_event`：AI Request Count、AI Request Success Rate、Total Tokens、Estimated Cost、Downstream AI Route Trend、Failed Requests、Slow Requests、Error Code TopN
- 统一语义：
  - 请求成功率：`HTTP 2xx / total requests`
  - 上游尝试成功率：`2xx upstream responses / total upstream responses`
  - 慢请求：`service_duration_ms > 5000`
  - AI `Total Tokens` 使用窗口汇总口径，保证长窗口总量不小于短窗口

## 任务

- [x] 后端拆分 `MAIN` 与 `AI` 的原生行构建逻辑，不再共用统一监控模板。
- [x] `/dashboard` 移除 Token、Cost、Provider/Model/Consumer、AI Routes、异常明细表等 AI 专属内容。
- [x] `/dashboard` 新增 `Gateway Pod Count`、公共请求/异常/资源规模区块。
- [x] `/ai/dashboard` 顶部原生监控调整为 `AI Overview / Token Runtime / AI Request / AI Exceptions`。
- [x] `AI Exceptions` 新增 `Error Code TopN` 表。
- [x] 前端上提页面级时间状态，统一 `NativeDashboardView` 与 `PortalStatsPanel` 的 `from/to`。
- [x] `PortalStatsPanel` 接收共享时间范围，不新增 Portal Stats 后端接口。
- [x] 更新中英文文案，覆盖新行名、卡片名、表格列名。

## 验收点

- [ ] `/dashboard` 页面不再出现 Total Tokens、Estimated Cost、Provider Usage、Model Usage、Consumer Usage、AI Route 请求趋势。
- [ ] `/dashboard` 在 Portal DB 不可用时仍可加载完整公共监控。
- [ ] `/ai/dashboard` 顶部原生监控与下半部分 `AI 用量统计` 使用同一套 `from/to`。
- [ ] `/ai/dashboard` 的 Total Tokens 在 5 分钟、15 分钟、24 小时、7 天窗口下满足单调不减。
- [ ] `/ai/dashboard` 的 Token/Cache/Image 吞吐在低频流量下可稳定出数。
- [ ] `Failed Requests / Slow Requests` 与 `usage-events` 的 request path、error、duration 字段语义一致。

## 测试

- [x] `go test ./internal/service/platform ./internal/service/portal`
- [x] `npm run typecheck`
- [ ] `/dashboard` smoke：主看板仅展示公共指标
- [ ] `/ai/dashboard` smoke：时间切换后顶部监控与 `AI 用量统计` 同步刷新

## 本轮说明

- 这轮优先完成信息架构与时间联动，不扩展 WAF、XDS、Service TopN 到主看板首版。
- 主看板异常 TopN 采用 Prometheus 聚合结果；AI 请求级异常仍在 `/ai/dashboard` 展示。
