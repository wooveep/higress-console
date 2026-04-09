package com.alibaba.higress.console.model.portal;

import java.util.ArrayList;
import java.util.List;

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
public class OrgDepartmentNode {

    private String departmentId;
    private String name;
    private String parentDepartmentId;
    private String adminConsumerName;
    private String adminDisplayName;
    private Integer level;
    private Integer memberCount;

    @Builder.Default
    private List<OrgDepartmentNode> children = new ArrayList<>();
}
