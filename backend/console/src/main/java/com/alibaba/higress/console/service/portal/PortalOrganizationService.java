package com.alibaba.higress.console.service.portal;

import java.util.List;

import javax.annotation.Resource;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.AssetGrantRecord;
import com.alibaba.higress.console.model.portal.OrgAccountRecord;
import com.alibaba.higress.console.model.portal.OrgDepartmentNode;
import com.alibaba.higress.console.model.portal.PortalUserRecord;
import com.alibaba.higress.sdk.exception.ValidationException;
import com.alibaba.higress.sdk.model.consumer.Consumer;

/**
 * @author Codex
 */
@Service
public class PortalOrganizationService {

    private static final String STATUS_DISABLED = "disabled";

    @Resource
    private PortalUserJdbcService portalUserJdbcService;

    @Resource
    private PortalOrganizationJdbcService portalOrganizationJdbcService;

    @Resource
    private PortalAssetGrantJdbcService portalAssetGrantJdbcService;

    @Resource
    private AuthorizationSubjectResolver authorizationSubjectResolver;

    @Resource
    private PortalConsumerProjectionService portalConsumerProjectionService;

    @Resource
    private PortalConsumerLevelAuthReconcileService portalConsumerLevelAuthReconcileService;

    @Resource
    private PortalAgentCatalogJdbcService portalAgentCatalogJdbcService;

    @Value("${higress.portal.org.default-password:}")
    private String defaultOrgPassword;

    public List<OrgDepartmentNode> listDepartmentTree() {
        ensureEnabled();
        return portalOrganizationJdbcService.listDepartmentTree();
    }

    public OrgDepartmentNode createDepartment(DepartmentMutation mutation) {
        ensureEnabled();
        if (mutation == null) {
            throw new ValidationException("Department request cannot be null.");
        }
        OrgDepartmentNode node = portalOrganizationJdbcService.createDepartment(mutation.getName(),
            mutation.getParentDepartmentId(),
            PortalOrganizationJdbcService.DepartmentAdminMutation.builder()
                .consumerName(mutation.getAdminConsumerName())
                .displayName(mutation.getAdminDisplayName())
                .email(mutation.getAdminEmail())
                .userLevel(mutation.getAdminUserLevel())
                .password(mutation.getAdminPassword())
                .build(),
            defaultOrgPassword);
        syncProjectionAndAuth("org-department-create");
        return node;
    }

    public OrgDepartmentNode updateDepartment(String departmentId, DepartmentMutation mutation) {
        ensureEnabled();
        if (mutation == null) {
            throw new ValidationException("Department request cannot be null.");
        }
        OrgDepartmentNode node = portalOrganizationJdbcService.updateDepartment(departmentId, mutation.getName(),
            mutation.getAdminConsumerName());
        syncProjectionAndAuth("org-department-update");
        return node;
    }

    public OrgDepartmentNode moveDepartment(String departmentId, String parentDepartmentId) {
        ensureEnabled();
        OrgDepartmentNode node = portalOrganizationJdbcService.moveDepartment(departmentId, parentDepartmentId);
        syncProjectionAndAuth("org-department-move");
        return node;
    }

    public void deleteDepartment(String departmentId) {
        ensureEnabled();
        portalOrganizationJdbcService.deleteDepartment(departmentId);
        syncProjectionAndAuth("org-department-delete");
    }

    public List<OrgAccountRecord> listAccounts() {
        ensureEnabled();
        return portalOrganizationJdbcService.listAccounts();
    }

    public OrgAccountRecord createAccount(AccountMutation mutation) {
        ensureEnabled();
        String consumerName = requireConsumerName(mutation == null ? null : mutation.getConsumerName());
        ensureMutableConsumer(consumerName);
        PortalUserRecord created = upsertPortalUser(buildConsumerPayload(consumerName, mutation), true);
        OrgAccountRecord record = portalOrganizationJdbcService.assignAccount(consumerName,
            mutation == null ? null : mutation.getDepartmentId(),
            mutation == null ? null : mutation.getParentConsumerName());
        if (record != null && created != null) {
            record.setTempPassword(created.getTempPassword());
        }
        syncProjectionAndAuth("org-account-create");
        return record;
    }

    public OrgAccountRecord updateAccount(String consumerName, AccountMutation mutation) {
        ensureEnabled();
        String normalizedConsumerName = requireConsumerName(consumerName);
        ensureMutableConsumer(normalizedConsumerName);
        PortalUserRecord existed = portalUserJdbcService.queryByConsumerName(normalizedConsumerName);
        if (existed == null) {
            throw new ValidationException("Consumer not found: " + normalizedConsumerName);
        }
        if (requiresPortalUserUpdate(existed, mutation)) {
            upsertPortalUser(buildConsumerPayload(normalizedConsumerName, mutation), false);
        }
        OrgAccountRecord record = portalOrganizationJdbcService.assignAccount(normalizedConsumerName,
            mutation == null ? null : mutation.getDepartmentId(),
            mutation == null ? null : mutation.getParentConsumerName());
        syncProjectionAndAuth("org-account-update");
        return record;
    }

    public OrgAccountRecord updateAccountAssignment(String consumerName, String departmentId, String parentConsumerName) {
        ensureEnabled();
        String normalizedConsumerName = requireConsumerName(consumerName);
        ensureMutableConsumer(normalizedConsumerName);
        OrgAccountRecord record = portalOrganizationJdbcService.assignAccount(normalizedConsumerName, departmentId,
            parentConsumerName);
        syncProjectionAndAuth("org-account-assignment");
        return record;
    }

    public OrgAccountRecord updateAccountStatus(String consumerName, String status) {
        ensureEnabled();
        String normalizedConsumerName = requireConsumerName(consumerName);
        ensureMutableConsumer(normalizedConsumerName);
        String normalizedStatus = normalizeStatus(status);
        if (STATUS_DISABLED.equals(normalizedStatus)) {
            portalUserJdbcService.disableAllApiKeys(normalizedConsumerName);
        }
        portalUserJdbcService.updateStatus(normalizedConsumerName, normalizedStatus);
        syncProjectionAndAuth("org-account-status");
        return portalOrganizationJdbcService.queryAccount(normalizedConsumerName);
    }

    public List<AssetGrantRecord> listGrants(String assetType, String assetId) {
        ensureEnabled();
        return portalAssetGrantJdbcService.listGrants(assetType, assetId);
    }

    public List<AssetGrantRecord> replaceGrants(String assetType, String assetId, List<AssetGrantRecord> grants) {
        ensureEnabled();
        List<AssetGrantRecord> result = portalAssetGrantJdbcService.replaceGrants(assetType, assetId, grants);
        if (StringUtils.equalsIgnoreCase(StringUtils.trimToEmpty(assetType), "agent_catalog")
            && StringUtils.isNotBlank(assetId)) {
            portalAgentCatalogJdbcService.syncPublishedAgentMcpAuthorization(assetId);
        }
        return result;
    }

    public List<String> resolveGrantedConsumers(String assetType, String assetId) {
        ensureEnabled();
        return authorizationSubjectResolver.resolveConsumers(assetType, assetId);
    }

    private Consumer buildConsumerPayload(String consumerName, AccountMutation mutation) {
        Consumer consumer = new Consumer();
        consumer.setName(consumerName);
        consumer.setPortalDisplayName(mutation == null ? null : mutation.getDisplayName());
        consumer.setPortalEmail(mutation == null ? null : mutation.getEmail());
        consumer.setPortalUserLevel(mutation == null ? null : mutation.getUserLevel());
        consumer.setPortalPassword(mutation == null ? null : mutation.getPassword());
        consumer.setPortalStatus(mutation == null ? null : mutation.getStatus());
        consumer.setCredentials(java.util.Collections.emptyList());
        return consumer;
    }

    private PortalUserRecord upsertPortalUser(Consumer consumer, boolean forCreate) {
        if (consumer == null || StringUtils.isBlank(consumer.getName())) {
            throw new ValidationException("consumerName cannot be blank.");
        }
        if (forCreate && portalUserJdbcService.queryByConsumerName(consumer.getName()) != null) {
            throw new ValidationException("Consumer already exists: " + consumer.getName());
        }
        if (!forCreate && portalUserJdbcService.queryByConsumerName(consumer.getName()) == null) {
            throw new ValidationException("Consumer not found: " + consumer.getName());
        }
        if (StringUtils.isNotBlank(consumer.getPortalStatus())) {
            normalizeStatus(consumer.getPortalStatus());
        }
        PortalUserRecord record = portalUserJdbcService.upsertFromConsumer(consumer, "console");
        if (record == null) {
            throw new IllegalStateException("Failed to upsert portal user.");
        }
        return record;
    }

    private boolean requiresPortalUserUpdate(PortalUserRecord existed, AccountMutation mutation) {
        if (existed == null) {
            return true;
        }
        if (mutation == null) {
            return false;
        }
        if (StringUtils.isNotBlank(StringUtils.trimToNull(mutation.getPassword()))) {
            return true;
        }
        if (isChanged(mutation.getDisplayName(), existed.getDisplayName())) {
            return true;
        }
        if (isChanged(mutation.getEmail(), existed.getEmail())) {
            return true;
        }
        if (StringUtils.isNotBlank(mutation.getUserLevel())
            && !StringUtils.equalsIgnoreCase(normalizeUserLevel(mutation.getUserLevel()), existed.getUserLevel())) {
            return true;
        }
        return StringUtils.isNotBlank(mutation.getStatus())
            && !StringUtils.equalsIgnoreCase(normalizeStatus(mutation.getStatus()), existed.getStatus());
    }

    private boolean isChanged(String requestedValue, String existedValue) {
        String normalizedRequested = StringUtils.trimToNull(requestedValue);
        if (normalizedRequested == null) {
            return false;
        }
        return !StringUtils.equals(normalizedRequested, StringUtils.trimToEmpty(existedValue));
    }

    private void syncProjectionAndAuth(String trigger) {
        portalConsumerProjectionService.syncNow();
        portalConsumerLevelAuthReconcileService.reconcileNow(trigger);
    }

    private String normalizeStatus(String status) {
        String normalized = StringUtils.lowerCase(StringUtils.trimToEmpty(status));
        if (!StringUtils.equals(normalized, "active") && !StringUtils.equals(normalized, "disabled")
            && !StringUtils.equals(normalized, "pending")) {
            throw new ValidationException("status must be active/disabled/pending.");
        }
        return normalized;
    }

    private String normalizeUserLevel(String userLevel) {
        String normalized = StringUtils.lowerCase(StringUtils.trimToEmpty(userLevel));
        if (StringUtils.equals(normalized, "plus") || StringUtils.equals(normalized, "pro")
            || StringUtils.equals(normalized, "ultra")) {
            return normalized;
        }
        return "normal";
    }

    private String requireConsumerName(String consumerName) {
        String normalized = StringUtils.trimToNull(consumerName);
        if (normalized == null) {
            throw new ValidationException("consumerName cannot be blank.");
        }
        return normalized;
    }

    private void ensureMutableConsumer(String consumerName) {
        if (portalUserJdbcService.isBuiltinAdministrator(consumerName)) {
            throw new ValidationException("Built-in administrator cannot be modified from organization APIs.");
        }
    }

    private void ensureEnabled() {
        if (!portalUserJdbcService.enabled() || !portalOrganizationJdbcService.enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
    }

    @lombok.Data
    @lombok.Builder
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class AccountMutation {

        private String consumerName;
        private String displayName;
        private String email;
        private String userLevel;
        private String password;
        private String status;
        private String departmentId;
        private String parentConsumerName;
    }

    @lombok.Data
    @lombok.Builder
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class DepartmentMutation {

        private String name;
        private String parentDepartmentId;
        private String adminConsumerName;
        private String adminDisplayName;
        private String adminEmail;
        private String adminUserLevel;
        private String adminPassword;
    }
}
