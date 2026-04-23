# AIGateway Console TODO（Consumer 深化整合）

## 已完成
- [x] Consumer 真源切到 `portal_user`（列表/详情/创建/更新/状态/删除）。
- [x] 独立密码重置接口：`POST /v1/consumers/{name}/password/reset`。
- [x] 邀请码管理接口：创建/列表/禁用（并支持重新启用）。
- [x] MCP 名称前后端 RFC1123 小写校验，K8s 422 -> HTTP 400。
- [x] AI Quota 的 consumer 校验改为查 DB 真源。
- [x] DB -> key-auth 周期投影与启动回填任务（幂等）。
- [x] 组织架构前端新增重置密码与邀请码管理入口。
- [x] Route/AI Route/MCP 消费者下拉并集策略，避免历史值丢失。

## 待完成（建议后续迭代）
- [ ] 为 `portal_user` / `portal_api_key` 增补显式索引校验脚本与启动告警。
- [ ] 增加投影同步指标（成功/失败/耗时/差异数）并接入监控。
- [ ] 增加 migration dry-run 模式（只比对不写入）用于生产演练。
- [ ] 增加邀请码使用明细表（审计需求）与后台查询。
- [ ] 增加 API Key 审计日志（创建、禁用、删除、使用来源）。

## 发布前检查清单
- [ ] Console 后端编译通过（可先 `-Dpmd.skip=true` 验证功能链路）。
- [ ] Console 前端 build 通过。
- [ ] 同一数据库下，Portal 注册用户可在 Console 组织架构可见。
- [ ] 邀请码时间字段显示正常，不再出现 `Invalid Date`。
- [ ] disabled 用户在一个同步周期内完成网关鉴权失效。
