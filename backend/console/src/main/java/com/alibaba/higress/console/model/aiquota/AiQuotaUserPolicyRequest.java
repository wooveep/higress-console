package com.alibaba.higress.console.model.aiquota;

import io.swagger.v3.oas.annotations.media.Schema;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@NoArgsConstructor
@AllArgsConstructor
@Schema(description = "AI quota user-level policy request")
public class AiQuotaUserPolicyRequest {

    @Schema(description = "Total cost limit in micro_yuan")
    private Long limitTotal;

    @Schema(description = "5-hour rolling cost limit in micro_yuan")
    private Long limit5h;

    @Schema(description = "Daily cost limit in micro_yuan")
    private Long limitDaily;

    @Schema(description = "Daily reset mode")
    private String dailyResetMode;

    @Schema(description = "Daily reset time, such as 00:00")
    private String dailyResetTime;

    @Schema(description = "Weekly cost limit in micro_yuan")
    private Long limitWeekly;

    @Schema(description = "Monthly cost limit in micro_yuan")
    private Long limitMonthly;

    @Schema(description = "Soft reset start time in RFC3339 format or yyyy-MM-dd'T'HH:mm interpreted in UTC")
    private String costResetAt;
}
