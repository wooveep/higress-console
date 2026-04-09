package com.alibaba.higress.console.service.portal;

import java.util.Collections;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

import javax.annotation.Resource;

import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.AssetGrantRecord;
import com.alibaba.higress.console.model.portal.OrgAccountRecord;
import com.alibaba.higress.sdk.model.RouteAuthConfig;

/**
 * @author Codex
 */
@Service
public class AuthorizationSubjectResolver {

    private static final String STATUS_DISABLED = "disabled";

    @Resource
    private PortalOrganizationJdbcService portalOrganizationJdbcService;

    @Resource
    private PortalAssetGrantJdbcService portalAssetGrantJdbcService;

    public List<String> resolveConsumers(String assetType, String assetId) {
        if (!portalOrganizationJdbcService.enabled() || !portalAssetGrantJdbcService.enabled()) {
            return Collections.emptyList();
        }
        List<AssetGrantRecord> grants = portalAssetGrantJdbcService.listGrants(assetType, assetId);
        if (grants.isEmpty()) {
            return Collections.emptyList();
        }

        List<OrgAccountRecord> accounts = portalOrganizationJdbcService.listAccounts();
        if (accounts.isEmpty()) {
            return Collections.emptyList();
        }

        Set<String> departmentIds = grants.stream()
            .filter(item -> StringUtils.equalsIgnoreCase(item.getSubjectType(), "department"))
            .map(AssetGrantRecord::getSubjectId)
            .filter(StringUtils::isNotBlank)
            .flatMap(departmentId -> portalOrganizationJdbcService.listDepartmentIdsInSubtree(departmentId).stream())
            .collect(Collectors.toCollection(LinkedHashSet::new));

        Set<String> consumerNames = grants.stream()
            .filter(item -> StringUtils.equalsIgnoreCase(item.getSubjectType(), "consumer"))
            .map(AssetGrantRecord::getSubjectId)
            .filter(StringUtils::isNotBlank)
            .collect(Collectors.toCollection(LinkedHashSet::new));
        Set<String> allowedLevels = grants.stream()
            .filter(item -> StringUtils.equalsIgnoreCase(item.getSubjectType(), "user_level"))
            .map(AssetGrantRecord::getSubjectId)
            .filter(StringUtils::isNotBlank)
            .map(StringUtils::lowerCase)
            .collect(Collectors.toCollection(LinkedHashSet::new));
        int minAllowedLevelRank = allowedLevels.stream().mapToInt(this::levelRank).min().orElse(Integer.MAX_VALUE);

        for (OrgAccountRecord account : accounts) {
            if (account == null || StringUtils.isBlank(account.getConsumerName())) {
                continue;
            }
            if (STATUS_DISABLED.equalsIgnoreCase(StringUtils.trimToEmpty(account.getStatus()))) {
                continue;
            }
            if (departmentIds.contains(account.getDepartmentId())) {
                consumerNames.add(account.getConsumerName());
            }
            if (minAllowedLevelRank != Integer.MAX_VALUE && levelRank(account.getUserLevel()) >= minAllowedLevelRank) {
                consumerNames.add(account.getConsumerName());
            }
        }

        return consumerNames.stream().sorted().collect(Collectors.toList());
    }

    private int levelRank(String level) {
        if (StringUtils.equalsIgnoreCase(level, RouteAuthConfig.USER_LEVEL_ULTRA)) {
            return 4;
        }
        if (StringUtils.equalsIgnoreCase(level, RouteAuthConfig.USER_LEVEL_PRO)) {
            return 3;
        }
        if (StringUtils.equalsIgnoreCase(level, RouteAuthConfig.USER_LEVEL_PLUS)) {
            return 2;
        }
        return 1;
    }
}
