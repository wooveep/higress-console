package com.alibaba.higress.console.controller;

import java.util.List;

import javax.annotation.Resource;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.validation.annotation.Validated;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import com.alibaba.higress.console.controller.dto.Response;
import com.alibaba.higress.console.controller.util.ControllerUtil;
import com.alibaba.higress.console.model.portal.PortalDepartmentBillRecord;
import com.alibaba.higress.console.model.portal.PortalUsageEventRecord;
import com.alibaba.higress.console.model.portal.PortalUsageStatRecord;
import com.alibaba.higress.console.service.portal.PortalUsageStatsService;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
import io.swagger.v3.oas.annotations.tags.Tag;

@RestController("PortalStatsController")
@RequestMapping("/v1/portal/stats")
@Validated
@Tag(name = "Portal Stats APIs")
public class PortalStatsController {

    private PortalUsageStatsService portalUsageStatsService;

    @Resource
    public void setPortalUsageStatsService(PortalUsageStatsService portalUsageStatsService) {
        this.portalUsageStatsService = portalUsageStatsService;
    }

    @GetMapping("/usage")
    @Operation(summary = "List usage stats grouped by consumer and model")
    @ApiResponses(value = {@ApiResponse(responseCode = "200", description = "Usage stats listed successfully"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Response<List<PortalUsageStatRecord>>> listUsage(
        @RequestParam(required = false) Long from, @RequestParam(required = false) Long to) {
        try {
            List<PortalUsageStatRecord> result = portalUsageStatsService.listUsage(from, to);
            return ControllerUtil.buildResponseEntity(result);
        } catch (IllegalStateException ex) {
            return ResponseEntity.status(HttpStatus.SERVICE_UNAVAILABLE).body(Response.failure(ex.getMessage()));
        }
    }

    @GetMapping("/usage-events")
    @Operation(summary = "List request-level usage events")
    public ResponseEntity<Response<List<PortalUsageEventRecord>>> listUsageEvents(
        @RequestParam(required = false) Long from, @RequestParam(required = false) Long to,
        @RequestParam(required = false) String consumerName, @RequestParam(required = false) String departmentId,
        @RequestParam(required = false) Boolean includeChildren, @RequestParam(required = false) String apiKeyId,
        @RequestParam(required = false) String modelId, @RequestParam(required = false) String routeName,
        @RequestParam(required = false) String requestStatus, @RequestParam(required = false) String usageStatus,
        @RequestParam(required = false) Integer pageNum, @RequestParam(required = false) Integer pageSize) {
        try {
            List<PortalUsageEventRecord> result = portalUsageStatsService.listUsageEvents(from, to, consumerName,
                departmentId, includeChildren, apiKeyId, modelId, routeName, requestStatus, usageStatus, pageNum,
                pageSize);
            return ControllerUtil.buildResponseEntity(result);
        } catch (IllegalStateException ex) {
            return ResponseEntity.status(HttpStatus.SERVICE_UNAVAILABLE).body(Response.failure(ex.getMessage()));
        }
    }

    @GetMapping("/department-bills")
    @Operation(summary = "List department bill summaries")
    public ResponseEntity<Response<List<PortalDepartmentBillRecord>>> listDepartmentBills(
        @RequestParam(required = false) Long from, @RequestParam(required = false) Long to,
        @RequestParam(required = false) String departmentId, @RequestParam(required = false) Boolean includeChildren) {
        try {
            List<PortalDepartmentBillRecord> result = portalUsageStatsService.listDepartmentBills(from, to,
                departmentId, includeChildren);
            return ControllerUtil.buildResponseEntity(result);
        } catch (IllegalStateException ex) {
            return ResponseEntity.status(HttpStatus.SERVICE_UNAVAILABLE).body(Response.failure(ex.getMessage()));
        }
    }
}
