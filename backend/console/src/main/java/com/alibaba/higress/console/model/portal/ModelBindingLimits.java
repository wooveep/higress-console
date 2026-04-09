package com.alibaba.higress.console.model.portal;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ModelBindingLimits {

    private Long rpm;

    private Long tpm;

    private Long contextWindow;
}
