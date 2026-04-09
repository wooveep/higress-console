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
package com.alibaba.higress.console.controller;

import javax.annotation.Resource;
import javax.validation.constraints.NotBlank;

import org.apache.commons.lang3.StringUtils;
import org.springdoc.api.annotations.ParameterObject;
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
import org.springframework.web.bind.annotation.RestController;

import com.alibaba.higress.console.controller.dto.PaginatedResponse;
import com.alibaba.higress.console.controller.dto.Response;
import com.alibaba.higress.console.controller.util.ControllerUtil;
import com.alibaba.higress.console.model.portal.PortalPasswordResetResult;
import com.alibaba.higress.console.service.portal.PortalConsumerService;
import com.alibaba.higress.sdk.exception.ValidationException;
import com.alibaba.higress.sdk.model.CommonPageQuery;
import com.alibaba.higress.sdk.model.PaginatedResult;
import com.alibaba.higress.sdk.model.consumer.Consumer;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
import io.swagger.v3.oas.annotations.tags.Tag;

@RestController("ConsumersController")
@RequestMapping("/v1/consumers")
@Validated
@Tag(name = "Consumer APIs")
public class ConsumersController {

    private PortalConsumerService portalConsumerService;

    @Resource
    public void setPortalConsumerService(PortalConsumerService portalConsumerService) {
        this.portalConsumerService = portalConsumerService;
    }

    @GetMapping
    @Operation(summary = "List consumers")
    @ApiResponses(value = {@ApiResponse(responseCode = "200", description = "Consumers listed successfully"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<PaginatedResponse<Consumer>> list(@ParameterObject CommonPageQuery query) {
        PaginatedResult<Consumer> consumers = portalConsumerService.list(query);
        return ControllerUtil.buildResponseEntity(consumers);
    }

    @PostMapping
    @Operation(summary = "Add a consumer")
    @ApiResponses(value = {@ApiResponse(responseCode = "200", description = "Consumer added successfully"),
        @ApiResponse(responseCode = "400", description = "Consumer data is not valid"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Response<Consumer>> add(@RequestBody Consumer consumer) {
        Consumer created = portalConsumerService.addOrUpdate(consumer, true);
        return ControllerUtil.buildResponseEntity(created);
    }

    @GetMapping(value = "/{name}")
    @Operation(summary = "Get consumer by name")
    @ApiResponses(value = {@ApiResponse(responseCode = "200", description = "Consumer found"),
        @ApiResponse(responseCode = "404", description = "Consumer not found"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Response<Consumer>> query(@PathVariable("name") @NotBlank String name) {
        return ControllerUtil.buildResponseEntity(portalConsumerService.query(name));
    }

    @GetMapping("/departments")
    @Operation(summary = "List departments")
    @ApiResponses(value = {@ApiResponse(responseCode = "200", description = "Departments listed successfully"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Response<java.util.List<String>>> listDepartments() {
        return ControllerUtil.buildResponseEntity(portalConsumerService.listDepartments());
    }

    @PostMapping("/departments")
    @Operation(summary = "Add a department (deprecated)")
    @ApiResponses(value = {@ApiResponse(responseCode = "204", description = "Department request accepted"),
        @ApiResponse(responseCode = "400", description = "Department data is not valid"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Void> addDepartment(@RequestBody DepartmentRequest request) {
        if (request == null || StringUtils.isBlank(request.getName())) {
            throw new ValidationException("Department name cannot be blank.");
        }
        portalConsumerService.addDepartmentCompat(request.getName());
        return ResponseEntity.noContent().build();
    }

    @PutMapping("/{name}")
    @Operation(summary = "Update an existed consumer")
    @ApiResponses(value = {@ApiResponse(responseCode = "200", description = "Consumer updated successfully"),
        @ApiResponse(responseCode = "400", description = "Consumer data is not valid"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Response<Consumer>> put(@PathVariable("name") @NotBlank String name,
        @RequestBody Consumer consumer) {
        if (StringUtils.isBlank(consumer.getName())) {
            consumer.setName(name);
        } else if (!StringUtils.equals(name, consumer.getName())) {
            throw new ValidationException("Consumer name in the URL doesn't match the one in the body.");
        }
        Consumer updated = portalConsumerService.addOrUpdate(consumer, false);
        return ControllerUtil.buildResponseEntity(updated);
    }

    @PatchMapping("/{name}/status")
    @Operation(summary = "Update consumer status")
    @ApiResponses(value = {@ApiResponse(responseCode = "200", description = "Consumer status updated successfully"),
        @ApiResponse(responseCode = "400", description = "Invalid status value"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Response<Consumer>> updateStatus(@PathVariable("name") @NotBlank String name,
        @RequestBody ConsumerStatusRequest request) {
        if (request == null || StringUtils.isBlank(request.getStatus())) {
            throw new ValidationException("status cannot be blank.");
        }
        Consumer updated = portalConsumerService.updateStatus(name, request.getStatus().trim().toLowerCase());
        return ControllerUtil.buildResponseEntity(updated);
    }

    @DeleteMapping("/{name}")
    @Operation(summary = "Delete a consumer")
    @ApiResponses(value = {@ApiResponse(responseCode = "204", description = "Consumer deleted successfully"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Response<Consumer>> delete(@PathVariable("name") @NotBlank String name) {
        portalConsumerService.delete(name);
        return ResponseEntity.noContent().build();
    }

    @PostMapping("/{name}/password/reset")
    @Operation(summary = "Reset consumer portal password")
    @ApiResponses(value = {@ApiResponse(responseCode = "200", description = "Password reset successfully"),
        @ApiResponse(responseCode = "400", description = "Consumer data is not valid"),
        @ApiResponse(responseCode = "500", description = "Internal server error")})
    public ResponseEntity<Response<ResetPasswordResponse>>
        resetPortalPassword(@PathVariable("name") @NotBlank String name) {
        PortalPasswordResetResult result = portalConsumerService.resetPassword(name);
        ResetPasswordResponse response = ResetPasswordResponse.builder().consumerName(result.getConsumerName())
            .tempPassword(result.getTempPassword()).updatedAt(result.getUpdatedAt()).build();
        return ControllerUtil.buildResponseEntity(response);
    }

    public static class DepartmentRequest {

        private String name;

        public String getName() {
            return name;
        }

        public void setName(String name) {
            this.name = name;
        }
    }

    public static class ConsumerStatusRequest {

        private String status;

        public String getStatus() {
            return status;
        }

        public void setStatus(String status) {
            this.status = status;
        }
    }

    @lombok.Data
    @lombok.Builder
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class ResetPasswordResponse {

        private String consumerName;
        private String tempPassword;
        private java.time.LocalDateTime updatedAt;
    }
}
