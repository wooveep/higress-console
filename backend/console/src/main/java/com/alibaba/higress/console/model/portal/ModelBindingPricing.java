package com.alibaba.higress.console.model.portal;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ModelBindingPricing {

    private String currency;
    private Double inputCostPerToken;
    private Double outputCostPerToken;
    private Double inputCostPerRequest;
    private Double cacheCreationInputTokenCost;
    private Double cacheCreationInputTokenCostAbove1hr;
    private Double cacheReadInputTokenCost;
    private Double inputCostPerTokenAbove200kTokens;
    private Double outputCostPerTokenAbove200kTokens;
    private Double cacheCreationInputTokenCostAbove200kTokens;
    private Double cacheReadInputTokenCostAbove200kTokens;
    private Double outputCostPerImage;
    private Double outputCostPerImageToken;
    private Double inputCostPerImage;
    private Double inputCostPerImageToken;
    private Boolean supportsPromptCaching;
}
