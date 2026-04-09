package com.alibaba.higress.console.model.portal;

import io.swagger.v3.oas.annotations.media.Schema;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@Schema(description = "Portal usage event record")
public class PortalUsageEventRecord {

    private String eventId;
    private String requestId;
    private String traceId;
    private String consumerName;
    private String departmentId;
    private String departmentPath;
    private String apiKeyId;
    private String modelId;
    private Long priceVersionId;
    private String routeName;
    private String requestKind;
    private String requestStatus;
    private String usageStatus;
    private Integer httpStatus;
    private Long inputTokens;
    private Long outputTokens;
    private Long totalTokens;
    private Long requestCount;
    private Long costMicroYuan;
    private String occurredAt;
}
