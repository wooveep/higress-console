<h1 align="center">
    <img src="https://img.alicdn.com/imgextra/i2/O1CN01NwxLDd20nxfGBjxmZ_!!6000000006895-2-tps-960-290.png" alt="AIGateway" width="240" height="72.5">
  <br>
  Gateway Console for AIGateway
</h1>

AIGateway Console 用于管理 AIGateway 的配置规则及其他开箱即用的能力集成，首个可用版本考虑基于 kubernetes 部署环境，预期包含服务管理、路由管理、域名管理等基础能力。
后续规划逐步迭代可观测能力、插件能力、登录管理能力。

当前平台正式发布口径为 `1.0.0`。  
正式发布说明、镜像包说明和部署说明统一以根目录 `docs/release/1.0.0/` 为准。

## 前置介绍

此项目包含前端（NodeJS）、后端（GoFrame）两个部分，前端（frontend）部分在构建完成后会随着后端代码（goframe）一起部署。

## 本地启动

### 推荐仓库级入口

优先使用仓库根目录脚本，而不是手工拼接构建/部署命令：

```bash
cd /path/to/aigateway-group
python3 ./scripts/aigateway-dev.py show
python3 ./scripts/aigateway-dev.py check-connectivity
python3 ./scripts/aigateway-dev.py build --components console
./start.sh dev
```

如果你准备本地源码运行 `console` 或 `portal`，推荐先只把集群里的核心依赖拉起来：

```bash
python3 ./scripts/aigateway-dev.py minikube-dev --core-only
```

这会保留 `postgresql / redis / controller / plugin-server / gateway / pilot`，同时在 Helm 层禁用集群内的
`aigateway-console` 与 `aigateway-portal`，避免和本地源码进程抢端口、抢镜像验证入口。

`console` 本地前后端一键启动：

```bash
cd aigateway-console
./start.sh
```

项目标准依赖端口建议对齐为：

```text
backend  -> http://127.0.0.1:18081
frontend -> http://127.0.0.1:3001
postgres -> 127.0.0.1:5432
grafana  -> http://127.0.0.1:3000
```

这些端口与仓库级 `minikube-dev --core-only` + port-forward 约定对齐，避免和集群内 `8080 / 8081`
入口冲突。常用覆盖参数：

```text
CONSOLE_BACKEND_PORT
CONSOLE_FRONTEND_PORT
CONSOLE_LISTEN_ADDR
PORTAL_DB_DRIVER / PORTAL_DB_HOST / PORTAL_DB_PORT / PORTAL_DB_USER / PORTAL_DB_PASSWORD / PORTAL_DB_NAME / PORTAL_DB_PARAMS
AIGATEWAY_CONSOLE_GRAFANA_SERVICE / AIGATEWAY_CONSOLE_GRAFANA_PORT / AIGATEWAY_CONSOLE_GRAFANA_PATH
```

当前 `console` 镜像链路已经固定为：

```text
frontend npm run build
-> backend/resource/public/html
-> backend/Dockerfile
-> aigateway/console:<tag>
```

如果是本地源码直跑，并且需要验证 `Organization / Model Assets / Agent Catalog / AI Sensitive`
这些 Portal 相关页面，还需要额外提供可用的 Portal DB 配置：

```text
clients.portaldb.enabled=true
AIGATEWAY_CONSOLE_PORTALDB_DRIVER=postgres
AIGATEWAY_CONSOLE_PORTALDB_DSN=host=127.0.0.1 port=5432 user=postgres password=postgres dbname=aigateway_portal sslmode=disable
```

如果是 K8S / Helm 部署，Console 现在会优先从 Helm 注入的结构化依赖配置自动发现并连接：

```text
PORTAL_DB_DRIVER / PORTAL_DB_HOST / PORTAL_DB_PORT / PORTAL_DB_USER / PORTAL_DB_PASSWORD / PORTAL_DB_NAME / PORTAL_DB_PARAMS
AIGATEWAY_CONSOLE_GRAFANA_SERVICE / AIGATEWAY_CONSOLE_GRAFANA_PORT / AIGATEWAY_CONSOLE_GRAFANA_PATH
AIGATEWAY_CONSOLE_NAMESPACE / AIGATEWAY_CONSOLE_CLUSTER_DOMAIN
```

`helm/dev-mode.yaml` 里的 `dev.portForward` 只负责把集群里的服务暴露到本机端口，不参与 Console
后端依赖发现。

未提供 `portaldb` 时，Console 现在会进入“Portal 功能不可用但页面不报错”的降级模式；
对应页面会显示 `Portal database is unavailable`，而不是连续弹出 `503` 错误框。

仓库级开发脚本默认还会把常用依赖暴露到本机，便于 Console 联调：

```text
127.0.0.1:8080   aigateway-console
127.0.0.1:8081   aigateway-portal
127.0.0.1:3000   aigateway-console-grafana
127.0.0.1:9090   aigateway-console-prometheus
127.0.0.1:3100   aigateway-console-loki
127.0.0.1:5432   aigateway-core-postgresql-pgpool
127.0.0.1:6379   redis-stack-server
127.0.0.1:8888   aigateway-controller
127.0.0.1:18080  aigateway-plugin-server
```

推荐在联调前先跑一次：

```bash
python3 ./scripts/aigateway-dev.py check-connectivity
```

### 前端项目

#### 第一步、配置 Node 环境

注：建议 Node 版本选择长期稳定支持版本 16.18.1 及以上

#### 第二步、安装依赖

```bash
cd frontend && npm install
```

#### 第三步、本地启动

```bash
npm start
```

#### 第四步、打包

```bash
npm run build
#打包生成文件 frontend/build
```

### 后端项目

#### 第一步、配置 Go 环境

注：建议 Go 版本选择 1.23 及以上。

#### 第二步、说明当前目录结构

```bash
backend              # 新的 GoFrame 后端
backend-java-legacy  # 原 Java/Spring Boot 后端迁移参考基线
```

#### 第三步、编译 & 测试

```bash
cd backend && ./build.sh
```

#### 第四步、部署 & 启动

```bash
cd backend && ./start.sh
```

#### 第五步、访问

主页，默认 8080 端口

```html
http://localhost:8080/landing
```

可以通过以下方法开启 Swagger UI，并通过访问 Swagger 页面了解 API 情况。

**Swagger UI URL：**
```html
http://localhost:8080/swagger
```

## 开发规范

### 后端项目
