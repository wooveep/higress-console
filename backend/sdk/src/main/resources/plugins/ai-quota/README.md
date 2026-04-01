---
title: AI 配额管理
keywords: [ AI网关, AI配额 ]
description: AI 配额管理插件配置参考
---

## 功能说明

`ai-quota` 插件支持两种工作模式：

- `token`：兼容旧模式，直接对 Redis 中的 token 配额做准入和扣减。
- `amount`：金额模式。请求开始时校验用户余额和模型价格，请求结束后按真实 token 用量计算金额扣减，并把消费事件写入 Redis Stream。

`ai-quota` 插件需要配合认证插件比如 `key-auth`、`jwt-auth` 等插件获取认证身份的 consumer 名称。在金额模式下不再依赖 `ai-statistics` 作为主账务来源。

`amount` 模式下的日/周/月自然窗口统一按 `UTC` 计算：

- 日窗口在上海时区的次日 `00:00` 重置
- 周窗口以周一 `00:00` 为起点
- 月窗口以下月 1 日 `00:00` 为起点

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`750`

## 配置说明

| 名称                 | 数据类型            | 填写要求                                 | 默认值 | 描述                                         |
|--------------------|-----------------|--------------------------------------| ---- |--------------------------------------------|
| `quota_unit`      | string          | 选填                                   | amount | 配额模式，支持 `token` 或 `amount` |
| `redis_key_prefix` | string          |  选填                                     |   chat_quota:   | token 模式下的 quota redis key 前缀                         |
| `balance_key_prefix` | string       | 选填                                   | billing:balance: | amount 模式下的余额 redis key 前缀 |
| `price_key_prefix` | string         | 选填                                   | billing:model-price: | amount 模式下的模型价格 redis key 前缀 |
| `usage_event_stream` | string       | 选填                                   | billing:usage:stream | amount 模式下消费事件 Redis Stream 名称 |
| `usage_event_dedup_prefix` | string | 选填                                   | billing:usage:event: | amount 模式下消费事件去重 key 前缀 |
| `admin_consumer`   | string          | 必填                                   |      | 管理 quota 管理身份的 consumer 名称                 |
| `admin_path`       | string          | 选填                                   |   /quota   | 管理 quota 请求 path 前缀                        |
| `redis`            | object          | 是                                    |      | redis相关配置                                  |

`redis`中每一项的配置字段说明

| 配置项       | 类型   | 必填 | 默认值                                                     | 说明                        |
| ------------ | ------ | ---- | ---------------------------------------------------------- | --------------------------- |
| service_name | string | 必填 | -                                                          | redis 服务名称，带服务类型的完整 FQDN 名称，例如 my-redis.dns、redis.my-ns.svc.cluster.local     |
| service_port | int    | 否   | 服务类型为固定地址（static service）默认值为80，其他为6379 | 输入redis服务的服务端口     |
| username     | string | 否   | -                                                          | redis用户名                 |
| password     | string | 否   | -                                                          | redis密码                   |
| timeout      | int    | 否   | 1000                                                       | redis连接超时时间，单位毫秒 |



## 配置示例

### 金额模式
```yaml
quota_unit: amount
balance_key_prefix: "billing:balance:"
price_key_prefix: "billing:model-price:"
usage_event_stream: "billing:usage:stream"
usage_event_dedup_prefix: "billing:usage:event:"
admin_consumer: consumer3
admin_path: /quota
redis:
  service_name: redis-service.default.svc.cluster.local
  service_port: 6379
  timeout: 2000
```

### 金额模式计费口径

`amount` 模式读取 `price_key_prefix + <provider>/<model>` 对应的 Redis Hash。该价格快照由 Portal 提前物化，已包含 `ModelPriceData` 的回退结果，运行态直接按生效价格计费。

会被写入消费事件并进入 Portal 账本聚合的 usage 字段包括：

- `input_tokens` / `output_tokens`
- `cache_creation_input_tokens` / `cache_creation_5m_input_tokens` / `cache_creation_1h_input_tokens`
- `cache_read_input_tokens`
- `input_image_tokens` / `output_image_tokens`
- `input_image_count` / `output_image_count`
- `request_count`

最终金额会同时覆盖 token 单价、`input_cost_per_request`、图像按张费用以及 `above_200k` 分级价格。

金额模式里的日/周/月额度窗口 TTL 也按 `UTC` 推导，而不是依赖部署机器的本地时区。


###  刷新 quota / balance

如果当前请求 url 的后缀符合 admin_path，例如插件在 example.com/v1/chat/completions 这个路由上生效，那么更新 quota 可以通过
curl https://example.com/v1/chat/completions/quota/refresh -H "Authorization: Bearer credential3" -d "consumer=consumer1&quota=10000" 

在 `token` 模式下，Redis 中 key 为 `chat_quota:consumer1` 的值会被刷新为 `10000`。

在 `amount` 模式下，管理接口会直接写 `balance_key_prefix + consumer`，通常由控制面或后端投影器统一维护。

### 查询 quota

查询特定用户的 quota 可以通过 curl https://example.com/v1/chat/completions/quota?consumer=consumer1 -H "Authorization: Bearer credential3"
将返回： {"quota": 10000, "consumer": "consumer1"}

### 增减 quota / balance

增减特定用户的 quota 可以通过 curl https://example.com/v1/chat/completions/quota/delta -d "consumer=consumer1&value=100" -H "Authorization: Bearer credential3"
这样 Redis 中 Key 为 chat_quota:consumer1 的值就会增加100，可以支持负数，则减去对应值。
