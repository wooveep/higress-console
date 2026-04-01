/*
 * Copyright (c) 2022-2023 Alibaba Group Holding Ltd.
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
package com.alibaba.higress.console.service;

import java.io.IOException;
import java.net.MalformedURLException;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.Iterator;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.SynchronousQueue;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import javax.annotation.PostConstruct;
import javax.annotation.Resource;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import org.apache.commons.collections4.CollectionUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.methods.CloseableHttpResponse;
import org.apache.http.client.methods.HttpUriRequest;
import org.apache.http.client.methods.RequestBuilder;
import org.apache.http.client.utils.URIBuilder;
import org.apache.http.entity.BufferedHttpEntity;
import org.apache.http.entity.InputStreamEntity;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.message.BasicHeader;
import org.apache.tomcat.util.http.fileupload.util.Streams;
import org.jetbrains.annotations.NotNull;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.client.grafana.GrafanaClient;
import com.alibaba.higress.console.client.grafana.models.Datasource;
import com.alibaba.higress.console.client.grafana.models.DatasourceCreationResult;
import com.alibaba.higress.console.client.grafana.models.GrafanaDashboard;
import com.alibaba.higress.console.client.grafana.models.GrafanaSearchResult;
import com.alibaba.higress.console.client.grafana.models.SearchType;
import com.alibaba.higress.console.constant.SystemConfigKey;
import com.alibaba.higress.console.constant.UserConfigKey;
import com.alibaba.higress.console.model.DashboardInfo;
import com.alibaba.higress.console.model.DashboardType;
import com.alibaba.higress.sdk.constant.HigressConstants;
import com.alibaba.higress.sdk.exception.BusinessException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.node.ArrayNode;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.google.common.collect.ImmutableSet;
import com.google.common.util.concurrent.ThreadFactoryBuilder;

import lombok.extern.slf4j.Slf4j;

/**
 * @author CH3CHO
 */
@Slf4j
@Service
public class DashboardServiceImpl implements DashboardService {

    /*
     * ignore hop-to-hop headers.
     * https://datatracker.ietf.org/doc/html/rfc2616#section-13.5.1
     */
    private static final Set<String> IGNORE_REQUEST_HEADERS =
            ImmutableSet.of("connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
                    "te", "trailers", "upgrade", "transfer-encoding", "content-length", "accept-encoding");
    private static final Set<String> IGNORE_RESPONSE_HEADERS =
            ImmutableSet.of("connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
                    "te", "trailers", "upgrade", "transfer-encoding", "content-length", "content-encoding", "server");

    private static final String DATASOURCE_UID_PLACEHOLDER = "${datasource.id}";
    private static final String MAIN_DASHBOARD_DATA_PATH = "/dashboard/main.json";
    private static final String LOG_DASHBOARD_DATA_PATH = "/dashboard/logs.json";
    private static final String AI_DASHBOARD_DATA_PATH = "/dashboard/ai.json";
    private static final String PROM_DATASOURCE_TYPE = "prometheus";
    private static final String LOKI_DATASOURCE_TYPE = "loki";
    private static final String DATASOURCE_ACCESS = "proxy";
    private static final String MONITORING_SELECTOR_KEY_PLACEHOLDER = "${monitoring.selector.key}";
    private static final String MONITORING_NAMESPACE_PLACEHOLDER = "${monitoring.namespace}";
    private static final String MONITORING_GATEWAY_NAME_PLACEHOLDER = "${monitoring.gateway.name}";
    private static final String MONITORING_GATEWAY_SELECTOR_VALUE_PLACEHOLDER =
        "${monitoring.gateway.selector.value}";
    private static final String MONITORING_GATEWAY_CONTAINER_PLACEHOLDER = "${monitoring.gateway.container.name}";
    private static final String MONITORING_CORE_CONTAINER_PLACEHOLDER = "${monitoring.core.container.name}";
    private static final String PROMETHEUS_QUERY_PATH = "/api/v1/query";
    private static final String PROMETHEUS_QUERY_RANGE_PATH = "/api/v1/query_range";
    private static final String PANEL_TYPE_ROW = "row";
    private static final String PANEL_TYPE_STAT = "stat";
    private static final String PANEL_TYPE_TABLE = "table";
    private static final String PANEL_TYPE_TIMESERIES = "timeseries";
    private static final String PANEL_TYPE_GRAPH = "graph";
    private static final String PANEL_TITLE_CPU = "CPU";
    private static final String PANEL_TITLE_MEMORY = "Memory";
    private static final String CONTAINER_CPU_USAGE_METRIC = "container_cpu_usage_seconds_total";
    private static final String CONTAINER_MEMORY_USAGE_METRIC = "container_memory_working_set_bytes";
    private static final String ISTIO_AGENT_CPU_USAGE_METRIC = "istio_agent_process_cpu_seconds_total";
    private static final String ISTIO_AGENT_MEMORY_USAGE_METRIC = "istio_agent_process_resident_memory_bytes";
    private static final String ENVOY_MEMORY_USAGE_METRIC = "envoy_server_memory_physical_size";
    private static final String VALUE_FIELD = "Value";
    private static final long DEFAULT_DASHBOARD_RANGE_MILLIS = 5 * 60 * 1000L;
    private static final int DEFAULT_MAX_DATA_POINTS = 120;
    private static final int MIN_QUERY_STEP_SECONDS = 15;
    private static final Pattern LABEL_VALUES_QUERY_PATTERN = Pattern.compile("label_values\\((.+?),\\s*([^)]+)\\)");
    private static final Pattern VARIABLE_TEMPLATE_PATTERN = Pattern.compile("\\{\\{\\s*([^}]+?)\\s*\\}\\}");

    private static final ExecutorService EXECUTOR =
        new ThreadPoolExecutor(1, 1, 1, TimeUnit.MINUTES, new SynchronousQueue<>(),
            new ThreadFactoryBuilder().setDaemon(true).setNameFormat("DashboardService-Initializer-%d").build());

    @Value("${" + SystemConfigKey.DASHBOARD_OVERWRITE_WHEN_STARTUP_KEY + ":"
        + SystemConfigKey.DASHBOARD_OVERWRITE_WHEN_STARTUP_DEFAULT + "}")
    private boolean overwriteWhenStartUp = SystemConfigKey.DASHBOARD_OVERWRITE_WHEN_STARTUP_DEFAULT;

    @Value("${" + SystemConfigKey.DASHBOARD_BASE_URL_KEY + ":}")
    private String apiBaseUrl;

    private URL apiBaseUrlObject;

    @Value("${" + SystemConfigKey.DASHBOARD_USERNAME_KEY + ":" + SystemConfigKey.DASHBOARD_USERNAME_DEFAULT + "}")
    private String username = SystemConfigKey.DASHBOARD_USERNAME_DEFAULT;

    @Value("${" + SystemConfigKey.DASHBOARD_PASSWORD_KEY + ":" + SystemConfigKey.DASHBOARD_PASSWORD_DEFAULT + "}")
    private String password = SystemConfigKey.DASHBOARD_PASSWORD_DEFAULT;

    @Value("${" + SystemConfigKey.DASHBOARD_DATASOURCE_PROM_NAME_KEY + ":"
        + SystemConfigKey.DASHBOARD_DATASOURCE_PROM_NAME_DEFAULT + "}")
    private String promDatasourceName = SystemConfigKey.DASHBOARD_DATASOURCE_PROM_NAME_DEFAULT;

    @Value("${" + SystemConfigKey.DASHBOARD_DATASOURCE_PROM_URL_KEY + ":}")
    private String promDatasourceUrl;

    @Value("${" + SystemConfigKey.DASHBOARD_DATASOURCE_LOKI_NAME_KEY + ":"
        + SystemConfigKey.DASHBOARD_DATASOURCE_LOKI_NAME_DEFAULT + "}")
    private String lokiDatasourceName = SystemConfigKey.DASHBOARD_DATASOURCE_LOKI_NAME_DEFAULT;

    @Value("${" + SystemConfigKey.DASHBOARD_DATASOURCE_LOKI_URL_KEY + ":}")
    private String lokiDatasourceUrl;

    @Value("${" + SystemConfigKey.NS_KEY + ":" + HigressConstants.NS_DEFAULT + "}")
    private String controllerNamespace = HigressConstants.NS_DEFAULT;

    @Value("${" + SystemConfigKey.CONTROLLER_INGRESS_CLASS_NAME_KEY + ":"
        + HigressConstants.CONTROLLER_INGRESS_CLASS_NAME_DEFAULT + "}")
    private String controllerIngressClassName = HigressConstants.CONTROLLER_INGRESS_CLASS_NAME_DEFAULT;

    @Value("${" + SystemConfigKey.DASHBOARD_GATEWAY_LABEL_KEY + ":}")
    private String dashboardGatewayLabelKey;

    @Value("${" + SystemConfigKey.DASHBOARD_GATEWAY_NAME_KEY + ":}")
    private String dashboardGatewayName;

    @Value("${" + SystemConfigKey.DASHBOARD_GATEWAY_CONTAINER_NAME_KEY + ":}")
    private String dashboardGatewayContainerName;

    @Value("${" + SystemConfigKey.DASHBOARD_CORE_CONTAINER_NAME_KEY + ":}")
    private String dashboardCoreContainerName;

    @Value("${" + SystemConfigKey.DASHBOARD_PROXY_CONNECTION_TIMEOUT_KEY + ":"
        + SystemConfigKey.DASHBOARD_PROXY_CONNECTION_TIMEOUT_DEFAULT + "}")
    private int proxyConnectionTimeout = SystemConfigKey.DASHBOARD_PROXY_CONNECTION_TIMEOUT_DEFAULT;

    @Value("${" + SystemConfigKey.DASHBOARD_PROXY_SOCKET_TIMEOUT_KEY + ":"
        + SystemConfigKey.DASHBOARD_PROXY_SOCKET_TIMEOUT_DEFAULT + "}")
    private int proxySocketTimeout = SystemConfigKey.DASHBOARD_PROXY_SOCKET_TIMEOUT_DEFAULT;

    private ConfigService configService;
    private GrafanaClient grafanaClient;
    private CloseableHttpClient realServerClient;
    private String realServerBaseUrl;

    private Map<DashboardType, DashboardConfiguration> dashboardConfigurations;

    @Resource
    public void setConfigService(ConfigService configService) {
        this.configService = configService;
    }

    @PostConstruct
    public void initialize() {
        Map<DashboardType, DashboardConfiguration> dashboardConfigurations = new HashMap<>();
        try {
            dashboardConfigurations.put(DashboardType.MAIN,
                new DashboardConfiguration(DashboardType.MAIN, MAIN_DASHBOARD_DATA_PATH));
            dashboardConfigurations.put(DashboardType.AI,
                new DashboardConfiguration(DashboardType.AI, AI_DASHBOARD_DATA_PATH));
            dashboardConfigurations.put(DashboardType.LOG,
                new DashboardConfiguration(DashboardType.LOG, LOG_DASHBOARD_DATA_PATH));
        } catch (IOException e) {
            throw new IllegalStateException("Error occurs when loading dashboard configurations from resource.", e);
        }
        this.dashboardConfigurations = Collections.unmodifiableMap(dashboardConfigurations);

        if (isBuiltIn()) {
            try {
                apiBaseUrlObject = new URL(apiBaseUrl);
            } catch (MalformedURLException e) {
                throw new IllegalArgumentException("Invalid dashboard base url: " + apiBaseUrl, e);
            }

            RequestConfig requestConfig = RequestConfig.custom().setConnectTimeout(proxyConnectionTimeout)
                .setSocketTimeout(proxySocketTimeout).build();
            realServerClient =
                HttpClients.custom().setDefaultRequestConfig(requestConfig).disableRedirectHandling().build();
            realServerBaseUrl = apiBaseUrl.substring(0, apiBaseUrl.length() - apiBaseUrlObject.getPath().length());

            grafanaClient = new GrafanaClient(apiBaseUrl, username, password);
            EXECUTOR.submit(new DashboardInitializer(overwriteWhenStartUp));
        }
    }

    public boolean isBuiltIn() {
        return StringUtils.isNoneBlank(apiBaseUrl, promDatasourceUrl, lokiDatasourceUrl);
    }

    @Override
    public DashboardInfo getDashboardInfo() {
        return getDashboardInfo(DashboardType.MAIN);
    }

    @Override
    public DashboardInfo getDashboardInfo(DashboardType type) {
        return isBuiltIn() ? getBuiltInDashboardInfo(type) : getConfiguredDashboardInfo(type);
    }

    @Override
    public void initializeDashboard(boolean overwrite) {
        if (!isBuiltIn()) {
            throw new IllegalStateException("No built-in dashboard is available.");
        }

        List<Datasource> datasources;
        try {
            datasources = grafanaClient.getDatasources();
        } catch (IOException e) {
            throw new BusinessException("Error occurs when loading datasources from Grafana.", e);
        }
        String promDatasourceUid = configurePrometheusDatasource(datasources);
        String lokiDatasourceUid = configureLokiDatasource(datasources);

        List<GrafanaSearchResult> results;
        try {
            results = grafanaClient.search(null, SearchType.DB, null, null);
        } catch (IOException e) {
            throw new BusinessException("Error occurs when loading dashboard info from Grafana.", e);
        }
        for (DashboardConfiguration configuration : dashboardConfigurations.values()) {
            String datasourceId = configuration.getType() == DashboardType.LOG ? lokiDatasourceUid : promDatasourceUid;
            configureDashboard(results, configuration.getDashboard().getTitle(), configuration.getRaw(), datasourceId,
                overwrite);
        }
    }

    @Override
    public void setDashboardUrl(String url) {
        setDashboardUrl(DashboardType.MAIN, url);
    }

    @Override
    public void setDashboardUrl(DashboardType type, String url) {
        if (StringUtils.isBlank(url)) {
            throw new IllegalArgumentException("url cannot be null or blank.");
        }
        if (isBuiltIn()) {
            throw new IllegalStateException("Manual dashboard configuration is disabled.");
        }
        DashboardConfiguration configuration = getDashboardConfiguration(type);
        configService.setConfig(configuration.getConfigKey(), url);
    }

    @Override
    public String buildConfigData(String datasourceUid) {
        return buildConfigData(DashboardType.MAIN, datasourceUid);
    }

    @Override
    public String buildConfigData(DashboardType type, String datasourceUid) {
        DashboardConfiguration configuration = getDashboardConfiguration(type);
        return buildConfigData(configuration.getRaw(), datasourceUid);
    }

    @Override
    public ObjectNode getNativeDashboard(DashboardType type, Long from, Long to, String gateway, String namespace) {
        if (!isBuiltIn()) {
            throw new IllegalStateException("Native dashboard is only available for built-in monitoring.");
        }
        if (type == DashboardType.LOG) {
            throw new IllegalArgumentException("Native dashboard does not support LOG panels.");
        }

        DashboardConfiguration configuration = getDashboardConfiguration(type);
        ObjectNode dashboardNode = parseConfiguredDashboard(configuration);
        TimeWindow timeWindow = resolveTimeWindow(dashboardNode, from, to);
        VariableContext variables = resolveVariables(dashboardNode, gateway, namespace);

        ObjectNode result = GrafanaClient.MAPPER.createObjectNode();
        result.put("title", dashboardNode.path("title").asText(configuration.getDashboard().getTitle()));
        result.put("type", type.toString());
        result.put("from", timeWindow.getFrom());
        result.put("to", timeWindow.getTo());
        result.put("defaultRangeMs", timeWindow.getDefaultRangeMs());
        result.set("variables", buildVariableState(variables));
        result.set("rows", renderDashboardRows(dashboardNode, variables, timeWindow));
        return result;
    }

    @Override
    public void forwardDashboardRequest(HttpServletRequest request, HttpServletResponse response) throws IOException {
        if (!isBuiltIn()) {
            throw new IllegalStateException(
                "Dashboard request forward function is only available for built-in dashboard.");
        }

        HttpUriRequest proxyRequest = buildRealServerRequest(request);
        try (CloseableHttpResponse proxyResponse = realServerClient.execute(proxyRequest)) {
            forwardResponse(response, proxyResponse);
        }
    }

    private String configurePrometheusDatasource(List<Datasource> existedDatasources) {
        String datasourceUid = null;
        if (CollectionUtils.isNotEmpty(existedDatasources)) {
            datasourceUid = existedDatasources.stream().filter(ds -> promDatasourceUrl.equals(ds.getUrl())).findFirst()
                .map(Datasource::getUid).orElse(null);
        }
        if (datasourceUid == null) {
            Datasource datasource = new Datasource();
            datasource.setType(PROM_DATASOURCE_TYPE);
            datasource.setName(promDatasourceName);
            datasource.setUrl(promDatasourceUrl);
            datasource.setAccess(DATASOURCE_ACCESS);
            try {
                DatasourceCreationResult result = grafanaClient.createDatasource(datasource);
                if (result.getDatasource() == null) {
                    throw new BusinessException("Creating data source call returns success but no datasource object."
                        + " Message=" + result.getMessage());
                }
                datasourceUid = result.getDatasource().getUid();
            } catch (IOException e) {
                throw new BusinessException("Error occurs when creating Prometheus datasource in Grafana.", e);
            }
        }
        return datasourceUid;
    }

    private String configureLokiDatasource(List<Datasource> existedDatasources) {
        String datasourceUid = null;
        if (CollectionUtils.isNotEmpty(existedDatasources)) {
            datasourceUid = existedDatasources.stream().filter(ds -> lokiDatasourceUrl.equals(ds.getUrl())).findFirst()
                .map(Datasource::getUid).orElse(null);
        }
        if (datasourceUid == null) {
            Datasource datasource = new Datasource();
            datasource.setType(LOKI_DATASOURCE_TYPE);
            datasource.setName(lokiDatasourceName);
            datasource.setUrl(lokiDatasourceUrl);
            datasource.setAccess(DATASOURCE_ACCESS);
            try {
                DatasourceCreationResult result = grafanaClient.createDatasource(datasource);
                if (result.getDatasource() == null) {
                    throw new BusinessException("Creating data source call returns success but no datasource object."
                        + " Message=" + result.getMessage());
                }
                datasourceUid = result.getDatasource().getUid();
            } catch (IOException e) {
                throw new BusinessException("Error occurs when creating Loki datasource in Grafana.", e);
            }
        }
        return datasourceUid;
    }

    private void configureDashboard(List<GrafanaSearchResult> results, String title, String configuration,
        String datasourceUid, boolean overwrite) {
        if (StringUtils.isEmpty(title)) {
            throw new IllegalStateException("No title is found in the configured dashboard.");
        }

        String existedDashboardUid = results.stream().filter(r -> title.equals(r.getTitle()))
            .map(GrafanaSearchResult::getUid).findFirst().orElse(null);
        if (StringUtils.isNotEmpty(existedDashboardUid) && !overwrite) {
            return;
        }

        String dashboardData = buildConfigData(configuration, datasourceUid);
        GrafanaDashboard dashboard;
        try {
            dashboard = GrafanaClient.parseDashboardData(dashboardData);
            dashboard.setId(null);
            dashboard.setUid(null);
            dashboard.setVersion(null);
        } catch (IOException e) {
            throw new IllegalStateException("Unable to parse the configured dashboard data.", e);
        }

        try {
            if (StringUtils.isNotEmpty(existedDashboardUid)) {
                GrafanaDashboard existedDashboard = grafanaClient.getDashboard(existedDashboardUid);
                if (existedDashboard != null) {
                    dashboard.setId(existedDashboard.getId());
                    dashboard.setUid(existedDashboardUid);
                    dashboard.setVersion(existedDashboard.getVersion());
                }
            }
            if (dashboard.getId() == null) {
                grafanaClient.createDashboard(dashboard);
            } else {
                grafanaClient.updateDashboard(dashboard);
            }
        } catch (IOException e) {
            throw new BusinessException("Error occurs when creating Higress dashboard in Grafana.", e);
        }
    }

    private DashboardInfo getBuiltInDashboardInfo(DashboardType type) {
        DashboardConfiguration configuration = dashboardConfigurations.get(type);
        if (configuration == null) {
            throw new IllegalArgumentException("Invalid dashboard type: " + type);
        }
        List<GrafanaSearchResult> results;
        try {
            results = grafanaClient.search(null, SearchType.DB, null, null);
        } catch (IOException e) {
            throw new BusinessException("Error occurs when loading dashboard info from Grafana.", e);
        }
        if (CollectionUtils.isEmpty(results)) {
            return new DashboardInfo(true, null, null);
        }
        String expectedTitle = configuration.getDashboard().getTitle();
        if (StringUtils.isEmpty(expectedTitle)) {
            throw new IllegalStateException("No title is found in the configured dashboard.");
        }
        Optional<GrafanaSearchResult> result =
            results.stream().filter(r -> expectedTitle.equals(r.getTitle())).findFirst();
        return result.map(r -> new DashboardInfo(true, r.getUid(), r.getUrl())).orElse(null);
    }

    private DashboardInfo getConfiguredDashboardInfo(DashboardType type) {
        DashboardConfiguration configuration = dashboardConfigurations.get(type);
        String url = configService.getString(configuration.getConfigKey());
        return new DashboardInfo(false, null, url);
    }

    private ObjectNode parseConfiguredDashboard(DashboardConfiguration configuration) {
        try {
            return (ObjectNode) GrafanaClient.MAPPER.readTree(buildConfigData(configuration.getRaw(),
                DATASOURCE_UID_PLACEHOLDER));
        } catch (IOException e) {
            throw new IllegalStateException("Unable to parse dashboard configuration for native rendering.", e);
        }
    }

    private TimeWindow resolveTimeWindow(ObjectNode dashboardNode, Long from, Long to) {
        long now = System.currentTimeMillis();
        long defaultRangeMs = parseRelativeDuration(dashboardNode.path("time").path("from").asText());
        if (defaultRangeMs <= 0) {
            defaultRangeMs = DEFAULT_DASHBOARD_RANGE_MILLIS;
        }
        long end = to != null && to > 0 ? to : now;
        long start = from != null && from > 0 && from < end ? from : end - defaultRangeMs;
        if (start >= end) {
            start = Math.max(0, end - defaultRangeMs);
        }
        return new TimeWindow(start, end, defaultRangeMs);
    }

    private VariableContext resolveVariables(ObjectNode dashboardNode, String requestedGateway, String requestedNamespace) {
        JsonNode gatewayVariable = findVariableNode(dashboardNode, "gateway");
        JsonNode namespaceVariable = findVariableNode(dashboardNode, "namespace");

        List<String> gatewayOptions = loadVariableOptions(gatewayVariable);
        String configuredGateway = gatewayVariable.path("current").path("value").asText(null);
        String selectedGateway = selectOption(requestedGateway, configuredGateway, gatewayOptions);

        List<String> namespaceOptions = loadVariableOptions(namespaceVariable);
        String configuredNamespace = namespaceVariable.path("current").path("value").asText(null);
        String derivedNamespace = applyRegex(selectedGateway, namespaceVariable.path("regex").asText());
        String selectedNamespace = selectOption(requestedNamespace,
            StringUtils.firstNonBlank(derivedNamespace, configuredNamespace, controllerNamespace), namespaceOptions);

        if (StringUtils.isBlank(selectedGateway) && CollectionUtils.isNotEmpty(gatewayOptions)) {
            selectedGateway = gatewayOptions.get(0);
        }
        if (StringUtils.isBlank(selectedNamespace)) {
            selectedNamespace = StringUtils.firstNonBlank(derivedNamespace, configuredNamespace, controllerNamespace);
        }

        gatewayOptions = ensureSelectedOption(gatewayOptions, selectedGateway);
        namespaceOptions = ensureSelectedOption(namespaceOptions, selectedNamespace);
        String gatewayContainerName = resolveGatewayContainerName(selectedNamespace);
        String coreContainerName = resolveCoreContainerName(selectedNamespace);
        return new VariableContext(selectedGateway, gatewayOptions, selectedNamespace, namespaceOptions,
            gatewayContainerName, coreContainerName, resolveWorkloadMetricsMode(selectedNamespace, gatewayContainerName));
    }

    private ObjectNode buildVariableState(VariableContext variableContext) {
        ObjectNode variables = GrafanaClient.MAPPER.createObjectNode();
        variables.set("gateway", buildVariableNode(variableContext.getGateway(), variableContext.getGatewayOptions()));
        variables.set("namespace",
            buildVariableNode(variableContext.getNamespace(), variableContext.getNamespaceOptions()));
        return variables;
    }

    private ObjectNode buildVariableNode(String value, List<String> options) {
        ObjectNode variableNode = GrafanaClient.MAPPER.createObjectNode();
        variableNode.put("value", StringUtils.defaultString(value));
        ArrayNode optionNodes = variableNode.putArray("options");
        for (String option : options) {
            optionNodes.add(option);
        }
        return variableNode;
    }

    private ArrayNode renderDashboardRows(ObjectNode dashboardNode, VariableContext variables, TimeWindow timeWindow) {
        List<ObjectNode> rowNodes = new ArrayList<>();
        List<ObjectNode> metricPanels = new ArrayList<>();
        ArrayNode panels = dashboardNode.withArray("panels");
        for (JsonNode panelNode : panels) {
            if (!(panelNode instanceof ObjectNode)) {
                continue;
            }
            ObjectNode objectNode = (ObjectNode) panelNode;
            if (PANEL_TYPE_ROW.equals(objectNode.path("type").asText())) {
                rowNodes.add(objectNode);
            } else {
                metricPanels.add(objectNode);
            }
        }

        rowNodes.sort(this::compareGridPosition);
        metricPanels.sort(this::compareGridPosition);

        List<RowBucket> rows = new ArrayList<>();
        for (ObjectNode rowNode : rowNodes) {
            RowBucket row = new RowBucket(rowNode.path("title").asText(), rowNode.path("collapsed").asBoolean(false),
                gridPosValue(rowNode, "y"));
            for (JsonNode nestedPanelNode : rowNode.withArray("panels")) {
                if (nestedPanelNode instanceof ObjectNode) {
                    row.getPanels().add(renderPanel((ObjectNode) nestedPanelNode, variables, timeWindow));
                }
            }
            rows.add(row);
        }
        if (rows.isEmpty()) {
            rows.add(new RowBucket(dashboardNode.path("title").asText("Overview"), false, Integer.MIN_VALUE));
        }

        for (ObjectNode panelNode : metricPanels) {
            RowBucket row = resolveRowBucket(rows, panelNode);
            row.getPanels().add(renderPanel(panelNode, variables, timeWindow));
        }

        ArrayNode renderedRows = GrafanaClient.MAPPER.createArrayNode();
        for (RowBucket row : rows) {
            if (row.getPanels().isEmpty()) {
                continue;
            }
            ObjectNode rowNode = renderedRows.addObject();
            rowNode.put("title", row.getTitle());
            rowNode.put("collapsed", row.isCollapsed());
            rowNode.set("panels", row.getPanels());
        }
        return renderedRows;
    }

    private RowBucket resolveRowBucket(List<RowBucket> rows, ObjectNode panelNode) {
        int panelY = gridPosValue(panelNode, "y");
        RowBucket selectedRow = rows.get(0);
        for (RowBucket row : rows) {
            if (panelY >= row.getStartY()) {
                selectedRow = row;
            } else {
                break;
            }
        }
        return selectedRow;
    }

    private ObjectNode renderPanel(ObjectNode panelNode, VariableContext variables, TimeWindow timeWindow) {
        ObjectNode effectivePanelNode = adaptPanelForNativeRendering(panelNode, variables);
        ObjectNode renderedPanel = GrafanaClient.MAPPER.createObjectNode();
        String normalizedType = normalizePanelType(effectivePanelNode.path("type").asText());
        renderedPanel.put("id", effectivePanelNode.path("id").asInt());
        renderedPanel.put("title", effectivePanelNode.path("title").asText());
        renderedPanel.put("type", normalizedType);
        renderedPanel.put("unit", resolvePanelUnit(effectivePanelNode));
        renderedPanel.set("gridPos", effectivePanelNode.path("gridPos").deepCopy());
        try {
            if (PANEL_TYPE_STAT.equals(normalizedType)) {
                renderedPanel.set("stat", renderStatPanel(effectivePanelNode, variables, timeWindow));
            } else if (PANEL_TYPE_TABLE.equals(normalizedType)) {
                renderedPanel.set("table", renderTablePanel(effectivePanelNode, variables, timeWindow));
            } else {
                renderedPanel.set("series", renderTimeseriesPanel(effectivePanelNode, variables, timeWindow));
            }
        } catch (Exception ex) {
            log.warn("Failed to render native dashboard panel: {}", effectivePanelNode.path("title").asText(), ex);
            renderedPanel.put("error", ex.getMessage());
        }
        return renderedPanel;
    }

    private ObjectNode renderStatPanel(ObjectNode panelNode, VariableContext variables, TimeWindow timeWindow)
        throws IOException {
        ObjectNode statNode = GrafanaClient.MAPPER.createObjectNode();
        ObjectNode targetNode = firstVisibleTarget(panelNode.withArray("targets"));
        if (targetNode == null) {
            statNode.putNull("value");
            return statNode;
        }

        JsonNode dataNode = queryPrometheusInstant(resolveQueryExpression(targetNode.path("expr").asText(),
            variables, timeWindow), timeWindow.getTo());
        Double value = extractInstantValue(dataNode);
        if (value == null) {
            statNode.putNull("value");
        } else {
            statNode.put("value", value);
        }
        return statNode;
    }

    private ArrayNode renderTimeseriesPanel(ObjectNode panelNode, VariableContext variables, TimeWindow timeWindow)
        throws IOException {
        ArrayNode renderedSeries = GrafanaClient.MAPPER.createArrayNode();
        for (JsonNode targetNode : panelNode.withArray("targets")) {
            if (!(targetNode instanceof ObjectNode) || targetNode.path("hide").asBoolean(false)) {
                continue;
            }
            ObjectNode objectNode = (ObjectNode) targetNode;
            JsonNode dataNode = queryPrometheusRange(resolveQueryExpression(objectNode.path("expr").asText(),
                variables, timeWindow), timeWindow, resolveStepSeconds(panelNode, objectNode, timeWindow));
            appendSeries(renderedSeries, dataNode, objectNode, panelNode.path("title").asText());
        }
        return renderedSeries;
    }

    private ObjectNode renderTablePanel(ObjectNode panelNode, VariableContext variables, TimeWindow timeWindow)
        throws IOException {
        List<ObjectNode> visibleTargets = new ArrayList<>();
        for (JsonNode targetNode : panelNode.withArray("targets")) {
            if (targetNode instanceof ObjectNode && !targetNode.path("hide").asBoolean(false)) {
                visibleTargets.add((ObjectNode) targetNode);
            }
        }

        List<LinkedHashMap<String, Object>> rows = new ArrayList<>();
        if (!visibleTargets.isEmpty()) {
            List<TableFrame> frames = new ArrayList<>();
            boolean multiTarget = visibleTargets.size() > 1;
            for (ObjectNode targetNode : visibleTargets) {
                JsonNode dataNode = queryPrometheusInstant(resolveQueryExpression(targetNode.path("expr").asText(),
                    variables, timeWindow), timeWindow.getTo());
                frames.add(toTableFrame(dataNode, targetNode.path("refId").asText(), multiTarget));
            }
            rows = mergeTableFrames(frames, panelNode.withArray("transformations"));
            rows = applyTableTransformations(rows, panelNode.withArray("transformations"));
            rows = applyTableSorting(rows, panelNode.path("options").path("sortBy"));
        }

        ObjectNode tableNode = GrafanaClient.MAPPER.createObjectNode();
        List<String> columns = collectColumns(rows);
        ArrayNode columnNodes = tableNode.putArray("columns");
        for (String column : columns) {
            ObjectNode columnNode = columnNodes.addObject();
            columnNode.put("key", column);
            columnNode.put("title", column);
        }
        ArrayNode rowNodes = tableNode.putArray("rows");
        for (Map<String, Object> row : rows) {
            ObjectNode rowNode = rowNodes.addObject();
            for (Map.Entry<String, Object> entry : row.entrySet()) {
                rowNode.set(entry.getKey(), GrafanaClient.MAPPER.valueToTree(entry.getValue()));
            }
        }
        return tableNode;
    }

    private String buildConfigData(String dashboardConfiguration, String datasourceUid) {
        String configuredDashboard = dashboardConfiguration;
        for (Map.Entry<String, String> entry : buildDashboardTemplateParameters(datasourceUid).entrySet()) {
            configuredDashboard = configuredDashboard.replace(entry.getKey(), entry.getValue());
        }
        return configuredDashboard;
    }

    private ObjectNode firstVisibleTarget(ArrayNode targets) {
        for (JsonNode targetNode : targets) {
            if (targetNode instanceof ObjectNode && !targetNode.path("hide").asBoolean(false)) {
                return (ObjectNode) targetNode;
            }
        }
        return null;
    }

    private int compareGridPosition(ObjectNode left, ObjectNode right) {
        int yCompare = Integer.compare(gridPosValue(left, "y"), gridPosValue(right, "y"));
        if (yCompare != 0) {
            return yCompare;
        }
        return Integer.compare(gridPosValue(left, "x"), gridPosValue(right, "x"));
    }

    private int gridPosValue(ObjectNode panelNode, String key) {
        return panelNode.path("gridPos").path(key).asInt();
    }

    private String normalizePanelType(String panelType) {
        if (PANEL_TYPE_GRAPH.equals(panelType) || PANEL_TYPE_TIMESERIES.equals(panelType)) {
            return PANEL_TYPE_TIMESERIES;
        }
        if (PANEL_TYPE_TABLE.equals(panelType)) {
            return PANEL_TYPE_TABLE;
        }
        return PANEL_TYPE_STAT.equals(panelType) ? PANEL_TYPE_STAT : PANEL_TYPE_TIMESERIES;
    }

    private String resolvePanelUnit(ObjectNode panelNode) {
        String unit = panelNode.path("fieldConfig").path("defaults").path("unit").asText();
        if (StringUtils.isNotBlank(unit)) {
            return unit;
        }
        return panelNode.path("yaxes").path(0).path("format").asText("");
    }

    private ObjectNode adaptPanelForNativeRendering(ObjectNode panelNode, VariableContext variables) {
        if (isContainerCpuPanel(panelNode)) {
            return variables.getWorkloadMetricsMode() == WorkloadMetricsMode.CONTAINER
                ? buildContainerCpuPanel(panelNode, variables)
                : buildStandaloneCpuPanel(panelNode, variables);
        }
        if (isContainerMemoryPanel(panelNode)) {
            return variables.getWorkloadMetricsMode() == WorkloadMetricsMode.CONTAINER
                ? buildContainerMemoryPanel(panelNode, variables)
                : buildStandaloneMemoryPanel(panelNode, variables);
        }
        return panelNode;
    }

    private boolean isContainerCpuPanel(ObjectNode panelNode) {
        return PANEL_TITLE_CPU.equals(panelNode.path("title").asText())
            && panelContainsMetric(panelNode, CONTAINER_CPU_USAGE_METRIC);
    }

    private boolean isContainerMemoryPanel(ObjectNode panelNode) {
        return PANEL_TITLE_MEMORY.equals(panelNode.path("title").asText())
            && panelContainsMetric(panelNode, CONTAINER_MEMORY_USAGE_METRIC);
    }

    private boolean panelContainsMetric(ObjectNode panelNode, String metricName) {
        for (JsonNode targetNode : panelNode.withArray("targets")) {
            if (StringUtils.contains(targetNode.path("expr").asText(), metricName)) {
                return true;
            }
        }
        return false;
    }

    private ObjectNode buildContainerCpuPanel(ObjectNode panelNode, VariableContext variables) {
        ObjectNode adjustedPanel = panelNode.deepCopy();
        ArrayNode targets = adjustedPanel.putArray("targets");
        targets.addObject()
            .put("expr", buildContainerCpuExpression(variables))
            .put("legendFormat", "{{pod}}")
            .put("refId", "A");
        return adjustedPanel;
    }

    private ObjectNode buildContainerMemoryPanel(ObjectNode panelNode, VariableContext variables) {
        ObjectNode adjustedPanel = panelNode.deepCopy();
        ArrayNode targets = adjustedPanel.putArray("targets");
        targets.addObject()
            .put("expr", buildContainerGatewayMemoryExpression(variables))
            .put("legendFormat", "{{pod}}-envoy")
            .put("refId", "A");
        targets.addObject()
            .put("expr", buildContainerDiscoveryMemoryExpression(variables))
            .put("legendFormat", "{{pod}}-istio")
            .put("refId", "B");
        if (StringUtils.isNotBlank(variables.getCoreContainerName())
            && !"discovery".equals(variables.getCoreContainerName())) {
            targets.addObject()
                .put("expr", buildContainerCoreMemoryExpression(variables))
                .put("legendFormat", "{{pod}}-core")
                .put("refId", "C");
        }
        return adjustedPanel;
    }

    private ObjectNode buildStandaloneCpuPanel(ObjectNode panelNode, VariableContext variables) {
        ObjectNode adjustedPanel = panelNode.deepCopy();
        ArrayNode targets = adjustedPanel.putArray("targets");
        targets.addObject()
            .put("expr", buildStandaloneCpuExpression(variables))
            .put("legendFormat", "{{pod}}-agent")
            .put("refId", "A");
        return adjustedPanel;
    }

    private ObjectNode buildStandaloneMemoryPanel(ObjectNode panelNode, VariableContext variables) {
        ObjectNode adjustedPanel = panelNode.deepCopy();
        ArrayNode targets = adjustedPanel.putArray("targets");
        targets.addObject()
            .put("expr", buildStandaloneEnvoyMemoryExpression(variables))
            .put("legendFormat", "{{pod}}-envoy")
            .put("refId", "A");
        targets.addObject()
            .put("expr", buildStandaloneAgentMemoryExpression(variables))
            .put("legendFormat", "{{pod}}-agent")
            .put("refId", "B");
        targets.addObject()
            .put("expr", buildStandaloneTotalMemoryExpression(variables))
            .put("legendFormat", "{{pod}}-total")
            .put("refId", "C");
        return adjustedPanel;
    }

    private String buildContainerCpuExpression(VariableContext variables) {
        return String.format(Locale.ROOT,
            "100 * sum(irate(%s{container=\"%s\", namespace=\"%s\"}[1m])) by (pod)",
            CONTAINER_CPU_USAGE_METRIC, escapePrometheusLabelValue(variables.getGatewayContainerName()),
            escapePrometheusLabelValue(resolveDashboardNamespace(variables.getNamespace())));
    }

    private String buildContainerGatewayMemoryExpression(VariableContext variables) {
        return String.format(Locale.ROOT, "max(%s{container=\"%s\", namespace=\"%s\"}) by (pod)",
            CONTAINER_MEMORY_USAGE_METRIC, escapePrometheusLabelValue(variables.getGatewayContainerName()),
            escapePrometheusLabelValue(resolveDashboardNamespace(variables.getNamespace())));
    }

    private String buildContainerDiscoveryMemoryExpression(VariableContext variables) {
        return String.format(Locale.ROOT, "max(%s{container=\"discovery\", namespace=\"%s\"}) by (pod)",
            CONTAINER_MEMORY_USAGE_METRIC, escapePrometheusLabelValue(resolveDashboardNamespace(variables.getNamespace())));
    }

    private String buildContainerCoreMemoryExpression(VariableContext variables) {
        return String.format(Locale.ROOT, "max(%s{container=\"%s\", namespace=\"%s\"}) by (pod)",
            CONTAINER_MEMORY_USAGE_METRIC, escapePrometheusLabelValue(variables.getCoreContainerName()),
            escapePrometheusLabelValue(resolveDashboardNamespace(variables.getNamespace())));
    }

    private String buildStandaloneCpuExpression(VariableContext variables) {
        return String.format(Locale.ROOT,
            "100 * max(rate(%s{container=\"%s\", namespace=\"%s\"}[1m])) by (pod)",
            ISTIO_AGENT_CPU_USAGE_METRIC, escapePrometheusLabelValue(variables.getGatewayContainerName()),
            escapePrometheusLabelValue(resolveDashboardNamespace(variables.getNamespace())));
    }

    private String buildStandaloneEnvoyMemoryExpression(VariableContext variables) {
        return String.format(Locale.ROOT, "max(%s{container=\"%s\", namespace=\"%s\"}) by (pod)",
            ENVOY_MEMORY_USAGE_METRIC, escapePrometheusLabelValue(variables.getGatewayContainerName()),
            escapePrometheusLabelValue(resolveDashboardNamespace(variables.getNamespace())));
    }

    private String buildStandaloneAgentMemoryExpression(VariableContext variables) {
        return String.format(Locale.ROOT, "max(%s{container=\"%s\", namespace=\"%s\"}) by (pod)",
            ISTIO_AGENT_MEMORY_USAGE_METRIC, escapePrometheusLabelValue(variables.getGatewayContainerName()),
            escapePrometheusLabelValue(resolveDashboardNamespace(variables.getNamespace())));
    }

    private String buildStandaloneTotalMemoryExpression(VariableContext variables) {
        String namespace = escapePrometheusLabelValue(resolveDashboardNamespace(variables.getNamespace()));
        String containerName = escapePrometheusLabelValue(variables.getGatewayContainerName());
        return String.format(Locale.ROOT,
            "max(%s{container=\"%s\", namespace=\"%s\"}) by (pod) + "
                + "max(%s{container=\"%s\", namespace=\"%s\"}) by (pod)",
            ENVOY_MEMORY_USAGE_METRIC, containerName, namespace,
            ISTIO_AGENT_MEMORY_USAGE_METRIC, containerName, namespace);
    }

    private int resolveStepSeconds(ObjectNode panelNode, ObjectNode targetNode, TimeWindow timeWindow) {
        long rangeSeconds = Math.max(1L, (timeWindow.getTo() - timeWindow.getFrom()) / 1000L);
        int maxDataPoints = panelNode.path("maxDataPoints").asInt(DEFAULT_MAX_DATA_POINTS);
        int calculatedStep = (int) Math.ceil((double) rangeSeconds / Math.max(1, maxDataPoints));
        int configuredStep = targetNode.path("step").asInt(0);
        return Math.max(MIN_QUERY_STEP_SECONDS, Math.max(calculatedStep, configuredStep));
    }

    private JsonNode findVariableNode(ObjectNode dashboardNode, String name) {
        for (JsonNode variableNode : dashboardNode.path("templating").path("list")) {
            if (name.equals(variableNode.path("name").asText())) {
                return variableNode;
            }
        }
        return GrafanaClient.MAPPER.createObjectNode();
    }

    private List<String> loadVariableOptions(JsonNode variableNode) {
        String query = variableNode.path("query").path("query").asText(variableNode.path("definition").asText());
        Matcher matcher = LABEL_VALUES_QUERY_PATTERN.matcher(StringUtils.trimToEmpty(query));
        if (!matcher.matches()) {
            return ensureSelectedOption(Collections.emptyList(), variableNode.path("current").path("value").asText());
        }

        List<String> rawValues = queryPrometheusLabelValues(StringUtils.trim(matcher.group(2)),
            StringUtils.trim(matcher.group(1)));
        String regex = variableNode.path("regex").asText();
        LinkedHashSet<String> options = new LinkedHashSet<>();
        for (String rawValue : rawValues) {
            String transformed = applyRegex(rawValue, regex);
            if (StringUtils.isNotBlank(transformed)) {
                options.add(transformed);
            }
        }
        return ensureSelectedOption(new ArrayList<>(options), variableNode.path("current").path("value").asText());
    }

    private List<String> queryPrometheusLabelValues(String labelName, String matchSelector) {
        Map<String, String> parameters = new LinkedHashMap<>();
        if (StringUtils.isNotBlank(matchSelector)) {
            parameters.put("match[]", matchSelector);
        }
        JsonNode dataNode;
        try {
            dataNode = executePrometheusApi("/api/v1/label/" + labelName + "/values", parameters);
        } catch (IOException e) {
            throw new BusinessException("Error occurs when loading Prometheus label values.", e);
        }

        List<String> values = new ArrayList<>();
        if (dataNode instanceof ArrayNode) {
            for (JsonNode valueNode : dataNode) {
                values.add(valueNode.asText());
            }
        }
        return values;
    }

    private List<String> ensureSelectedOption(List<String> options, String selected) {
        List<String> normalizedOptions = new ArrayList<>(options);
        if (StringUtils.isNotBlank(selected) && !normalizedOptions.contains(selected)) {
            normalizedOptions.add(0, selected);
        }
        return normalizedOptions;
    }

    private String selectOption(String requestedValue, String configuredValue, List<String> options) {
        if (StringUtils.isNotBlank(requestedValue) && (CollectionUtils.isEmpty(options) || options.contains(requestedValue))) {
            return requestedValue;
        }
        if (StringUtils.isNotBlank(configuredValue) && (CollectionUtils.isEmpty(options) || options.contains(configuredValue))) {
            return configuredValue;
        }
        return CollectionUtils.isEmpty(options) ? requestedValue : options.get(0);
    }

    private String applyRegex(String value, String grafanaRegex) {
        if (StringUtils.isBlank(value) || StringUtils.isBlank(grafanaRegex)) {
            return value;
        }

        String regex = grafanaRegex;
        if (regex.startsWith("/") && regex.length() > 1) {
            int lastSlash = regex.lastIndexOf('/');
            if (lastSlash > 0) {
                regex = regex.substring(1, lastSlash);
            }
        }

        Matcher matcher = Pattern.compile(regex).matcher(value);
        if (!matcher.find()) {
            return null;
        }
        return matcher.groupCount() >= 1 ? matcher.group(1) : matcher.group();
    }

    private WorkloadMetricsMode resolveWorkloadMetricsMode(String namespace, String gatewayContainerName) {
        String expression = String.format(Locale.ROOT, "count(%s{container=\"%s\", namespace=\"%s\"})",
            CONTAINER_MEMORY_USAGE_METRIC, escapePrometheusLabelValue(gatewayContainerName),
            escapePrometheusLabelValue(resolveDashboardNamespace(namespace)));
        try {
            Double value = extractInstantValue(queryPrometheusInstant(expression, System.currentTimeMillis()));
            if (value != null && value > 0) {
                return WorkloadMetricsMode.CONTAINER;
            }
        } catch (Exception ex) {
            log.debug("Failed to detect container-level workload metrics. Falling back to process metrics.", ex);
        }
        return WorkloadMetricsMode.PROCESS_FALLBACK;
    }

    private String resolveDashboardNamespace(String namespace) {
        return StringUtils.firstNonBlank(namespace, controllerNamespace, HigressConstants.NS_DEFAULT);
    }

    private String resolveSelectorKey() {
        return StringUtils.firstNonBlank(dashboardGatewayLabelKey, controllerIngressClassName,
            HigressConstants.CONTROLLER_INGRESS_CLASS_NAME_DEFAULT);
    }

    private String resolveGatewayName() {
        return StringUtils.firstNonBlank(dashboardGatewayName, resolveSelectorKey() + "-gateway");
    }

    private String resolveGatewayContainerName(String namespace) {
        return resolveContainerName(namespace,
            Arrays.asList(dashboardGatewayContainerName, resolveGatewayName(), "aigateway-gateway", "higress-gateway",
                "istio-proxy"),
            Arrays.asList(CONTAINER_MEMORY_USAGE_METRIC, ISTIO_AGENT_MEMORY_USAGE_METRIC, ENVOY_MEMORY_USAGE_METRIC));
    }

    private String resolveGatewayContainerName() {
        return StringUtils.firstNonBlank(dashboardGatewayContainerName, resolveGatewayName());
    }

    private String resolveCoreContainerName(String namespace) {
        return resolveContainerName(namespace,
            Arrays.asList(dashboardCoreContainerName, resolveSelectorKey() + "-core", "aigateway-core",
                "higress-core"),
            Collections.singletonList(CONTAINER_MEMORY_USAGE_METRIC));
    }

    private String resolveCoreContainerName() {
        return StringUtils.firstNonBlank(dashboardCoreContainerName, resolveSelectorKey() + "-core");
    }

    private String resolveContainerName(String namespace, List<String> candidates, List<String> metricNames) {
        LinkedHashSet<String> normalizedCandidates = new LinkedHashSet<>();
        for (String candidate : candidates) {
            if (StringUtils.isNotBlank(candidate)) {
                normalizedCandidates.add(candidate);
            }
        }
        if (normalizedCandidates.isEmpty()) {
            return "";
        }

        String effectiveNamespace = resolveDashboardNamespace(namespace);
        for (String candidate : normalizedCandidates) {
            for (String metricName : metricNames) {
                if (hasMetricForContainer(metricName, candidate, effectiveNamespace)) {
                    return candidate;
                }
            }
        }
        return normalizedCandidates.iterator().next();
    }

    private boolean hasMetricForContainer(String metricName, String containerName, String namespace) {
        String expression = String.format(Locale.ROOT, "count(%s{container=\"%s\", namespace=\"%s\"})",
            metricName, escapePrometheusLabelValue(containerName), escapePrometheusLabelValue(namespace));
        try {
            Double value = extractInstantValue(queryPrometheusInstant(expression, System.currentTimeMillis()));
            return value != null && value > 0;
        } catch (Exception ex) {
            log.debug("Failed to inspect Prometheus metric {} for container {} in namespace {}.", metricName,
                containerName, namespace, ex);
            return false;
        }
    }

    private String escapePrometheusLabelValue(String value) {
        return StringUtils.replace(StringUtils.defaultString(value), "\"", "\\\"");
    }

    private String resolveQueryExpression(String expression, VariableContext variables, TimeWindow timeWindow) {
        String resolved = expression;
        resolved = resolved.replace("$gateway", StringUtils.defaultString(variables.getGateway()));
        resolved = resolved.replace("$namespace", StringUtils.defaultString(variables.getNamespace()));
        resolved = resolved.replace("$__range", timeWindow.getPrometheusRange());
        return resolved;
    }

    private long parseRelativeDuration(String timeExpression) {
        if (StringUtils.isBlank(timeExpression)) {
            return DEFAULT_DASHBOARD_RANGE_MILLIS;
        }
        if ("now".equals(timeExpression)) {
            return 0L;
        }
        String normalized = timeExpression.startsWith("now-") ? timeExpression.substring(4) : timeExpression;
        if (normalized.length() < 2) {
            return DEFAULT_DASHBOARD_RANGE_MILLIS;
        }
        char unit = normalized.charAt(normalized.length() - 1);
        long amount;
        try {
            amount = Long.parseLong(normalized.substring(0, normalized.length() - 1));
        } catch (NumberFormatException ex) {
            return DEFAULT_DASHBOARD_RANGE_MILLIS;
        }
        switch (unit) {
            case 's':
                return amount * 1000L;
            case 'm':
                return amount * 60_000L;
            case 'h':
                return amount * 60L * 60L * 1000L;
            case 'd':
                return amount * 24L * 60L * 60L * 1000L;
            case 'w':
                return amount * 7L * 24L * 60L * 60L * 1000L;
            default:
                return DEFAULT_DASHBOARD_RANGE_MILLIS;
        }
    }

    private JsonNode queryPrometheusInstant(String expression, long timeMillis) throws IOException {
        Map<String, String> parameters = new LinkedHashMap<>();
        parameters.put("query", expression);
        parameters.put("time", String.valueOf(timeMillis / 1000L));
        return executePrometheusApi(PROMETHEUS_QUERY_PATH, parameters);
    }

    private JsonNode queryPrometheusRange(String expression, TimeWindow timeWindow, int stepSeconds)
        throws IOException {
        Map<String, String> parameters = new LinkedHashMap<>();
        parameters.put("query", expression);
        parameters.put("start", String.valueOf(timeWindow.getFrom() / 1000L));
        parameters.put("end", String.valueOf(timeWindow.getTo() / 1000L));
        parameters.put("step", String.valueOf(stepSeconds));
        return executePrometheusApi(PROMETHEUS_QUERY_RANGE_PATH, parameters);
    }

    private JsonNode executePrometheusApi(String apiPath, Map<String, String> parameters) throws IOException {
        URIBuilder uriBuilder;
        try {
            uriBuilder = new URIBuilder(buildPrometheusApiUrl(apiPath));
        } catch (Exception ex) {
            throw new IOException("Invalid Prometheus API URL.", ex);
        }
        for (Map.Entry<String, String> entry : parameters.entrySet()) {
            uriBuilder.addParameter(entry.getKey(), entry.getValue());
        }
        HttpUriRequest request = RequestBuilder.get().setUri(uriBuilder.toString()).build();
        try (CloseableHttpResponse response = realServerClient.execute(request)) {
            int statusCode = response.getStatusLine().getStatusCode();
            String body = response.getEntity() == null ? "" : IOUtils.toString(response.getEntity().getContent(),
                StandardCharsets.UTF_8);
            if (statusCode / 100 != 2) {
                throw new BusinessException("Prometheus query failed. Status=" + statusCode + ", body=" + body);
            }

            JsonNode rootNode = GrafanaClient.MAPPER.readTree(body);
            if (!"success".equals(rootNode.path("status").asText())) {
                throw new BusinessException("Prometheus query failed. Body=" + body);
            }
            return rootNode.path("data");
        } catch (Exception ex) {
            if (ex instanceof IOException) {
                throw (IOException) ex;
            }
            throw new IOException("Unable to execute Prometheus query.", ex);
        }
    }

    private String buildPrometheusApiUrl(String apiPath) {
        String baseUrl = StringUtils.removeEnd(promDatasourceUrl, "/");
        if (apiPath.startsWith("/")) {
            return baseUrl + apiPath;
        }
        return baseUrl + "/" + apiPath;
    }

    private void appendSeries(ArrayNode renderedSeries, JsonNode dataNode, ObjectNode targetNode, String panelTitle) {
        if (!"matrix".equals(dataNode.path("resultType").asText())) {
            return;
        }

        for (JsonNode resultNode : dataNode.path("result")) {
            ObjectNode seriesNode = renderedSeries.addObject();
            seriesNode.put("name", buildSeriesName(resultNode.path("metric"),
                targetNode.path("legendFormat").asText(null), panelTitle, targetNode.path("refId").asText()));
            seriesNode.set("labels", resultNode.path("metric").deepCopy());
            ArrayNode points = seriesNode.putArray("points");
            for (JsonNode valueNode : resultNode.path("values")) {
                if (valueNode.size() < 2) {
                    continue;
                }
                Double pointValue = parsePrometheusNumber(valueNode.path(1).asText());
                if (pointValue == null) {
                    continue;
                }
                ObjectNode pointNode = points.addObject();
                pointNode.put("time", (long) (valueNode.path(0).asDouble() * 1000L));
                pointNode.put("value", pointValue);
            }
        }
    }

    private String buildSeriesName(JsonNode metricNode, String legendFormat, String panelTitle, String refId) {
        if (StringUtils.isBlank(legendFormat)) {
            return buildAutoSeriesName(metricNode, panelTitle, refId);
        }
        if ("__auto".equals(legendFormat)) {
            return buildAutoSeriesName(metricNode, panelTitle, refId);
        }

        Matcher matcher = VARIABLE_TEMPLATE_PATTERN.matcher(legendFormat);
        StringBuffer buffer = new StringBuffer();
        while (matcher.find()) {
            String labelName = matcher.group(1);
            matcher.appendReplacement(buffer, Matcher.quoteReplacement(metricNode.path(labelName).asText("")));
        }
        matcher.appendTail(buffer);
        return StringUtils.defaultIfBlank(buffer.toString(), buildAutoSeriesName(metricNode, panelTitle, refId));
    }

    private String buildAutoSeriesName(JsonNode metricNode, String panelTitle, String refId) {
        Iterator<Map.Entry<String, JsonNode>> fields = metricNode.fields();
        List<String> labels = new ArrayList<>();
        while (fields.hasNext()) {
            Map.Entry<String, JsonNode> entry = fields.next();
            if ("__name__".equals(entry.getKey())) {
                continue;
            }
            labels.add(entry.getKey() + "=" + entry.getValue().asText());
        }
        if (labels.isEmpty()) {
            return StringUtils.defaultIfBlank(panelTitle, refId);
        }
        return String.join(", ", labels);
    }

    private Double extractInstantValue(JsonNode dataNode) {
        String resultType = dataNode.path("resultType").asText();
        if ("vector".equals(resultType) && dataNode.path("result").isArray() && dataNode.path("result").size() > 0) {
            JsonNode valueNode = dataNode.path("result").path(0).path("value");
            if (valueNode.size() > 1) {
                return parsePrometheusNumber(valueNode.path(1).asText());
            }
        }
        if ("scalar".equals(resultType) && dataNode.path("result").isArray() && dataNode.path("result").size() > 1) {
            return parsePrometheusNumber(dataNode.path("result").path(1).asText());
        }
        return null;
    }

    private Double parsePrometheusNumber(String value) {
        if (StringUtils.isBlank(value) || "NaN".equalsIgnoreCase(value)) {
            return null;
        }
        try {
            return Double.valueOf(value);
        } catch (NumberFormatException ex) {
            return null;
        }
    }

    private TableFrame toTableFrame(JsonNode dataNode, String refId, boolean multiTarget) {
        String valueKey = multiTarget ? VALUE_FIELD + " #" + refId : VALUE_FIELD;
        List<LinkedHashMap<String, Object>> rows = new ArrayList<>();
        if (!"vector".equals(dataNode.path("resultType").asText())) {
            return new TableFrame(rows);
        }

        for (JsonNode resultNode : dataNode.path("result")) {
            LinkedHashMap<String, Object> row = new LinkedHashMap<>();
            Iterator<Map.Entry<String, JsonNode>> fields = resultNode.path("metric").fields();
            while (fields.hasNext()) {
                Map.Entry<String, JsonNode> entry = fields.next();
                if ("__name__".equals(entry.getKey())) {
                    continue;
                }
                row.put(entry.getKey(), entry.getValue().asText());
            }
            Double value = parsePrometheusNumber(resultNode.path("value").path(1).asText());
            row.put(valueKey, value);
            rows.add(row);
        }
        return new TableFrame(rows);
    }

    private List<LinkedHashMap<String, Object>> mergeTableFrames(List<TableFrame> frames, ArrayNode transformations) {
        if (frames.isEmpty()) {
            return new ArrayList<>();
        }
        if (frames.size() == 1) {
            return new ArrayList<>(frames.get(0).getRows());
        }

        String joinField = null;
        for (JsonNode transformation : transformations) {
            if ("joinByField".equals(transformation.path("id").asText())) {
                joinField = transformation.path("options").path("byField").asText(null);
                break;
            }
        }
        if (StringUtils.isBlank(joinField)) {
            return new ArrayList<>(frames.get(0).getRows());
        }

        LinkedHashMap<String, LinkedHashMap<String, Object>> joinedRows = new LinkedHashMap<>();
        for (LinkedHashMap<String, Object> row : frames.get(0).getRows()) {
            Object joinValue = row.get(joinField);
            if (joinValue != null) {
                joinedRows.put(String.valueOf(joinValue), new LinkedHashMap<>(row));
            }
        }
        for (int i = 1; i < frames.size(); i++) {
            Map<String, LinkedHashMap<String, Object>> frameIndex = new LinkedHashMap<>();
            for (LinkedHashMap<String, Object> row : frames.get(i).getRows()) {
                Object joinValue = row.get(joinField);
                if (joinValue != null) {
                    frameIndex.put(String.valueOf(joinValue), row);
                }
            }
            Iterator<Map.Entry<String, LinkedHashMap<String, Object>>> iterator = joinedRows.entrySet().iterator();
            while (iterator.hasNext()) {
                Map.Entry<String, LinkedHashMap<String, Object>> entry = iterator.next();
                LinkedHashMap<String, Object> frameRow = frameIndex.get(entry.getKey());
                if (frameRow == null) {
                    iterator.remove();
                } else {
                    entry.getValue().putAll(frameRow);
                }
            }
        }
        return new ArrayList<>(joinedRows.values());
    }

    private List<LinkedHashMap<String, Object>> applyTableTransformations(List<LinkedHashMap<String, Object>> rows,
        ArrayNode transformations) {
        List<LinkedHashMap<String, Object>> transformedRows = rows;
        List<String> explicitOrder = null;
        Map<String, String> renameMap = Collections.emptyMap();
        for (JsonNode transformation : transformations) {
            String id = transformation.path("id").asText();
            if ("filterFieldsByName".equals(id)) {
                transformedRows = filterTableRows(transformedRows, transformation.path("options"));
            } else if ("organize".equals(id)) {
                Organization organization = buildOrganization(transformation.path("options"));
                transformedRows = organization.apply(transformedRows);
                explicitOrder = organization.getOrderedColumns();
                renameMap = organization.getRenameByName();
            }
        }

        List<String> columns = explicitOrder != null ? explicitOrder : collectColumns(transformedRows);
        if (!renameMap.isEmpty() && explicitOrder == null) {
            columns = renameColumns(columns, renameMap);
        }
        List<LinkedHashMap<String, Object>> reorderedRows = new ArrayList<>();
        for (LinkedHashMap<String, Object> row : transformedRows) {
            LinkedHashMap<String, Object> reorderedRow = new LinkedHashMap<>();
            for (String column : columns) {
                reorderedRow.put(column, row.get(column));
            }
            for (Map.Entry<String, Object> entry : row.entrySet()) {
                if (!reorderedRow.containsKey(entry.getKey())) {
                    reorderedRow.put(entry.getKey(), entry.getValue());
                }
            }
            reorderedRows.add(reorderedRow);
        }
        return reorderedRows;
    }

    private List<LinkedHashMap<String, Object>> filterTableRows(List<LinkedHashMap<String, Object>> rows,
        JsonNode optionsNode) {
        List<String> includedNames = new ArrayList<>();
        for (JsonNode nameNode : optionsNode.path("include").path("names")) {
            includedNames.add(nameNode.asText());
        }
        String includePattern = optionsNode.path("include").path("pattern").asText();
        Pattern compiledPattern = StringUtils.isNotBlank(includePattern) ? Pattern.compile(includePattern) : null;
        if (includedNames.isEmpty() && compiledPattern == null) {
            return rows;
        }

        List<LinkedHashMap<String, Object>> filteredRows = new ArrayList<>();
        for (LinkedHashMap<String, Object> row : rows) {
            LinkedHashMap<String, Object> filteredRow = new LinkedHashMap<>();
            for (Map.Entry<String, Object> entry : row.entrySet()) {
                if (!includedNames.isEmpty() && !includedNames.contains(entry.getKey())) {
                    continue;
                }
                if (compiledPattern != null && !compiledPattern.matcher(entry.getKey()).find()) {
                    continue;
                }
                filteredRow.put(entry.getKey(), entry.getValue());
            }
            filteredRows.add(filteredRow);
        }
        return filteredRows;
    }

    private List<LinkedHashMap<String, Object>> applyTableSorting(List<LinkedHashMap<String, Object>> rows,
        JsonNode sortByNode) {
        if (!sortByNode.isArray() || sortByNode.size() == 0) {
            return rows;
        }

        JsonNode sortNode = sortByNode.path(0);
        String displayName = sortNode.path("displayName").asText(null);
        if (StringUtils.isBlank(displayName)) {
            return rows;
        }
        boolean descending = sortNode.path("desc").asBoolean(false);
        List<LinkedHashMap<String, Object>> sortedRows = new ArrayList<>(rows);
        sortedRows.sort((left, right) -> compareTableValues(left.get(displayName), right.get(displayName)));
        if (descending) {
            Collections.reverse(sortedRows);
        }
        return sortedRows;
    }

    private int compareTableValues(Object left, Object right) {
        if (left == null && right == null) {
            return 0;
        }
        if (left == null) {
            return -1;
        }
        if (right == null) {
            return 1;
        }
        if (left instanceof Number && right instanceof Number) {
            return Double.compare(((Number) left).doubleValue(), ((Number) right).doubleValue());
        }
        return String.valueOf(left).compareToIgnoreCase(String.valueOf(right));
    }

    private List<String> collectColumns(List<LinkedHashMap<String, Object>> rows) {
        LinkedHashSet<String> columns = new LinkedHashSet<>();
        for (LinkedHashMap<String, Object> row : rows) {
            columns.addAll(row.keySet());
        }
        return new ArrayList<>(columns);
    }

    private Organization buildOrganization(JsonNode optionsNode) {
        Map<String, String> renameByName = new LinkedHashMap<>();
        Iterator<Map.Entry<String, JsonNode>> renameFields = optionsNode.path("renameByName").fields();
        while (renameFields.hasNext()) {
            Map.Entry<String, JsonNode> entry = renameFields.next();
            renameByName.put(entry.getKey(), entry.getValue().asText());
        }

        Map<String, Integer> indexByName = new LinkedHashMap<>();
        Iterator<Map.Entry<String, JsonNode>> indexFields = optionsNode.path("indexByName").fields();
        while (indexFields.hasNext()) {
            Map.Entry<String, JsonNode> entry = indexFields.next();
            indexByName.put(entry.getKey(), entry.getValue().asInt());
        }
        return new Organization(renameByName, indexByName);
    }

    private List<String> renameColumns(List<String> columns, Map<String, String> renameByName) {
        List<String> renamedColumns = new ArrayList<>();
        for (String column : columns) {
            renamedColumns.add(renameByName.containsKey(column) ? renameByName.get(column) : column);
        }
        return renamedColumns;
    }

    private Map<String, String> buildDashboardTemplateParameters(String datasourceUid) {
        String selectorKey = resolveSelectorKey();
        String namespace = resolveDashboardNamespace(controllerNamespace);
        String gatewayName = resolveGatewayName();
        String gatewaySelectorValue = namespace + "-" + gatewayName;
        String gatewayContainerName = resolveGatewayContainerName();
        String coreContainerName = resolveCoreContainerName();

        Map<String, String> parameters = new HashMap<>();
        parameters.put(DATASOURCE_UID_PLACEHOLDER, datasourceUid);
        parameters.put(MONITORING_SELECTOR_KEY_PLACEHOLDER, selectorKey);
        parameters.put(MONITORING_NAMESPACE_PLACEHOLDER, namespace);
        parameters.put(MONITORING_GATEWAY_NAME_PLACEHOLDER, gatewayName);
        parameters.put(MONITORING_GATEWAY_SELECTOR_VALUE_PLACEHOLDER, gatewaySelectorValue);
        parameters.put(MONITORING_GATEWAY_CONTAINER_PLACEHOLDER, gatewayContainerName);
        parameters.put(MONITORING_CORE_CONTAINER_PLACEHOLDER, coreContainerName);
        return parameters;
    }

    private HttpUriRequest buildRealServerRequest(HttpServletRequest originalRequest) throws IOException {
        String servletPath = originalRequest.getServletPath();
        if (!servletPath.startsWith(apiBaseUrlObject.getPath())) {
            throw new IllegalArgumentException("Invalid dashboard request path: " + servletPath);
        }

        String url = realServerBaseUrl + servletPath;
        if (originalRequest.getQueryString() != null) {
            url = url + "?" + originalRequest.getQueryString();
        }

        HttpEntity entity = new BufferedHttpEntity(
            new InputStreamEntity(originalRequest.getInputStream(), originalRequest.getContentLength()));
        HttpUriRequest request =
            RequestBuilder.create(originalRequest.getMethod()).setEntity(entity).setUri(url).build();

        Collections.list(originalRequest.getHeaderNames()).stream()
            .filter(name -> !IGNORE_REQUEST_HEADERS.contains(name.toLowerCase()))
            .forEach(name -> request.setHeader(new BasicHeader(name, originalRequest.getHeader(name))));

        return request;
    }

    private void forwardResponse(HttpServletResponse response, HttpResponse forwardResponse) throws IOException {
        Arrays.stream(forwardResponse.getAllHeaders())
            .filter(header -> !IGNORE_RESPONSE_HEADERS.contains(header.getName().toLowerCase()))
            .forEach(header -> response.setHeader(header.getName(), header.getValue()));
        response.setStatus(forwardResponse.getStatusLine().getStatusCode());
        Streams.copy(forwardResponse.getEntity().getContent(), response.getOutputStream(), false);
    }

    private class DashboardInitializer implements Runnable {

        private final boolean overwrite;

        private DashboardInitializer(boolean overwrite) {
            this.overwrite = overwrite;
        }

        @Override
        public void run() {
            while (!Thread.interrupted()) {
                try {
                    initializeDashboard(overwrite);
                    return;
                } catch (Exception ex) {
                    log.error("Error occurs when trying to initialize the dashboard.", ex);
                    try {
                        TimeUnit.SECONDS.sleep(5);
                    } catch (InterruptedException e) {
                        log.warn("Initialization thread is interrupted.", e);
                    }
                }
            }
        }
    }

    private @NotNull DashboardConfiguration getDashboardConfiguration(DashboardType type) {
        DashboardConfiguration configuration = dashboardConfigurations.get(type);
        if (configuration == null) {
            throw new IllegalArgumentException("Invalid dashboard type: " + type);
        }
        return configuration;
    }

    @lombok.Value
    private static class TimeWindow {

        long from;
        long to;
        long defaultRangeMs;

        String getPrometheusRange() {
            long rangeMillis = Math.max(1L, to - from);
            long seconds = Math.max(1L, rangeMillis / 1000L);
            if (seconds % (7L * 24L * 60L * 60L) == 0) {
                return seconds / (7L * 24L * 60L * 60L) + "w";
            }
            if (seconds % (24L * 60L * 60L) == 0) {
                return seconds / (24L * 60L * 60L) + "d";
            }
            if (seconds % (60L * 60L) == 0) {
                return seconds / (60L * 60L) + "h";
            }
            if (seconds % 60L == 0) {
                return seconds / 60L + "m";
            }
            return seconds + "s";
        }
    }

    @lombok.Value
    private static class VariableContext {

        String gateway;
        List<String> gatewayOptions;
        String namespace;
        List<String> namespaceOptions;
        String gatewayContainerName;
        String coreContainerName;
        WorkloadMetricsMode workloadMetricsMode;
    }

    private enum WorkloadMetricsMode {

        CONTAINER,
        PROCESS_FALLBACK
    }

    private static class RowBucket {

        private final String title;
        private final boolean collapsed;
        private final int startY;
        private final ArrayNode panels = GrafanaClient.MAPPER.createArrayNode();

        private RowBucket(String title, boolean collapsed, int startY) {
            this.title = title;
            this.collapsed = collapsed;
            this.startY = startY;
        }

        public String getTitle() {
            return title;
        }

        public boolean isCollapsed() {
            return collapsed;
        }

        public int getStartY() {
            return startY;
        }

        public ArrayNode getPanels() {
            return panels;
        }
    }

    @lombok.Value
    private static class TableFrame {

        List<LinkedHashMap<String, Object>> rows;
    }

    private static class Organization {

        private final Map<String, String> renameByName;
        private final Map<String, Integer> indexByName;
        private List<String> orderedColumns = new ArrayList<>();

        private Organization(Map<String, String> renameByName, Map<String, Integer> indexByName) {
            this.renameByName = renameByName;
            this.indexByName = indexByName;
        }

        public List<LinkedHashMap<String, Object>> apply(List<LinkedHashMap<String, Object>> rows) {
            List<String> originalColumns = new ArrayList<>();
            LinkedHashSet<String> originalColumnSet = new LinkedHashSet<>();
            for (LinkedHashMap<String, Object> row : rows) {
                originalColumnSet.addAll(row.keySet());
            }
            originalColumns.addAll(originalColumnSet);
            originalColumns.sort((left, right) -> {
                Integer leftIndex = indexByName.get(left);
                Integer rightIndex = indexByName.get(right);
                if (leftIndex == null && rightIndex == null) {
                    return 0;
                }
                if (leftIndex == null) {
                    return 1;
                }
                if (rightIndex == null) {
                    return -1;
                }
                return Integer.compare(leftIndex, rightIndex);
            });

            orderedColumns = new ArrayList<>();
            for (String column : originalColumns) {
                orderedColumns.add(renameByName.containsKey(column) ? renameByName.get(column) : column);
            }

            List<LinkedHashMap<String, Object>> organizedRows = new ArrayList<>();
            for (LinkedHashMap<String, Object> row : rows) {
                LinkedHashMap<String, Object> organizedRow = new LinkedHashMap<>();
                for (String column : originalColumns) {
                    String targetColumn = renameByName.containsKey(column) ? renameByName.get(column) : column;
                    organizedRow.put(targetColumn, row.get(column));
                }
                organizedRows.add(organizedRow);
            }
            return organizedRows;
        }

        public List<String> getOrderedColumns() {
            return orderedColumns;
        }

        public Map<String, String> getRenameByName() {
            return renameByName;
        }
    }

    @lombok.Value
    private static class DashboardConfiguration {

        DashboardType type;
        String configKey;
        String resourcePath;
        String raw;
        GrafanaDashboard dashboard;

        public DashboardConfiguration(DashboardType type, String resourcePath) throws IOException {
            this.type = type;
            this.configKey = type == DashboardType.MAIN ? UserConfigKey.DASHBOARD_URL
                : UserConfigKey.DASHBOARD_URL_PREFIX + type.toString().toLowerCase(Locale.ROOT);
            this.resourcePath = resourcePath;
            this.raw = IOUtils.resourceToString(resourcePath, StandardCharsets.UTF_8);
            this.dashboard = GrafanaClient.parseDashboardData(this.raw);
        }
    }
}
