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
@Schema(description = "Portal department bill summary")
public class PortalDepartmentBillRecord {

    private String departmentId;
    private String departmentName;
    private String departmentPath;
    private Long requestCount;
    private Long totalTokens;
    private Double totalCost;
    private Long activeConsumers;
}
