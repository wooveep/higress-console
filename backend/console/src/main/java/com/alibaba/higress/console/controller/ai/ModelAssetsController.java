package com.alibaba.higress.console.controller.ai;

import java.util.List;

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
import com.alibaba.higress.console.model.portal.ModelAssetBindingRecord;
import com.alibaba.higress.console.model.portal.ModelAssetOptionsRecord;
import com.alibaba.higress.console.model.portal.ModelAssetRecord;
import com.alibaba.higress.console.model.portal.ModelBindingPriceVersionRecord;
import com.alibaba.higress.console.service.portal.PortalModelAssetJdbcService;
import com.alibaba.higress.sdk.model.CommonPageQuery;
import com.alibaba.higress.sdk.model.PaginatedResult;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;

@RestController
@RequestMapping("/v1/ai/model-assets")
@Validated
@Tag(name = "Model Asset APIs")
public class ModelAssetsController {

    private PortalModelAssetJdbcService portalModelAssetJdbcService;

    @Resource
    public void setPortalModelAssetJdbcService(PortalModelAssetJdbcService portalModelAssetJdbcService) {
        this.portalModelAssetJdbcService = portalModelAssetJdbcService;
    }

    @GetMapping
    @Operation(summary = "List model assets")
    public ResponseEntity<PaginatedResponse<ModelAssetRecord>> list(@RequestParam(required = false) CommonPageQuery query) {
        PaginatedResult<ModelAssetRecord> result = portalModelAssetJdbcService.listAssets(query);
        return ControllerUtil.buildResponseEntity(result);
    }

    @GetMapping("/options")
    @Operation(summary = "Get model asset preset options")
    public ResponseEntity<Response<ModelAssetOptionsRecord>> queryOptions() {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.queryOptions());
    }

    @PostMapping
    @Operation(summary = "Create a model asset")
    public ResponseEntity<Response<ModelAssetRecord>> create(@RequestBody ModelAssetRecord request) {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.createAsset(request));
    }

    @GetMapping("/{assetId}")
    @Operation(summary = "Get model asset detail")
    public ResponseEntity<Response<ModelAssetRecord>> query(@PathVariable("assetId") @NotBlank String assetId) {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.queryAsset(assetId));
    }

    @PutMapping("/{assetId}")
    @Operation(summary = "Update a model asset")
    public ResponseEntity<Response<ModelAssetRecord>> update(@PathVariable("assetId") @NotBlank String assetId,
        @RequestBody ModelAssetRecord request) {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.updateAsset(assetId, request));
    }

    @PostMapping("/{assetId}/bindings")
    @Operation(summary = "Create a model binding")
    public ResponseEntity<Response<ModelAssetBindingRecord>> createBinding(
        @PathVariable("assetId") @NotBlank String assetId,
        @RequestBody ModelAssetBindingRecord request) {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.createBinding(assetId, request));
    }

    @PutMapping("/{assetId}/bindings/{bindingId}")
    @Operation(summary = "Update a model binding")
    public ResponseEntity<Response<ModelAssetBindingRecord>> updateBinding(
        @PathVariable("assetId") @NotBlank String assetId,
        @PathVariable("bindingId") @NotBlank String bindingId,
        @RequestBody ModelAssetBindingRecord request) {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.updateBinding(assetId, bindingId, request));
    }

    @PostMapping("/{assetId}/bindings/{bindingId}/publish")
    @Operation(summary = "Publish a model binding")
    public ResponseEntity<Response<ModelAssetBindingRecord>> publishBinding(
        @PathVariable("assetId") @NotBlank String assetId,
        @PathVariable("bindingId") @NotBlank String bindingId) {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.publishBinding(assetId, bindingId));
    }

    @PostMapping("/{assetId}/bindings/{bindingId}/unpublish")
    @Operation(summary = "Unpublish a model binding")
    public ResponseEntity<Response<ModelAssetBindingRecord>> unpublishBinding(
        @PathVariable("assetId") @NotBlank String assetId,
        @PathVariable("bindingId") @NotBlank String bindingId) {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.unpublishBinding(assetId, bindingId));
    }

    @GetMapping("/{assetId}/bindings/{bindingId}/price-versions")
    @Operation(summary = "List model price versions")
    public ResponseEntity<Response<List<ModelBindingPriceVersionRecord>>> listPriceVersions(
        @PathVariable("assetId") @NotBlank String assetId,
        @PathVariable("bindingId") @NotBlank String bindingId) {
        return ControllerUtil.buildResponseEntity(portalModelAssetJdbcService.listPriceVersions(assetId, bindingId));
    }

    @PostMapping("/{assetId}/bindings/{bindingId}/price-versions/{versionId}/restore")
    @Operation(summary = "Restore a model price version to binding draft")
    public ResponseEntity<Response<ModelAssetBindingRecord>> restorePriceVersion(
        @PathVariable("assetId") @NotBlank String assetId,
        @PathVariable("bindingId") @NotBlank String bindingId,
        @PathVariable("versionId") Long versionId) {
        return ControllerUtil.buildResponseEntity(
            portalModelAssetJdbcService.restorePriceVersion(assetId, bindingId, versionId));
    }
}
