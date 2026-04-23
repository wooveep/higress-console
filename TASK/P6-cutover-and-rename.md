# P6 切换与命名收口

- 状态：`done`
- 依赖：`P2`, `P3`, `P4`, `P5`

## 目标

- 切换前端服务调用。
- 完成显式 `higress` -> `aigateway` 命名收口。

## 任务

- [x] 前端服务调用切换到 Go 后端。
- [x] 显式 `higress` 配置键改为 `aigateway`。
- [x] 显式 `higress` 服务名、镜像名、文档名改为 `aigateway`。
- [x] `/system/aigateway-config` 这类接口改成 `aigateway` 命名。
- [x] 更新部署脚本、镜像构建、文档和验收手册。

## 验收点

- [x] 前端在 Go 后端上可运行。
- [x] 显式旧命名不再继续扩散。
- [x] 部署配置、镜像与文档已切换到新命名。

## 测试

- [x] 前后端联调 smoke。
- [x] 切换后回归测试。
- [x] 关键接口契约回归。

## 本轮说明

- 后端配置接口已从 `/system/higress-config` 硬切到 `/system/aigateway-config`，不再保留旧路径兼容层。
- `platform` 服务新增一次性迁移逻辑：当 ConfigMap 只存在旧 `higress` 键时，会自动迁移为 `aigateway` 并回写。
- 前端语言存储键已从 `higress-console.language` 切到 `aigateway-console.language`，启动时会单向迁移旧值。
- 默认 Domain、MCP 模板值、前端示例文案、Go module/import、Helm chart、脚本路径、仓库目录名都已同步收口到 `aigateway-console`。
- 历史归档语义保持不动：`backend-java-legacy/` 与根目录历史记录文件未做清洗式重写。
