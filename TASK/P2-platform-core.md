# P2 平台核心

- 状态：`done`
- 依赖：`P1`

## 目标

- 迁移系统初始化、会话鉴权、配置读写、Dashboard、Health、Landing。

## 任务

- [x] 迁移管理员登录与登出。
- [x] 迁移管理员 Secret 读写逻辑。
- [x] 迁移系统初始化逻辑。
- [x] 迁移默认证书、默认 Domain、默认 Route 初始化。
- [x] 迁移 Dashboard 配置读取。
- [x] 迁移 `/healthz`、`/landing`、`/system/*`、`/session/*`。
- [x] 补 `/user/info` 与 `/user/changePassword`，完成管理员登录态闭环。

## 验收点

- [x] 管理员可登录并保持会话。
- [x] 未登录访问受保护接口会被拦截。
- [x] 系统初始化链路可运行。
- [x] Landing 与静态资源托管正常。

## 测试

- [x] Session service 单元测试。
- [x] System service 单元测试。
- [x] Config service 单元测试。
- [x] Session / System / Health 契约测试。

## 本轮说明

- 当前管理员 Secret 和 `aigateway-config` 通过 `utility/clients/k8s` 的 fake client 维持运行时语义，真实 K8s 持久化接入留待后续收口。
- Dashboard 已完成接口和静态配置主链；真实 Grafana datasource 自动初始化与代理链路暂未接入。
