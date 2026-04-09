package com.alibaba.higress.console.model.portal;

import java.util.List;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ModelAssetCapabilities {

    private List<String> modalities;

    private List<String> features;

    private List<String> requestKinds;
}
