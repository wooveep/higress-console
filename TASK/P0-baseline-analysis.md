# P0 基线分析

- 状态：`done`
- 依赖：无

## 目标

- 识别旧 Java 后端真实结构和业务边界。
- 冻结 GoFrame 重写的分层、迁移顺序、命名规则和对照基线。

## 已完成

- [x] 确认旧后端为 `console + sdk` 双模块 Java 结构。
- [x] 确认 `console` 负责控制面 API、Portal JDBC、会话、静态资源、定时任务。
- [x] 确认 `sdk` 负责 K8s/CRD/插件/AI/MCP 适配。
- [x] 输出 `backend-analysis.md`。
- [x] 输出 `backend-business-lines.md`。
- [x] 输出 `backend-goframe-architecture.md`。
- [x] 确认旧后端迁移基线位于 `backend-java-legacy/`。

## 验收点

- [x] 可以从文档中看清四条业务主线。
- [x] 可以从文档中看清 Controller -> Service -> SDK/DB/K8s 的关系。
- [x] 可以从文档中看清 GoFrame 的目标目录与迁移顺序。

## 测试准备

- [x] 建立 `test/contracts/` 目录结构。
- [x] 从 Java 基线提取首批真实契约样本。
  - `backend/test/contracts/session/login-success.json`
  - `backend/test/contracts/system/info-success.json`
  - `backend/test/contracts/routes/list-success.json`
  - `backend/test/contracts/mcp/list-success.json`
  - `backend/test/contracts/ai-routes/list-success.json`
