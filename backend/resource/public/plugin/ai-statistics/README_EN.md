---
title: AI Statistics
keywords: [higress, AI, observability]
description: AI Statistics plugin configuration reference
---

## Introduction

Provides AI observability for request/response usage, token details, cache metrics, image metrics, logs, and tracing. This snapshot matches the Higress `2.2.1` semantics consumed by console builtin metadata APIs.

## Runtime Properties

Plugin Phase: `Stats Phase`
Plugin Priority: `900`

Recommended order: run after `model-router` and before response-masking plugins so routing is settled before request behavior is recorded.

## Configuration

| Name | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `attributes` | `[]Attribute` | optional | - | Additional fields to record in logs and spans |
| `disable_openai_usage` | `bool` | optional | `false` | Disable default OpenAI usage extraction for non-compatible protocols |
| `value_length_limit` | `int` | optional | `32000` | Maximum length for each extracted value |
| `enable_path_suffixes` | `[]string` | optional | common AI API paths | Only process requests with these path suffixes, or use `*` |
| `enable_content_types` | `[]string` | optional | `["text/event-stream","application/json"]` | Only buffer response bodies for these content types |
| `session_id_header` | `string` | optional | auto-detect | Custom request header used to read session ID |

### Built-in Attributes

The plugin can record these built-in keys without `value_source` / `value`:

- `question`
- `answer`
- `tool_calls`
- `reasoning`
- `reasoning_tokens`
- `cached_tokens`
- `input_token_details`
- `output_token_details`

### Detailed Usage Normalization

The plugin normalizes Anthropic, OpenAI, Gemini, and other upstream usage formats into one internal schema and writes the result into metrics, `ai_log`, and spans. Added detail fields include:

- `cache_creation_input_tokens`
- `cache_creation_5m_input_tokens`
- `cache_creation_1h_input_tokens`
- `cache_read_input_tokens`
- `input_image_tokens`
- `output_image_tokens`
- `input_image_count`
- `output_image_count`
- `request_count`
- `cache_ttl`

## Example

```yaml
value_length_limit: 32000
attributes:
  - key: consumer
    value_source: request_header
    value: x-mse-consumer
    apply_to_log: true
  - key: reasoning_tokens
    apply_to_log: true
  - key: cached_tokens
    apply_to_log: true
```

If a field is emitted with `as_separate_log_field`, add it to the gateway access log format explicitly:

```yaml
'{"consumer":"%FILTER_STATE(wasm.consumer:PLAIN)%"}'
```
