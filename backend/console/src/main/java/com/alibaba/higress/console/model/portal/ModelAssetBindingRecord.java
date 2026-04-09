package com.alibaba.higress.console.model.portal;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ModelAssetBindingRecord {

    private String bindingId;

    private String assetId;

    private String modelId;

    private String providerName;

    private String targetModel;

    private String protocol;

    private String endpoint;

    private String status;

    private String publishedAt;

    private String unpublishedAt;

    private String createdAt;

    private String updatedAt;

    private ModelBindingPricing pricing;

    private ModelBindingLimits limits;
}
