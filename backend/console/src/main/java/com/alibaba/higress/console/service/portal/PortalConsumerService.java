package com.alibaba.higress.console.service.portal;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import javax.annotation.Resource;

import org.apache.commons.collections4.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.PortalPasswordResetResult;
import com.alibaba.higress.console.model.portal.PortalUserRecord;
import com.alibaba.higress.sdk.exception.ValidationException;
import com.alibaba.higress.sdk.service.consumer.ConsumerService;
import com.alibaba.higress.sdk.model.CommonPageQuery;
import com.alibaba.higress.sdk.model.PaginatedResult;
import com.alibaba.higress.sdk.model.consumer.Consumer;
import com.alibaba.higress.sdk.model.consumer.CredentialType;
import com.alibaba.higress.sdk.model.consumer.KeyAuthCredential;
import com.alibaba.higress.sdk.model.consumer.KeyAuthCredentialSource;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalConsumerService {

    private static final String STATUS_ACTIVE = "active";
    private static final String STATUS_DISABLED = "disabled";
    private static final String STATUS_PENDING = "pending";
    private static final String USER_LEVEL_NORMAL = "normal";
    private static final String USER_LEVEL_PLUS = "plus";
    private static final String USER_LEVEL_PRO = "pro";
    private static final String USER_LEVEL_ULTRA = "ultra";
    private static final java.util.List<String> SUPPORTED_USER_LEVELS =
        java.util.Arrays.asList(USER_LEVEL_NORMAL, USER_LEVEL_PLUS, USER_LEVEL_PRO, USER_LEVEL_ULTRA);

    private PortalUserJdbcService portalUserJdbcService;
    private PortalConsumerProjectionService portalConsumerProjectionService;
    private PortalConsumerLevelAuthReconcileService portalConsumerLevelAuthReconcileService;
    private ConsumerService consumerService;
    private PortalOrganizationJdbcService portalOrganizationJdbcService;

    @Resource
    public void setPortalUserJdbcService(PortalUserJdbcService portalUserJdbcService) {
        this.portalUserJdbcService = portalUserJdbcService;
    }

    @Resource
    public void setPortalConsumerProjectionService(PortalConsumerProjectionService portalConsumerProjectionService) {
        this.portalConsumerProjectionService = portalConsumerProjectionService;
    }

    @Resource
    public void setPortalConsumerLevelAuthReconcileService(
        PortalConsumerLevelAuthReconcileService portalConsumerLevelAuthReconcileService) {
        this.portalConsumerLevelAuthReconcileService = portalConsumerLevelAuthReconcileService;
    }

    @Resource
    public void setConsumerService(ConsumerService consumerService) {
        this.consumerService = consumerService;
    }

    @Resource
    public void setPortalOrganizationJdbcService(PortalOrganizationJdbcService portalOrganizationJdbcService) {
        this.portalOrganizationJdbcService = portalOrganizationJdbcService;
    }

    public PaginatedResult<Consumer> list(CommonPageQuery query) {
        ensurePortalDatabaseEnabled();
        List<PortalUserRecord> users = portalUserJdbcService.listAllUsers();
        if (CollectionUtils.isEmpty(users)) {
            return PaginatedResult.createFromFullList(Collections.emptyList(), query);
        }
        List<String> names = new ArrayList<>(users.size());
        for (PortalUserRecord user : users) {
            if (user != null && StringUtils.isNotBlank(user.getConsumerName())) {
                names.add(user.getConsumerName());
            }
        }
        java.util.Map<String, List<String>> activeRawKeys = portalUserJdbcService.listActiveRawKeysByConsumerNames(names);
        List<Consumer> result = new ArrayList<>(users.size());
        for (PortalUserRecord user : users) {
            Consumer consumer = toApiConsumer(user, activeRawKeys.getOrDefault(
                user == null ? null : user.getConsumerName(), Collections.emptyList()));
            if (consumer != null) {
                result.add(consumer);
            }
        }
        result.sort(java.util.Comparator.comparing(c -> StringUtils.defaultString(c.getName())));
        return PaginatedResult.createFromFullList(result, query);
    }

    public Consumer query(String consumerName) {
        ensurePortalDatabaseEnabled();
        PortalUserRecord record = portalUserJdbcService.queryByConsumerName(consumerName);
        if (record == null) {
            return null;
        }
        return toApiConsumer(record, portalUserJdbcService.listActiveRawKeys(record.getConsumerName()));
    }

    public Consumer addOrUpdate(Consumer request, boolean forCreate) {
        ensurePortalDatabaseEnabled();
        if (request == null || StringUtils.isBlank(request.getName())) {
            throw new ValidationException("name cannot be blank.");
        }
        ensureNotBuiltinAdministrator(request.getName(),
            forCreate ? "created or edited from Console" : "edited from Console");
        validatePortalUserLevel(request.getPortalUserLevel());
        request.setPortalUserLevel(normalizePortalUserLevel(request.getPortalUserLevel()));
        if (CollectionUtils.isNotEmpty(request.getCredentials())) {
            log.warn("Deprecated field credentials is ignored for /v1/consumers {}. consumer={}",
                forCreate ? "POST" : "PUT", request.getName());
        }
        PortalUserRecord record = portalUserJdbcService.upsertFromConsumer(request, "console");
        if (record == null) {
            throw new IllegalStateException("Failed to upsert portal user.");
        }
        portalConsumerProjectionService.syncNow();
        portalConsumerLevelAuthReconcileService.reconcileNow("consumer-upsert");
        Consumer consumer = query(record.getConsumerName());
        if (consumer != null && forCreate && StringUtils.isNotBlank(record.getTempPassword())) {
            consumer.setPortalTempPassword(record.getTempPassword());
        }
        return consumer;
    }

    public Consumer updateStatus(String consumerName, String status) {
        ensurePortalDatabaseEnabled();
        validateStatus(status);
        ensureNotBuiltinAdministrator(consumerName, "enabled or disabled");
        PortalUserRecord record = portalUserJdbcService.queryByConsumerName(consumerName);
        if (record == null) {
            throw new ValidationException("Consumer not found: " + consumerName);
        }
        if (STATUS_DISABLED.equals(status)) {
            portalUserJdbcService.disableAllApiKeys(consumerName);
        }
        portalUserJdbcService.updateStatus(consumerName, status);
        portalConsumerProjectionService.syncNow();
        portalConsumerLevelAuthReconcileService.reconcileNow("consumer-status");
        return query(consumerName);
    }

    public void softDelete(String consumerName) {
        delete(consumerName);
    }

    public void delete(String consumerName) {
        ensurePortalDatabaseEnabled();
        if (StringUtils.isBlank(consumerName)) {
            throw new ValidationException("consumerName cannot be blank.");
        }
        ensureNotBuiltinAdministrator(consumerName, "deleted");
        PortalUserRecord record = portalUserJdbcService.queryByConsumerName(consumerName);
        if (record == null) {
            throw new ValidationException("Consumer not found: " + consumerName);
        }
        if (portalOrganizationJdbcService != null && portalOrganizationJdbcService.isDepartmentAdministrator(consumerName)) {
            throw new ValidationException("Department administrator cannot be deleted. Please reassign the department administrator first.");
        }
        if (consumerService != null) {
            consumerService.delete(consumerName);
        }
        portalUserJdbcService.logicalDelete(consumerName);
        portalConsumerProjectionService.syncNow();
        portalConsumerLevelAuthReconcileService.reconcileNow("consumer-delete");
    }

    public List<String> listDepartments() {
        ensurePortalDatabaseEnabled();
        return Collections.emptyList();
    }

    public void addDepartmentCompat(String departmentName) {
        if (StringUtils.isBlank(departmentName)) {
            throw new ValidationException("Department name cannot be blank.");
        }
        log.warn("Deprecated endpoint POST /v1/consumers/departments called. department={}", departmentName);
    }

    public PortalPasswordResetResult resetPassword(String consumerName) {
        ensurePortalDatabaseEnabled();
        ensureNotBuiltinAdministrator(consumerName, "reset password");
        return portalUserJdbcService.resetPassword(consumerName);
    }

    private Consumer toApiConsumer(PortalUserRecord record, List<String> rawKeys) {
        if (record == null || StringUtils.isBlank(record.getConsumerName())) {
            return null;
        }
        Consumer consumer = Consumer.builder().name(record.getConsumerName()).credentials(buildMaskedCredentials(rawKeys))
            .build();
        consumer.setPortalStatus(record.getStatus());
        consumer.setPortalDisplayName(record.getDisplayName());
        consumer.setPortalEmail(record.getEmail());
        consumer.setPortalUserSource(record.getSource());
        consumer.setPortalUserLevel(normalizePortalUserLevel(record.getUserLevel()));
        return consumer;
    }

    private List<com.alibaba.higress.sdk.model.consumer.Credential> buildMaskedCredentials(List<String> rawKeys) {
        if (CollectionUtils.isEmpty(rawKeys)) {
            return Collections.emptyList();
        }
        List<String> maskedKeys = new ArrayList<>(rawKeys.size());
        for (String rawKey : rawKeys) {
            String normalized = StringUtils.trimToNull(rawKey);
            if (normalized != null) {
                maskedKeys.add(maskKey(normalized));
            }
        }
        if (maskedKeys.isEmpty()) {
            return Collections.emptyList();
        }
        KeyAuthCredential credential = new KeyAuthCredential();
        credential.setType(CredentialType.KEY_AUTH);
        credential.setSource(KeyAuthCredentialSource.BEARER.name());
        credential.setValues(maskedKeys);
        return Collections.singletonList(credential);
    }

    private String maskKey(String raw) {
        if (raw.length() <= 8) {
            return "****";
        }
        return raw.substring(0, 4) + "****" + raw.substring(raw.length() - 4);
    }

    private void validateStatus(String status) {
        String normalized = StringUtils.lowerCase(StringUtils.trimToEmpty(status));
        if (!STATUS_ACTIVE.equals(normalized) && !STATUS_DISABLED.equals(normalized) && !STATUS_PENDING.equals(normalized)) {
            throw new ValidationException("status must be active/disabled/pending.");
        }
    }

    private void validatePortalUserLevel(String userLevel) {
        if (StringUtils.isBlank(userLevel)) {
            return;
        }
        String normalized = StringUtils.lowerCase(StringUtils.trimToEmpty(userLevel));
        if (!SUPPORTED_USER_LEVELS.contains(normalized)) {
            throw new ValidationException("portalUserLevel must be one of normal/plus/pro/ultra.");
        }
    }

    private String normalizePortalUserLevel(String userLevel) {
        String normalized = StringUtils.lowerCase(StringUtils.trimToEmpty(userLevel));
        if (SUPPORTED_USER_LEVELS.contains(normalized)) {
            return normalized;
        }
        return USER_LEVEL_NORMAL;
    }

    private void ensurePortalDatabaseEnabled() {
        if (!portalUserJdbcService.enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
    }

    private void ensureNotBuiltinAdministrator(String consumerName, String action) {
        if (portalUserJdbcService.isBuiltinAdministrator(consumerName)) {
            throw new ValidationException("Built-in administrator cannot be " + action + ".");
        }
    }
}
