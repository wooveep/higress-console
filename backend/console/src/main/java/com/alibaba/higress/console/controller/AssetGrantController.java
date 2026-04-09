package com.alibaba.higress.console.controller;

import java.util.Collections;
import java.util.List;

import javax.annotation.Resource;
import javax.validation.constraints.NotBlank;

import org.springframework.http.ResponseEntity;
import org.springframework.validation.annotation.Validated;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.alibaba.higress.console.controller.dto.Response;
import com.alibaba.higress.console.controller.util.ControllerUtil;
import com.alibaba.higress.console.model.portal.AssetGrantRecord;
import com.alibaba.higress.console.service.portal.PortalOrganizationService;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;

/**
 * @author Codex
 */
@RestController
@RequestMapping("/v1/assets")
@Validated
@Tag(name = "Asset Grant APIs")
public class AssetGrantController {

    private PortalOrganizationService portalOrganizationService;

    @Resource
    public void setPortalOrganizationService(PortalOrganizationService portalOrganizationService) {
        this.portalOrganizationService = portalOrganizationService;
    }

    @GetMapping("/{type}/{assetId}/grants")
    @Operation(summary = "List asset grants")
    public ResponseEntity<Response<List<AssetGrantRecord>>> listGrants(
        @PathVariable("type") @NotBlank String type,
        @PathVariable("assetId") @NotBlank String assetId) {
        return ControllerUtil.buildResponseEntity(portalOrganizationService.listGrants(type, assetId));
    }

    @PutMapping("/{type}/{assetId}/grants")
    @Operation(summary = "Replace asset grants")
    public ResponseEntity<Response<List<AssetGrantRecord>>> replaceGrants(
        @PathVariable("type") @NotBlank String type,
        @PathVariable("assetId") @NotBlank String assetId,
        @RequestBody AssetGrantReplaceRequest request) {
        List<AssetGrantRecord> grants = request == null ? Collections.emptyList() : request.getGrants();
        return ControllerUtil.buildResponseEntity(portalOrganizationService.replaceGrants(type, assetId, grants));
    }

    @lombok.Data
    public static class AssetGrantReplaceRequest {

        private List<AssetGrantRecord> grants;
    }
}
