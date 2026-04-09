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
public class AgentCatalogOptionServerRecord {

    private String mcpServerName;

    private String description;

    private String type;

    private List<String> domains;

    private Boolean authEnabled;

    private String authType;
}
