package com.alibaba.higress.console.controller;

import java.util.List;

import javax.annotation.Resource;
import javax.validation.constraints.NotBlank;

import org.apache.commons.lang3.StringUtils;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.validation.annotation.Validated;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

import com.alibaba.higress.console.controller.dto.Response;
import com.alibaba.higress.console.controller.util.ControllerUtil;
import com.alibaba.higress.console.model.portal.OrgAccountRecord;
import com.alibaba.higress.console.model.portal.OrgDepartmentNode;
import com.alibaba.higress.console.service.portal.PortalOrganizationService;
import com.alibaba.higress.console.service.portal.PortalOrganizationTemplateService;
import com.alibaba.higress.sdk.exception.ValidationException;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
import io.swagger.v3.oas.annotations.tags.Tag;

/**
 * @author Codex
 */
@RestController
@RequestMapping("/v1/org")
@Validated
@Tag(name = "Organization APIs")
public class OrganizationController {

    private PortalOrganizationService portalOrganizationService;

    private PortalOrganizationTemplateService portalOrganizationTemplateService;

    @Resource
    public void setPortalOrganizationService(PortalOrganizationService portalOrganizationService) {
        this.portalOrganizationService = portalOrganizationService;
    }

    @Resource
    public void setPortalOrganizationTemplateService(PortalOrganizationTemplateService portalOrganizationTemplateService) {
        this.portalOrganizationTemplateService = portalOrganizationTemplateService;
    }

    @GetMapping("/departments/tree")
    @Operation(summary = "List department tree")
    public ResponseEntity<Response<List<OrgDepartmentNode>>> listDepartmentTree() {
        return ControllerUtil.buildResponseEntity(portalOrganizationService.listDepartmentTree());
    }

    @PostMapping("/departments")
    @Operation(summary = "Create a department")
    public ResponseEntity<Response<OrgDepartmentNode>> createDepartment(@RequestBody DepartmentRequest request) {
        validateDepartmentRequest(request, true);
        return ControllerUtil.buildResponseEntity(
            portalOrganizationService.createDepartment(toDepartmentMutation(request, true)));
    }

    @PutMapping("/departments/{departmentId}")
    @Operation(summary = "Rename a department")
    public ResponseEntity<Response<OrgDepartmentNode>> updateDepartment(
        @PathVariable("departmentId") @NotBlank String departmentId,
        @RequestBody DepartmentRequest request) {
        validateDepartmentRequest(request, false);
        return ControllerUtil.buildResponseEntity(
            portalOrganizationService.updateDepartment(departmentId, toDepartmentMutation(request, false)));
    }

    @PatchMapping("/departments/{departmentId}/move")
    @Operation(summary = "Move a department")
    public ResponseEntity<Response<OrgDepartmentNode>> moveDepartment(
        @PathVariable("departmentId") @NotBlank String departmentId,
        @RequestBody DepartmentMoveRequest request) {
        if (request == null) {
            throw new ValidationException("request body cannot be null.");
        }
        return ControllerUtil.buildResponseEntity(
            portalOrganizationService.moveDepartment(departmentId, request.getParentDepartmentId()));
    }

    @DeleteMapping("/departments/{departmentId}")
    @Operation(summary = "Delete a department")
    public ResponseEntity<Void> deleteDepartment(@PathVariable("departmentId") @NotBlank String departmentId) {
        portalOrganizationService.deleteDepartment(departmentId);
        return ResponseEntity.noContent().build();
    }

    @GetMapping("/accounts")
    @Operation(summary = "List organization accounts")
    public ResponseEntity<Response<List<OrgAccountRecord>>> listAccounts() {
        return ControllerUtil.buildResponseEntity(portalOrganizationService.listAccounts());
    }

    @PostMapping("/accounts")
    @Operation(summary = "Create an organization account")
    @ApiResponses(value = {
        @ApiResponse(responseCode = "200", description = "Account created successfully"),
        @ApiResponse(responseCode = "400", description = "Invalid account data")
    })
    public ResponseEntity<Response<OrgAccountRecord>> createAccount(@RequestBody AccountRequest request) {
        return ControllerUtil.buildResponseEntity(portalOrganizationService.createAccount(toMutation(request, null)));
    }

    @PutMapping("/accounts/{consumerName}")
    @Operation(summary = "Update an organization account")
    public ResponseEntity<Response<OrgAccountRecord>> updateAccount(
        @PathVariable("consumerName") @NotBlank String consumerName,
        @RequestBody AccountRequest request) {
        return ControllerUtil.buildResponseEntity(
            portalOrganizationService.updateAccount(consumerName, toMutation(request, consumerName)));
    }

    @PatchMapping("/accounts/{consumerName}/assignment")
    @Operation(summary = "Update organization account assignment")
    public ResponseEntity<Response<OrgAccountRecord>> updateAccountAssignment(
        @PathVariable("consumerName") @NotBlank String consumerName,
        @RequestBody AccountAssignmentRequest request) {
        if (request == null) {
            throw new ValidationException("request body cannot be null.");
        }
        return ControllerUtil.buildResponseEntity(
            portalOrganizationService.updateAccountAssignment(consumerName, request.getDepartmentId(),
                request.getParentConsumerName()));
    }

    @PatchMapping("/accounts/{consumerName}/status")
    @Operation(summary = "Update organization account status")
    public ResponseEntity<Response<OrgAccountRecord>> updateAccountStatus(
        @PathVariable("consumerName") @NotBlank String consumerName,
        @RequestBody AccountStatusRequest request) {
        if (request == null || StringUtils.isBlank(request.getStatus())) {
            throw new ValidationException("status cannot be blank.");
        }
        return ControllerUtil.buildResponseEntity(
            portalOrganizationService.updateAccountStatus(consumerName, request.getStatus()));
    }

    @GetMapping("/template")
    @Operation(summary = "Download organization import template")
    public ResponseEntity<byte[]> downloadTemplate() {
        return buildWorkbookResponse("organization-template.xlsx", portalOrganizationTemplateService.downloadTemplate());
    }

    @GetMapping("/export")
    @Operation(summary = "Export organization workbook")
    public ResponseEntity<byte[]> exportOrganization() {
        return buildWorkbookResponse("organization-export.xlsx", portalOrganizationTemplateService.exportOrganization());
    }

    @PostMapping(value = "/import", consumes = MediaType.MULTIPART_FORM_DATA_VALUE)
    @Operation(summary = "Import organization workbook")
    public ResponseEntity<Response<PortalOrganizationTemplateService.ImportResult>> importOrganization(
        @RequestParam("file") MultipartFile file) {
        if (file == null || file.isEmpty()) {
            throw new ValidationException("Import file cannot be empty.");
        }
        try {
            return ControllerUtil.buildResponseEntity(
                portalOrganizationTemplateService.importOrganization(file.getBytes()));
        } catch (java.io.IOException ex) {
            throw new ValidationException("Failed to read import file.");
        }
    }

    private PortalOrganizationService.AccountMutation toMutation(AccountRequest request, String consumerName) {
        if (request == null) {
            throw new ValidationException("request body cannot be null.");
        }
        String normalizedConsumerName = StringUtils.firstNonBlank(request.getConsumerName(), consumerName);
        if (StringUtils.isBlank(normalizedConsumerName)) {
            throw new ValidationException("consumerName cannot be blank.");
        }
        return PortalOrganizationService.AccountMutation.builder()
            .consumerName(normalizedConsumerName)
            .displayName(request.getDisplayName())
            .email(request.getEmail())
            .userLevel(request.getUserLevel())
            .password(request.getPassword())
            .status(request.getStatus())
            .departmentId(request.getDepartmentId())
            .parentConsumerName(request.getParentConsumerName())
            .build();
    }

    private PortalOrganizationService.DepartmentMutation toDepartmentMutation(DepartmentRequest request, boolean creating) {
        if (request == null) {
            throw new ValidationException("request body cannot be null.");
        }
        AdminRequest admin = request.getAdmin();
        return PortalOrganizationService.DepartmentMutation.builder()
            .name(StringUtils.trimToNull(request.getName()))
            .parentDepartmentId(StringUtils.trimToNull(request.getParentDepartmentId()))
            .adminConsumerName(StringUtils.trimToNull(creating ? (admin == null ? null : admin.getConsumerName())
                : StringUtils.firstNonBlank(request.getAdminConsumerName(), admin == null ? null : admin.getConsumerName())))
            .adminDisplayName(StringUtils.trimToNull(admin == null ? null : admin.getDisplayName()))
            .adminEmail(StringUtils.trimToNull(admin == null ? null : admin.getEmail()))
            .adminUserLevel(StringUtils.trimToNull(admin == null ? null : admin.getUserLevel()))
            .adminPassword(StringUtils.trimToNull(admin == null ? null : admin.getPassword()))
            .build();
    }

    private void validateDepartmentRequest(DepartmentRequest request, boolean allowParent) {
        if (request == null) {
            throw new ValidationException("Department request cannot be null.");
        }
        if (!allowParent && StringUtils.isNotBlank(request.getParentDepartmentId())) {
            throw new ValidationException("parentDepartmentId is not allowed in rename request.");
        }
        if (allowParent) {
            if (StringUtils.isBlank(request.getName())) {
                throw new ValidationException("Department name cannot be blank.");
            }
            AdminRequest admin = request.getAdmin();
            if (admin == null || StringUtils.isBlank(admin.getConsumerName())) {
                throw new ValidationException("Department administrator is required.");
            }
            return;
        }
        if (StringUtils.isBlank(request.getName()) && StringUtils.isBlank(request.getAdminConsumerName())) {
            throw new ValidationException("Department update cannot be empty.");
        }
    }

    private ResponseEntity<byte[]> buildWorkbookResponse(String filename, byte[] content) {
        return ResponseEntity.ok()
            .header(HttpHeaders.CONTENT_DISPOSITION, "attachment; filename=\"" + filename + "\"")
            .contentType(MediaType.parseMediaType(
                "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"))
            .body(content);
    }

    @lombok.Data
    public static class DepartmentRequest {

        private String name;
        private String parentDepartmentId;
        private String adminConsumerName;
        private AdminRequest admin;
    }

    @lombok.Data
    public static class DepartmentMoveRequest {

        private String parentDepartmentId;
    }

    @lombok.Data
    public static class AccountRequest {

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
    public static class AccountAssignmentRequest {

        private String departmentId;
        private String parentConsumerName;
    }

    @lombok.Data
    public static class AccountStatusRequest {

        private String status;
    }

    @lombok.Data
    public static class AdminRequest {

        private String consumerName;
        private String displayName;
        private String email;
        private String userLevel;
        private String password;
    }
}
