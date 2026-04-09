package com.alibaba.higress.console.model.portal;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * @author Codex
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AssetGrantRecord {

    private String assetType;
    private String assetId;
    private String subjectType;
    private String subjectId;
}
