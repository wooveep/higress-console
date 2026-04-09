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
public class ModelAssetRecord {

    private String assetId;

    private String canonicalName;

    private String displayName;

    private String intro;

    private String createdAt;

    private String updatedAt;

    private List<String> tags;

    private ModelAssetCapabilities capabilities;

    private List<ModelAssetBindingRecord> bindings;
}
