package com.alibaba.higress.console.model.portal;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ModelBindingPriceVersionRecord {

    private Long versionId;

    private String modelId;

    private String currency;

    private String status;

    private Boolean active;

    private String effectiveFrom;

    private String effectiveTo;

    private String createdAt;

    private String updatedAt;

    private ModelBindingPricing pricing;
}
