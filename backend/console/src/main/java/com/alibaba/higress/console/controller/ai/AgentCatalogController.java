package com.alibaba.higress.console.controller.ai;

import javax.annotation.Resource;
import javax.validation.constraints.NotBlank;

import org.springframework.http.ResponseEntity;
import org.springframework.validation.annotation.Validated;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import com.alibaba.higress.console.controller.dto.PaginatedResponse;
import com.alibaba.higress.console.controller.dto.Response;
import com.alibaba.higress.console.controller.util.ControllerUtil;
import com.alibaba.higress.console.model.portal.AgentCatalogOptionsRecord;
import com.alibaba.higress.console.model.portal.AgentCatalogRecord;
import com.alibaba.higress.console.service.portal.PortalAgentCatalogJdbcService;
import com.alibaba.higress.sdk.model.CommonPageQuery;
import com.alibaba.higress.sdk.model.PaginatedResult;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;

@RestController
@RequestMapping("/v1/ai/agent-catalog")
@Validated
@Tag(name = "Agent Catalog APIs")
public class AgentCatalogController {

    private PortalAgentCatalogJdbcService portalAgentCatalogJdbcService;

    @Resource
    public void setPortalAgentCatalogJdbcService(PortalAgentCatalogJdbcService portalAgentCatalogJdbcService) {
        this.portalAgentCatalogJdbcService = portalAgentCatalogJdbcService;
    }

    @GetMapping
    @Operation(summary = "List agent catalog assets")
    public ResponseEntity<PaginatedResponse<AgentCatalogRecord>> list(@RequestParam(required = false) CommonPageQuery query) {
        PaginatedResult<AgentCatalogRecord> result = portalAgentCatalogJdbcService.listAssets(query);
        return ControllerUtil.buildResponseEntity(result);
    }

    @GetMapping("/options")
    @Operation(summary = "Get MCP server options for agent catalog")
    public ResponseEntity<Response<AgentCatalogOptionsRecord>> queryOptions() {
        return ControllerUtil.buildResponseEntity(portalAgentCatalogJdbcService.queryOptions());
    }

    @PostMapping
    @Operation(summary = "Create an agent catalog asset")
    public ResponseEntity<Response<AgentCatalogRecord>> create(@RequestBody AgentCatalogRecord request) {
        return ControllerUtil.buildResponseEntity(portalAgentCatalogJdbcService.createAsset(request));
    }

    @GetMapping("/{agentId}")
    @Operation(summary = "Get agent catalog detail")
    public ResponseEntity<Response<AgentCatalogRecord>> query(@PathVariable("agentId") @NotBlank String agentId) {
        return ControllerUtil.buildResponseEntity(portalAgentCatalogJdbcService.queryAsset(agentId));
    }

    @PutMapping("/{agentId}")
    @Operation(summary = "Update an agent catalog asset")
    public ResponseEntity<Response<AgentCatalogRecord>> update(@PathVariable("agentId") @NotBlank String agentId,
        @RequestBody AgentCatalogRecord request) {
        return ControllerUtil.buildResponseEntity(portalAgentCatalogJdbcService.updateAsset(agentId, request));
    }

    @PostMapping("/{agentId}/publish")
    @Operation(summary = "Publish an agent catalog asset")
    public ResponseEntity<Response<AgentCatalogRecord>> publish(@PathVariable("agentId") @NotBlank String agentId) {
        return ControllerUtil.buildResponseEntity(portalAgentCatalogJdbcService.publishAsset(agentId));
    }

    @PostMapping("/{agentId}/unpublish")
    @Operation(summary = "Unpublish an agent catalog asset")
    public ResponseEntity<Response<AgentCatalogRecord>> unpublish(@PathVariable("agentId") @NotBlank String agentId) {
        return ControllerUtil.buildResponseEntity(portalAgentCatalogJdbcService.unpublishAsset(agentId));
    }
}
