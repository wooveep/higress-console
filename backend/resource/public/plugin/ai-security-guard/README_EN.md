---
title: AI Content Security
keywords: [higress, AI, security]
description: Alibaba Cloud content security
---

## Introduction

Integrate with Aliyun content security service for detections of LLM input and output, ensuring that application content is legal and compliant.

## Runtime Properties

Plugin Phase: `CUSTOM`
Plugin Priority: `300`

## Configuration

| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| `serviceName` | string | required | - | Service name |
| `servicePort` | string | required | - | Service port |
| `serviceHost` | string | required | - | Host of Aliyun content security service endpoint |
| `accessKey` | string | required | - | Aliyun access key |
| `secretKey` | string | required | - | Aliyun secret key |
| `action` | string | required | `MultiModalGuard` | Guardrails business action |
| `checkRequest` | bool | optional | false | Check if the input is legal |
| `checkResponse` | bool | optional | false | Check if the output is legal |
| `requestCheckService` | string | optional | `llm_query_moderation` | Aliyun input moderation service |
| `responseCheckService` | string | optional | `llm_response_moderation` | Aliyun output moderation service |
| `requestContentJsonPath` | string | optional | `messages.@reverse.0.content` | JSONPath to extract request content |
| `responseContentJsonPath` | string | optional | `choices.0.message.content` | JSONPath to extract response content |
| `responseStreamContentJsonPath` | string | optional | `choices.0.delta.content` | JSONPath to extract streaming response content |
| `denyCode` | int | optional | 200 | HTTP status code used when content is blocked |
| `denyMessage` | string | optional | structured guard payload | Response content when the specified content is illegal |
| `protocol` | string | optional | `openai` | `openai` or `original` |
| `contentModerationLevelBar` | string | optional | `max` | Content moderation threshold |
| `promptAttackLevelBar` | string | optional | `max` | Prompt attack threshold |
| `sensitiveDataLevelBar` | string | optional | `S4` | Sensitive data threshold |
| `timeout` | int | optional | 2000 | Timeout for the guard service |
| `bufferLimit` | int | optional | 1000 | Length limit for each moderation request |

### Deny Response Body

When content is blocked, the plugin returns a structured JSON payload:

```json
{
  "blockedDetails": [
    {
      "Type": "contentModeration",
      "Level": "high",
      "Suggestion": "block"
    }
  ],
  "requestId": "AAAAAA-BBBB-CCCC-DDDD-EEEEEEE****",
  "guardCode": 200
}
```

Field descriptions:

| Field | Type | Description |
| --- | --- | --- |
| `blockedDetails` | array | Details of the triggered blocking dimensions |
| `blockedDetails[].Type` | string | Risk type such as `contentModeration`, `promptAttack`, `sensitiveData` |
| `blockedDetails[].Level` | string | Risk level such as `high`, `medium`, `low` |
| `blockedDetails[].Suggestion` | string | Suggested action, usually `block` |
| `requestId` | string | Request ID from the security service |
| `guardCode` | int | Business code from the security service |

Protocol embedding notes:

- `text_generation` non-streaming: serialized into `choices[0].message.content`
- `text_generation` streaming: serialized into the first chunk `delta.content`
- `protocol=original`: returned directly as the JSON response body
- `mcp`: serialized into `error.message`

## Example

```yaml
serviceName: safecheck.dns
servicePort: "443"
serviceHost: green-cip.cn-shanghai.aliyuncs.com
accessKey: XXXXX
secretKey: XXXXX
action: MultiModalGuard
checkRequest: true
checkResponse: true
```
