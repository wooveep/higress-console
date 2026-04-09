package com.alibaba.higress.console.service.portal;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;

import javax.annotation.PostConstruct;
import javax.annotation.Resource;

import org.apache.commons.collections4.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.PortalUserRecord;
import com.alibaba.higress.sdk.model.CommonPageQuery;
import com.alibaba.higress.sdk.model.PaginatedResult;
import com.alibaba.higress.sdk.model.consumer.Consumer;
import com.alibaba.higress.sdk.model.consumer.Credential;
import com.alibaba.higress.sdk.model.consumer.CredentialType;
import com.alibaba.higress.sdk.model.consumer.KeyAuthCredential;
import com.alibaba.higress.sdk.model.consumer.KeyAuthCredentialSource;
import com.alibaba.higress.sdk.service.consumer.ConsumerService;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalConsumerProjectionService {

    private static final String STATUS_DISABLED = "disabled";

    private final String revokedSalt = UUID.randomUUID().toString().replace("-", "");

    private boolean migrationCompleted = false;

    private PortalUserJdbcService portalUserJdbcService;
    private ConsumerService consumerService;

    @Value("${higress.portal.consumer.sync.migration-enabled:true}")
    private boolean migrationEnabled;

    @Value("${higress.portal.consumer.sync.orphan-cleanup-enabled:false}")
    private boolean orphanCleanupEnabled;

    @Value("${higress.portal.consumer.projection.write-enabled:false}")
    private boolean projectionWriteEnabled;

    @Resource
    public void setPortalUserJdbcService(PortalUserJdbcService portalUserJdbcService) {
        this.portalUserJdbcService = portalUserJdbcService;
    }

    @Resource
    public void setConsumerService(ConsumerService consumerService) {
        this.consumerService = consumerService;
    }

    @PostConstruct
    public void init() {
        syncNow();
    }

    @Scheduled(initialDelayString = "${higress.portal.consumer.sync.initial-delay-millis:10000}",
        fixedDelayString = "${higress.portal.consumer.sync.interval-millis:30000}")
    public void scheduledSync() {
        syncNow();
    }

    public synchronized void syncNow() {
        if (!portalUserJdbcService.enabled()) {
            return;
        }
        if (!projectionWriteEnabled) {
            return;
        }
        portalUserJdbcService.ensureBuiltinAdministrator();
        if (!migrationCompleted && migrationEnabled) {
            backfillLegacyConsumers();
            migrationCompleted = true;
        }
        projectPortalUsersToKeyAuth();
    }

    private void backfillLegacyConsumers() {
        List<Consumer> legacyConsumers = listGatewayConsumers();
        if (CollectionUtils.isEmpty(legacyConsumers)) {
            return;
        }
        for (Consumer legacyConsumer : legacyConsumers) {
            String consumerName = legacyConsumer.getName();
            if (StringUtils.isBlank(consumerName)) {
                continue;
            }
            PortalUserRecord user = portalUserJdbcService.ensureMigrationUser(consumerName);
            if (user == null) {
                continue;
            }
            List<String> rawKeys = extractRawKeys(legacyConsumer);
            for (String rawKey : rawKeys) {
                portalUserJdbcService.upsertMigratedApiKey(user.getConsumerName(), rawKey);
            }
        }
    }

    private void projectPortalUsersToKeyAuth() {
        List<PortalUserRecord> users = portalUserJdbcService.listAllUsers();
        if (CollectionUtils.isEmpty(users)) {
            cleanupAllGatewayConsumersIfNeeded();
            return;
        }
        List<String> consumerNames = users.stream().map(PortalUserRecord::getConsumerName).filter(StringUtils::isNotBlank)
            .distinct().collect(Collectors.toList());
        Map<String, List<String>> activeRawKeyMap = portalUserJdbcService.listActiveRawKeysByConsumerNames(consumerNames);
        Set<String> managedNames = consumerNames.stream().filter(
            name -> !portalUserJdbcService.isBuiltinAdministrator(name)).collect(Collectors.toCollection(
            LinkedHashSet::new));
        Map<String, Consumer> desiredMap = users.stream().map(user -> buildDesiredConsumer(user, activeRawKeyMap)).filter(
            Objects::nonNull).collect(Collectors.toMap(Consumer::getName, c -> c, (left, right) -> left));

        Map<String, Consumer> existingMap = listGatewayConsumers().stream().filter(c -> StringUtils.isNotBlank(c.getName()))
            .collect(Collectors.toMap(Consumer::getName, c -> c, (left, right) -> left));

        for (Map.Entry<String, Consumer> entry : desiredMap.entrySet()) {
            String consumerName = entry.getKey();
            Consumer desired = entry.getValue();
            Consumer existed = existingMap.remove(consumerName);
            if (sameProjection(existed, desired)) {
                continue;
            }
            try {
                consumerService.addOrUpdate(desired);
            } catch (Exception ex) {
                log.warn("Failed to project portal consumer {} to key-auth.", consumerName, ex);
            }
        }

        for (Map.Entry<String, Consumer> entry : existingMap.entrySet()) {
            String consumerName = entry.getKey();
            if (portalUserJdbcService.isBuiltinAdministrator(consumerName)) {
                continue;
            }
            boolean managedAndShouldDelete = managedNames.contains(consumerName);
            if (!managedAndShouldDelete && !orphanCleanupEnabled) {
                continue;
            }
            try {
                consumerService.delete(consumerName);
            } catch (Exception ex) {
                log.warn("Failed to cleanup projected consumer {} from key-auth.", consumerName, ex);
            }
        }
    }

    private void cleanupAllGatewayConsumersIfNeeded() {
        if (!orphanCleanupEnabled) {
            return;
        }
        for (Consumer consumer : listGatewayConsumers()) {
            if (consumer == null || StringUtils.isBlank(consumer.getName())) {
                continue;
            }
            try {
                consumerService.delete(consumer.getName());
            } catch (Exception ex) {
                log.warn("Failed to cleanup gateway consumer {} while portal user list is empty.",
                    consumer.getName(), ex);
            }
        }
    }

    private List<Consumer> listGatewayConsumers() {
        try {
            PaginatedResult<Consumer> result = consumerService.list(new CommonPageQuery());
            if (result == null || result.getData() == null) {
                return Collections.emptyList();
            }
            return result.getData();
        } catch (Exception ex) {
            log.warn("Failed to list gateway consumers from key-auth.", ex);
            return Collections.emptyList();
        }
    }

    private Consumer buildDesiredConsumer(PortalUserRecord user, Map<String, List<String>> activeRawKeyMap) {
        if (user == null || StringUtils.isBlank(user.getConsumerName())) {
            return null;
        }
        if (portalUserJdbcService.isBuiltinAdministrator(user.getConsumerName())) {
            return null;
        }
        List<String> keys = new ArrayList<>(activeRawKeyMap.getOrDefault(user.getConsumerName(), Collections.emptyList()));
        String status = StringUtils.lowerCase(StringUtils.trimToEmpty(user.getStatus()));
        if (STATUS_DISABLED.equals(status)) {
            keys = Collections.singletonList(buildRevokedCredentialValue(user.getConsumerName()));
        }
        if (CollectionUtils.isEmpty(keys)) {
            return null;
        }
        KeyAuthCredential keyAuthCredential = new KeyAuthCredential();
        keyAuthCredential.setType(CredentialType.KEY_AUTH);
        keyAuthCredential.setSource(KeyAuthCredentialSource.BEARER.name());
        keyAuthCredential.setValues(keys);

        Consumer consumer = new Consumer();
        consumer.setName(user.getConsumerName());
        consumer.setCredentials(Collections.singletonList(keyAuthCredential));
        return consumer;
    }

    private boolean sameProjection(Consumer existed, Consumer desired) {
        if (existed == null || desired == null) {
            return false;
        }
        Set<String> existedValues = new LinkedHashSet<>(extractRawKeys(existed));
        Set<String> desiredValues = new LinkedHashSet<>(extractRawKeys(desired));
        return existedValues.equals(desiredValues);
    }

    private List<String> extractRawKeys(Consumer consumer) {
        if (consumer == null || CollectionUtils.isEmpty(consumer.getCredentials())) {
            return Collections.emptyList();
        }
        List<String> result = new ArrayList<>();
        for (Credential credential : consumer.getCredentials()) {
            if (!(credential instanceof KeyAuthCredential)) {
                continue;
            }
            KeyAuthCredential keyAuthCredential = (KeyAuthCredential) credential;
            if (CollectionUtils.isEmpty(keyAuthCredential.getValues())) {
                continue;
            }
            for (String value : keyAuthCredential.getValues()) {
                String normalized = StringUtils.trimToNull(value);
                if (normalized != null) {
                    result.add(normalized);
                }
            }
        }
        return result;
    }

    private String buildRevokedCredentialValue(String consumerName) {
        String value = StringUtils.defaultString(consumerName) + ":" + revokedSalt + ":" + STATUS_DISABLED;
        return "revoked_" + sha256(value).substring(0, 32);
    }

    private String sha256(String value) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] bytes = digest.digest(value.getBytes(StandardCharsets.UTF_8));
            StringBuilder builder = new StringBuilder(bytes.length * 2);
            for (byte b : bytes) {
                builder.append(String.format("%02x", b));
            }
            return builder.toString();
        } catch (NoSuchAlgorithmException ex) {
            throw new IllegalStateException("SHA-256 algorithm is unavailable.", ex);
        }
    }
}
