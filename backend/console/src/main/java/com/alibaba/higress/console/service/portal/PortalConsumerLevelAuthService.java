/*
 * Copyright (c) 2022-2024 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
 * an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */
package com.alibaba.higress.console.service.portal;

import java.util.Collections;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.stream.Collectors;

import javax.annotation.Resource;

import org.apache.commons.collections4.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.PortalUserRecord;
import com.alibaba.higress.sdk.model.RouteAuthConfig;
import com.alibaba.higress.sdk.model.ai.AiRoute;
import com.alibaba.higress.sdk.model.mcp.ConsumerAuthInfo;
import com.alibaba.higress.sdk.model.mcp.McpServer;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalConsumerLevelAuthService {

    private static final String STATUS_DISABLED = "disabled";

    @Resource
    private PortalUserJdbcService portalUserJdbcService;

    public void resolveRouteAuthConfig(RouteAuthConfig authConfig) {
        if (authConfig == null) {
            return;
        }
        List<String> levels = RouteAuthConfig.normalizeAllowedConsumerLevels(authConfig.getAllowedConsumerLevels());
        authConfig.setAllowedConsumerLevels(levels);
        authConfig.setAllowedConsumers(expandConsumersByAllowedLevels(levels));
    }

    public void resolveAiRouteAuthConfig(AiRoute route) {
        if (route == null) {
            return;
        }
        resolveRouteAuthConfig(route.getAuthConfig());
    }

    public void resolveMcpConsumerAuth(McpServer instance) {
        if (instance == null) {
            return;
        }
        ConsumerAuthInfo consumerAuthInfo = instance.getConsumerAuthInfo();
        if (consumerAuthInfo == null) {
            return;
        }
        List<String> levels = RouteAuthConfig.normalizeAllowedConsumerLevels(consumerAuthInfo.getAllowedConsumerLevels());
        consumerAuthInfo.setAllowedConsumerLevels(levels);
        consumerAuthInfo.setAllowedConsumers(expandConsumersByAllowedLevels(levels));
    }

    private List<String> expandConsumersByAllowedLevels(List<String> levels) {
        if (CollectionUtils.isEmpty(levels)) {
            return Collections.emptyList();
        }
        if (!portalUserJdbcService.enabled()) {
            log.warn("Portal DB is unavailable when resolving consumers by level. levels={}", levels);
            return Collections.emptyList();
        }
        int minRank = levels.stream().mapToInt(this::levelRank).min().orElse(1);
        List<PortalUserRecord> users = portalUserJdbcService.listAllUsers();
        if (CollectionUtils.isEmpty(users)) {
            return Collections.emptyList();
        }

        LinkedHashSet<String> consumerNames = new LinkedHashSet<>();
        for (PortalUserRecord user : users) {
            if (user == null || StringUtils.isBlank(user.getConsumerName())) {
                continue;
            }
            if (portalUserJdbcService.isBuiltinAdministrator(user.getConsumerName())) {
                continue;
            }
            if (STATUS_DISABLED.equalsIgnoreCase(StringUtils.trimToEmpty(user.getStatus()))) {
                continue;
            }
            if (levelRank(user.getUserLevel()) >= minRank) {
                consumerNames.add(user.getConsumerName());
            }
        }
        if (consumerNames.isEmpty()) {
            return Collections.emptyList();
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
