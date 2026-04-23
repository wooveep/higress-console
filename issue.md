# Console UI 巡检问题记录

巡检时间：2026-04-16 00:25 - 00:29 CST  
巡检入口：`http://127.0.0.1:8080`  
登录账号：`admin / admin`

## 已验证的控件

- 登录页：用户名、密码、登录按钮
- 顶部导航：`/dashboard`、`/ai/dashboard`
- 页面级控件：`Refresh now`、侧边栏折叠按钮
- Dashboard 时间控件：`End time`、`Time range`、`Auto refresh` 点击可响应
- AI Dashboard 区块：`AI Overview`、`Token Runtime`、`AI Request`、`AI Exceptions`

说明：

- `AI 用量统计` 区域被 `Portal database is unavailable` 整体拦截，内部筛选/表格/导出控件当前无法继续验证。

## 发现的问题

### 1. `/dashboard` 下游 QPS 趋势图时间轴不按所选时间范围展开

现象：

- 页面时间范围选择的是 `Last 1 hour`。
- 但 `Downstream QPS Trend` 图表上的时间标签只显示最近几分钟，如 `00:19 / 00:24 / 00:28`。
- 图表视觉上只覆盖“有数据的最后几分钟”，没有把完整的 1 小时时间窗展示出来。

复现：

1. 登录后进入 `http://127.0.0.1:8080/dashboard`
2. 保持 `Time range = Last 1 hour`
3. 观察 `Gateway Request -> Downstream QPS Trend`

接口证据：

- `GET /dashboard/native?type=MAIN&from=<now-1h>&to=<now>`
- 返回的 `Downstream QPS Trend` 数据点时间为：
  - `2026-04-16 00:19:54`
  - `2026-04-16 00:20:54`
  - ...
  - `2026-04-16 00:27:54`
- 前端当前直接拿“首点/中点/尾点”做 X 轴标签，并用“数据点最小时间 ~ 最大时间”作为图表 domain。

影响：

- 用户会误以为时间范围没有生效。
- 稀疏数据场景下，1 小时、6 小时、24 小时图会被压缩成“最近几分钟/几小时”。

疑似位置：

- [NativeDashboardLineChart.vue](./frontend/src/features/dashboard/NativeDashboardLineChart.vue)
  - `xDomain` 目前基于 `lineData` 的最早/最晚点，而不是页面实际 `from/to`
  - `xTicks` 目前取 `first / middle / last` 数据点，不是取选中窗口的边界刻度

建议：

- 图表 X 轴 domain 改为页面实际时间窗 `from/to`
- 稀疏数据允许中间没有点，但时间轴仍按所选窗口完整展开
- X 轴刻度按窗口固定生成，而不是按返回点位生成

### 2. `/dashboard` 下游时延图表缺少可读性更好的自适应缩放

现象：

- `Downstream Latency` 图表会显示 `P50 / P90 / P99`
- 当前 Y 轴直接使用所有序列的原始最小值和最大值
- 在 `P99` 明显高于 `P50/P90` 时，低位曲线变化会被压得很平，肉眼不容易判断波动

复现：

1. 登录后进入 `http://127.0.0.1:8080/dashboard`
2. 观察 `Gateway Request -> Downstream Latency`

接口证据：

- 当前 1 小时窗口内返回值大致为：
  - `P50 ≈ 3.17ms ~ 3.59ms`
  - `P90 ≈ 4.91ms ~ 7.81ms`
  - `P99 ≈ 9.36ms ~ 9.78ms`
- 图表虽然 technically 会缩放，但没有：
  - 上下 padding
  - nicer ticks
  - 针对多分位序列差异的可读性优化

影响：

- 图表看起来“像没缩放”
- 小幅波动不明显，容易误判系统稳定性

疑似位置：

- [NativeDashboardLineChart.vue](./frontend/src/features/dashboard/NativeDashboardLineChart.vue)
  - `yDomain` 直接取 `Math.min(...values)` / `Math.max(...values)`
  - 没有额外 padding，也没有 nicer tick 处理

建议：

- 给 Y 轴增加上下 padding
- 使用 nicer tick 规则，而不是只画 `max / middle / min`
- 评估是否对 latency 图增加单独缩放策略，避免 `P99` 把 `P50/P90` 压平

### 3. `/ai/dashboard` 的 `AI 用量统计` 被错误判定为不可用

现象：

- 页面显示两处 `Portal database is unavailable`
- 整个 `AI 用量统计` 区域被 `PortalUnavailableState` 拦截
- 但实际 Portal 统计接口是可访问的，并返回 `200 OK + []`

复现：

1. 登录后进入 `http://127.0.0.1:8080/ai/dashboard`
2. 查看 `AI 用量统计`

接口证据：

- `GET /system/config` 返回：
  - `portal.enabled = true`
  - `portal.healthy = false`
- 但以下接口都返回 `200 OK`：
  - `GET /v1/portal/stats/usage`
  - `GET /v1/portal/stats/usage-events`
  - `GET /v1/portal/stats/department-bills`

影响：

- 页面把“Portal 不健康”直接等同于“Portal 功能不可用”
- 即使统计接口已经可用，`AI 用量统计` 仍完全不可操作
- 这会掩盖真实问题，也阻断用户继续使用筛选/明细/账单功能

疑似原因：

- 前端：
  - [usePortalAvailability.ts](./frontend/src/composables/usePortalAvailability.ts) 直接用 `!portalHealthy` 判定不可用
  - [app.ts](./frontend/src/stores/app.ts) 从 `/system/config` 读 `portalHealthy`
- 后端：
  - [portaldb/client.go](./backend/utility/clients/portaldb/client.go) 中 `SQLClient.Healthy()` 优先返回 `c.err`
  - `c.err` 在 `New()` 阶段可能由一次启动期 `AutoMigrate/EnsureSchema` 错误写死
  - 即使后续统计查询实际已经能跑，`portalHealthy` 仍可能长期保持 `false`

建议：

- 前端不要用全局 `portalHealthy=false` 直接拦死 `AI 用量统计`
- 优先按页面实际接口结果判断是否降级
- 后端排查 `portalHealthy` 的定义是否被启动期错误永久污染

### 4. 登录页初始化阶段会触发 401 和 favicon 404

现象：

- 登录前页面控制台出现：
  - `401 Unauthorized` on `/user/info`
  - `404 Not Found` on `/favicon.ico`

影响：

- 登录前控制台噪音较大
- 容易干扰真正的错误定位

优先级：

- 低，可后置处理

## 额外观察

### 2026-04-16 01:00 CST 复查更新

- `portal.enabled = true`
- `portal.healthy = true`
- `AI 用量统计` 当前不再是“被健康检查误杀”的状态

最新复查结果：

- `GET /v1/portal/stats/usage?from=<now-1h>&to=<now>` 返回空数组，是因为 `billing_usage_event` 最近 1 小时内确实没有新的完成事件
- 把时间窗放宽到最近 12 小时后，`usage / usage-events / usage-event-options` 都可以返回真实数据
- 当前账务表里最近一条 `parsed` 事件是：
  - `2026-04-15 16:38:51 +0800`
  - `consumer = liyuntian`
  - `model = doubao-seed-2-0-pro-260215`
  - `total_tokens = 563`

关于用户提供的流式测试命令：

- 复测命令：
  - `curl -i -N -sS --max-time 12 'http://ai.local/doubao/v1/chat/completions' ...`
- 结果：
  - 请求可以返回 SSE 流
  - 但在 `12s` 超时前，返回 chunk 中 `usage` 一直是 `null`
  - 客户端在最终 usage 回传前主动中断连接
  - 数据库里不会稳定新增 `billing_usage_event`

结论：

- 当前 `/ai/dashboard` 的“最近 1 小时无数据”并不完全是 Console 查询问题
- 更关键的上游原因是：这条用于测试的 Doubao SSE 请求在当前参数下经常不能自然结束，因此不会稳定生成账务事件
- 要继续验证 `AI 用量统计`，应优先使用：
  - 更短的 prompt
  - 不带 `--max-time 12` 的流式请求
  - 或能在服务端返回最终 usage 的非流式请求

### `/ai/dashboard` 顶部原生监控当前主要依赖 Prometheus，而 `AI 用量统计` 依赖 Portal 统计链路

所以当前页面会出现一种割裂状态：

- 顶部 `AI Dashboard` 可以正常显示部分原生监控
- 下半部分 `AI 用量统计` 因 `portalUnavailable` 直接整体不可用

这会让用户误以为“页面整体坏了”，实际上是两条数据链路状态不一致。

## 建议修复顺序

1. 先修 `portalUnavailable` 的误判，恢复 `AI 用量统计` 可访问性
2. 再修 `/dashboard` 的时间轴 domain 问题
3. 最后优化 latency 图的 Y 轴缩放与刻度策略
