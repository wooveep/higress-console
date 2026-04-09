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
public class AgentCatalogRecord {

    private String agentId;

    private String canonicalName;

    private String displayName;

    private String intro;

    private String description;

    private String iconUrl;

    private List<String> tags;

    private String mcpServerName;

    private String status;

    private Integer toolCount;

    private List<String> transportTypes;

    private String resourceSummary;

    private String promptSummary;

    private String publishedAt;

    private String unpublishedAt;

    private String createdAt;

    private String updatedAt;
}
