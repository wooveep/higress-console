---
title: AI Quota Management
keywords: [ AI Gateway, AI Quota ]
description: AI quota management plugin configuration reference
---
## Function Description
The `ai-quota` plugin supports two operating modes:

- `token`: legacy mode that checks and updates token quotas directly in Redis.
- `amount`: billing mode that checks wallet balance and model pricing before the request, then deducts real cost and emits usage events after the response.

The `ai-quota` plugin needs to work with authentication plugins such as `key-auth`, `jwt-auth`, etc., to obtain the consumer name associated with the authenticated identity. In `amount` mode it no longer relies on `ai-statistics` as the source of truth for billing.

Natural day/week/month windows in `amount` mode are always computed in `UTC`:

- daily windows reset at the next `00:00` in Shanghai
- weekly windows start on Monday `00:00`
- monthly windows start at the first day of the next month `00:00`

## Runtime Properties
Plugin execution phase: `default phase`
Plugin execution priority: `750`

## Configuration Description
| Name                 | Data Type        | Required Conditions                         | Default Value | Description                                       |
|---------------------|------------------|--------------------------------------------|---------------|---------------------------------------------------|
| `quota_unit`        | string           | Optional                                   | amount        | Quota mode, either `token` or `amount`            |
| `redis_key_prefix`  | string           | Optional                                   | chat_quota:   | Quota redis key prefix in `token` mode            |
| `balance_key_prefix` | string          | Optional                                   | billing:balance: | Wallet balance redis key prefix in `amount` mode |
| `price_key_prefix`  | string           | Optional                                   | billing:model-price: | Model price redis key prefix in `amount` mode |
| `usage_event_stream` | string          | Optional                                   | billing:usage:stream | Redis stream name for usage events in `amount` mode |
| `usage_event_dedup_prefix` | string    | Optional                                   | billing:usage:event: | Dedup key prefix for usage events in `amount` mode |
| `admin_consumer`    | string           | Required                                   |               | Consumer name for managing quota management identity |
| `admin_path`        | string           | Optional                                   |   /quota      | Prefix for the path to manage quota requests      |
| `redis`             | object           | Yes                                        |               | Redis related configuration                        |
Explanation of each configuration field in `redis`
| Configuration Item  | Type             | Required | Default Value                                            | Explanation                                   |
|---------------------|------------------|----------|---------------------------------------------------------|-----------------------------------------------|
| service_name        | string           | Required | -                                                       | Redis service name, full FQDN name with service type, e.g., my-redis.dns, redis.my-ns.svc.cluster.local |
| service_port        | int              | No       | Default value for static service is 80; others are 6379 | Service port for the redis service            |
| username            | string           | No       | -                                                       | Redis username                                |
| password            | string           | No       | -                                                       | Redis password                                |
| timeout             | int              | No       | 1000                                                    | Redis connection timeout in milliseconds      |

## Configuration Example
### Amount mode
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

### Detailed Billing Semantics In `amount` Mode

`amount` mode reads the Redis hash at `price_key_prefix + <provider>/<model>`. The Portal control plane materializes this price snapshot in advance, so all `ModelPriceData` fallback rules are already resolved before runtime charging begins.

The usage event and ledger pipeline now persist:

- `input_tokens` / `output_tokens`
- `cache_creation_input_tokens` / `cache_creation_5m_input_tokens` / `cache_creation_1h_input_tokens`
- `cache_read_input_tokens`
- `input_image_tokens` / `output_image_tokens`
- `input_image_count` / `output_image_count`
- `request_count`

The final charge can therefore include per-token pricing, `input_cost_per_request`, per-image pricing, and `above_200k` tier pricing in one ledger event.

Daily, weekly, and monthly amount-window TTLs are derived from the Shanghai calendar instead of the host machine timezone.

### Refresh Quota / Balance
If the suffix of the current request URL matches the admin_path, for example, if the plugin is effective on the route example.com/v1/chat/completions, then the quota can be updated via:
curl https://example.com/v1/chat/completions/quota/refresh -H "Authorization: Bearer credential3" -d "consumer=consumer1&quota=10000"
In `token` mode, the value of `chat_quota:consumer1` in Redis will be refreshed to `10000`.

In `amount` mode, the admin API writes `balance_key_prefix + consumer`, which is usually projected from the control plane or backend billing service.

### Query Quota
To query the quota of a specific user, you can use: 
curl https://example.com/v1/chat/completions/quota?consumer=consumer1 -H "Authorization: Bearer credential3"
The response will return: {"quota": 10000, "consumer": "consumer1"}

### Increase or Decrease Quota / Balance
To increase or decrease the quota of a specific user, you can use:
curl https://example.com/v1/chat/completions/quota/delta -d "consumer=consumer1&value=100" -H "Authorization: Bearer credential3"
This will increase the value of the key `chat_quota:consumer1` in Redis by 100, and negative values can also be supported, thus subtracting the corresponding value.
