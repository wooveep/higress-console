package com.alibaba.higress.console.model.portal;

import java.time.LocalDateTime;

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
public class OrgAccountRecord {

    private String consumerName;
    private String displayName;
    private String email;
    private String status;
    private String userLevel;
    private String source;
    private String departmentId;
    private String departmentName;
    private String departmentPath;
    private String parentConsumerName;
    private Boolean isDepartmentAdmin;
    private LocalDateTime lastLoginAt;
    private String tempPassword;
}
