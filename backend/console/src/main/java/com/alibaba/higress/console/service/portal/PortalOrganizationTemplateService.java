package com.alibaba.higress.console.service.portal;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Collections;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

import javax.annotation.Resource;

import org.apache.commons.lang3.BooleanUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.poi.ss.usermodel.Cell;
import org.apache.poi.ss.usermodel.DataFormatter;
import org.apache.poi.ss.usermodel.Row;
import org.apache.poi.ss.usermodel.Sheet;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.OrgAccountRecord;
import com.alibaba.higress.console.model.portal.OrgDepartmentNode;
import com.alibaba.higress.sdk.exception.ValidationException;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Service
public class PortalOrganizationTemplateService {

    private static final String SHEET_DEPARTMENTS = "Departments";
    private static final String SHEET_ACCOUNTS = "Accounts";

    private final DataFormatter dataFormatter = new DataFormatter();

    @Resource
    private PortalOrganizationService portalOrganizationService;

    @Value("${higress.portal.org.default-password:}")
    private String defaultOrgPassword;

    public byte[] downloadTemplate() {
        try (XSSFWorkbook workbook = new XSSFWorkbook()) {
            Sheet departments = workbook.createSheet(SHEET_DEPARTMENTS);
            writeRow(departments, 0, "departmentName", "parentDepartmentPath", "adminConsumerName");

            Sheet accounts = workbook.createSheet(SHEET_ACCOUNTS);
            writeRow(accounts, 0, "consumerName", "displayName", "email", "departmentPath", "userLevel",
                "parentConsumerName", "isDepartmentAdmin");

            return toBytes(workbook);
        } catch (IOException ex) {
            throw new IllegalStateException("Failed to generate organization template.", ex);
        }
    }

    public byte[] exportOrganization() {
        try (XSSFWorkbook workbook = new XSSFWorkbook()) {
            Sheet departments = workbook.createSheet(SHEET_DEPARTMENTS);
            writeRow(departments, 0, "departmentName", "parentDepartmentPath", "adminConsumerName");

            List<DepartmentExportRow> departmentRows = flattenDepartments(portalOrganizationService.listDepartmentTree(),
                null);
            for (int i = 0; i < departmentRows.size(); i++) {
                DepartmentExportRow row = departmentRows.get(i);
                writeRow(departments, i + 1, row.getDepartmentName(), row.getParentDepartmentPath(),
                    row.getAdminConsumerName());
            }

            Sheet accounts = workbook.createSheet(SHEET_ACCOUNTS);
            writeRow(accounts, 0, "consumerName", "displayName", "email", "departmentPath", "userLevel",
                "parentConsumerName", "isDepartmentAdmin");
            List<OrgAccountRecord> accountsData = portalOrganizationService.listAccounts();
            for (int i = 0; i < accountsData.size(); i++) {
                OrgAccountRecord account = accountsData.get(i);
                writeRow(accounts, i + 1, account.getConsumerName(), account.getDisplayName(), account.getEmail(),
                    account.getDepartmentPath(), account.getUserLevel(), account.getParentConsumerName(),
                    BooleanUtils.isTrue(account.getIsDepartmentAdmin()) ? "true" : "false");
            }
            return toBytes(workbook);
        } catch (IOException ex) {
            throw new IllegalStateException("Failed to export organization workbook.", ex);
        }
    }

    public ImportResult importOrganization(byte[] fileBytes) {
        if (fileBytes == null || fileBytes.length == 0) {
            throw new ValidationException("Import file cannot be empty.");
        }
        if (StringUtils.isBlank(defaultOrgPassword)) {
            throw new ValidationException("Default organization password is not configured.");
        }

        ImportPayload payload;
        try (InputStream inputStream = new ByteArrayInputStream(fileBytes);
            XSSFWorkbook workbook = new XSSFWorkbook(inputStream)) {
            payload = parseWorkbook(workbook);
        } catch (IOException ex) {
            throw new ValidationException("Invalid xlsx file.");
        }

        Map<String, AccountSheetRow> accountRowMap = payload.getAccounts().stream()
            .collect(LinkedHashMap::new, (map, item) -> map.put(item.getConsumerName(), item), Map::putAll);

        List<OrgDepartmentNode> beforeDepartments = portalOrganizationService.listDepartmentTree();
        Map<String, OrgDepartmentNode> departmentPathMap = buildDepartmentPathMap(beforeDepartments, null);
        List<OrgAccountRecord> beforeAccounts = portalOrganizationService.listAccounts();
        Map<String, OrgAccountRecord> accountMap = beforeAccounts.stream()
            .collect(LinkedHashMap::new, (map, item) -> map.put(item.getConsumerName(), item), Map::putAll);

        int createdDepartments = 0;
        int updatedDepartments = 0;
        List<DepartmentSheetRow> orderedDepartments = new ArrayList<>(payload.getDepartments());
        orderedDepartments.sort(Comparator.comparingInt(item -> depth(item.getDepartmentPath())));
        for (DepartmentSheetRow row : orderedDepartments) {
            AccountSheetRow adminRow = accountRowMap.get(row.getAdminConsumerName());
            PortalOrganizationService.DepartmentMutation mutation = PortalOrganizationService.DepartmentMutation.builder()
                .name(row.getDepartmentName())
                .parentDepartmentId(resolveDepartmentId(departmentPathMap, row.getParentDepartmentPath()))
                .adminConsumerName(row.getAdminConsumerName())
                .adminDisplayName(adminRow == null ? row.getAdminConsumerName() : adminRow.getDisplayName())
                .adminEmail(adminRow == null ? null : adminRow.getEmail())
                .adminUserLevel(adminRow == null ? "normal" : adminRow.getUserLevel())
                .adminPassword(defaultOrgPassword)
                .build();
            OrgDepartmentNode existing = departmentPathMap.get(row.getDepartmentPath());
            if (existing == null) {
                portalOrganizationService.createDepartment(mutation);
                createdDepartments++;
            } else {
                portalOrganizationService.updateDepartment(existing.getDepartmentId(),
                    PortalOrganizationService.DepartmentMutation.builder()
                        .name(row.getDepartmentName())
                        .adminConsumerName(row.getAdminConsumerName())
                        .build());
                updatedDepartments++;
            }
            departmentPathMap = buildDepartmentPathMap(portalOrganizationService.listDepartmentTree(), null);
        }

        int createdAccounts = 0;
        int updatedAccounts = 0;
        for (AccountSheetRow row : payload.getAccounts()) {
            PortalOrganizationService.AccountMutation mutation = PortalOrganizationService.AccountMutation.builder()
                .consumerName(row.getConsumerName())
                .displayName(row.getDisplayName())
                .email(row.getEmail())
                .departmentId(resolveDepartmentId(departmentPathMap, row.getDepartmentPath()))
                .parentConsumerName(row.getParentConsumerName())
                .userLevel(row.getUserLevel())
                .password(defaultOrgPassword)
                .build();
            if (accountMap.containsKey(row.getConsumerName())) {
                portalOrganizationService.updateAccount(row.getConsumerName(), mutation);
                updatedAccounts++;
            } else {
                portalOrganizationService.createAccount(mutation);
                createdAccounts++;
            }
        }
        return ImportResult.builder()
            .createdDepartments(createdDepartments)
            .updatedDepartments(updatedDepartments)
            .createdAccounts(createdAccounts)
            .updatedAccounts(updatedAccounts)
            .build();
    }

    private ImportPayload parseWorkbook(XSSFWorkbook workbook) {
        Sheet departmentsSheet = workbook.getSheet(SHEET_DEPARTMENTS);
        Sheet accountsSheet = workbook.getSheet(SHEET_ACCOUNTS);
        if (departmentsSheet == null || accountsSheet == null) {
            throw new ValidationException("Workbook must contain Departments and Accounts sheets.");
        }

        List<DepartmentSheetRow> departments = parseDepartments(departmentsSheet);
        List<AccountSheetRow> accounts = parseAccounts(accountsSheet);
        validateImportPayload(departments, accounts);
        return new ImportPayload(departments, accounts);
    }

    private List<DepartmentSheetRow> parseDepartments(Sheet sheet) {
        List<DepartmentSheetRow> result = new ArrayList<>();
        Set<String> seenPaths = new LinkedHashSet<>();
        for (int index = 1; index <= sheet.getLastRowNum(); index++) {
            Row row = sheet.getRow(index);
            if (row == null || isRowEmpty(row)) {
                continue;
            }
            String departmentName = getCellText(row, 0);
            String parentDepartmentPath = getCellText(row, 1);
            String adminConsumerName = getCellText(row, 2);
            if (StringUtils.isBlank(departmentName) || StringUtils.isBlank(adminConsumerName)) {
                throw new ValidationException(String.format("Departments row %d is missing required fields.", index + 1));
            }
            String departmentPath = buildDepartmentPath(parentDepartmentPath, departmentName);
            if (!seenPaths.add(departmentPath)) {
                throw new ValidationException("Duplicate department path in template: " + departmentPath);
            }
            result.add(DepartmentSheetRow.builder()
                .departmentName(departmentName)
                .parentDepartmentPath(parentDepartmentPath)
                .departmentPath(departmentPath)
                .adminConsumerName(adminConsumerName)
                .build());
        }
        return result;
    }

    private List<AccountSheetRow> parseAccounts(Sheet sheet) {
        List<AccountSheetRow> result = new ArrayList<>();
        Set<String> seenConsumers = new LinkedHashSet<>();
        for (int index = 1; index <= sheet.getLastRowNum(); index++) {
            Row row = sheet.getRow(index);
            if (row == null || isRowEmpty(row)) {
                continue;
            }
            String consumerName = getCellText(row, 0);
            if (StringUtils.isBlank(consumerName)) {
                throw new ValidationException(String.format("Accounts row %d is missing consumerName.", index + 1));
            }
            if (!seenConsumers.add(consumerName)) {
                throw new ValidationException("Duplicate account in template: " + consumerName);
            }
            result.add(AccountSheetRow.builder()
                .consumerName(consumerName)
                .displayName(getCellText(row, 1))
                .email(getCellText(row, 2))
                .departmentPath(getCellText(row, 3))
                .userLevel(StringUtils.defaultIfBlank(getCellText(row, 4), "normal"))
                .parentConsumerName(getCellText(row, 5))
                .departmentAdmin(BooleanUtils.toBoolean(getCellText(row, 6)))
                .build());
        }
        return result;
    }

    private void validateImportPayload(List<DepartmentSheetRow> departments, List<AccountSheetRow> accounts) {
        if (departments.isEmpty()) {
            throw new ValidationException("Departments sheet cannot be empty.");
        }
        if (accounts.isEmpty()) {
            throw new ValidationException("Accounts sheet cannot be empty.");
        }

        Map<String, DepartmentSheetRow> departmentsByPath = new LinkedHashMap<>();
        Set<String> adminConsumers = new LinkedHashSet<>();
        for (DepartmentSheetRow row : departments) {
            departmentsByPath.put(row.getDepartmentPath(), row);
            if (!adminConsumers.add(row.getAdminConsumerName())) {
                throw new ValidationException("One account cannot manage multiple departments: "
                    + row.getAdminConsumerName());
            }
            if (StringUtils.isNotBlank(row.getParentDepartmentPath())
                && !departmentsByPath.containsKey(row.getParentDepartmentPath())) {
                throw new ValidationException("Parent department path not found: " + row.getParentDepartmentPath());
            }
        }

        Map<String, AccountSheetRow> accountsByConsumer = new LinkedHashMap<>();
        for (AccountSheetRow row : accounts) {
            accountsByConsumer.put(row.getConsumerName(), row);
            if (StringUtils.isNotBlank(row.getDepartmentPath())
                && !departmentsByPath.containsKey(row.getDepartmentPath())) {
                throw new ValidationException("Account department path not found: " + row.getDepartmentPath());
            }
        }

        for (DepartmentSheetRow row : departments) {
            AccountSheetRow accountRow = accountsByConsumer.get(row.getAdminConsumerName());
            if (accountRow == null) {
                throw new ValidationException("Department administrator account is missing from Accounts sheet: "
                    + row.getAdminConsumerName());
            }
            if (!BooleanUtils.isTrue(accountRow.getDepartmentAdmin())) {
                throw new ValidationException("Department administrator must be marked as isDepartmentAdmin=true: "
                    + row.getAdminConsumerName());
            }
            if (!StringUtils.equals(accountRow.getDepartmentPath(), row.getDepartmentPath())) {
                throw new ValidationException("Department administrator department mismatch: "
                    + row.getAdminConsumerName());
            }
        }

        for (AccountSheetRow row : accounts) {
            if (BooleanUtils.isTrue(row.getDepartmentAdmin()) && !adminConsumers.contains(row.getConsumerName())) {
                throw new ValidationException("Account is marked as department admin but not declared in Departments sheet: "
                    + row.getConsumerName());
            }
        }
    }

    private List<DepartmentExportRow> flattenDepartments(Collection<OrgDepartmentNode> nodes, String parentPath) {
        if (nodes == null || nodes.isEmpty()) {
            return Collections.emptyList();
        }
        List<DepartmentExportRow> result = new ArrayList<>();
        for (OrgDepartmentNode node : nodes) {
            String currentPath = StringUtils.isBlank(parentPath) ? node.getName() : parentPath + " / " + node.getName();
            result.add(DepartmentExportRow.builder()
                .departmentName(node.getName())
                .parentDepartmentPath(parentPath)
                .adminConsumerName(node.getAdminConsumerName())
                .departmentPath(currentPath)
                .build());
            result.addAll(flattenDepartments(node.getChildren(), currentPath));
        }
        return result;
    }

    private Map<String, OrgDepartmentNode> buildDepartmentPathMap(Collection<OrgDepartmentNode> nodes, String parentPath) {
        Map<String, OrgDepartmentNode> result = new LinkedHashMap<>();
        if (nodes == null) {
            return result;
        }
        for (OrgDepartmentNode node : nodes) {
            String currentPath = StringUtils.isBlank(parentPath) ? node.getName() : parentPath + " / " + node.getName();
            result.put(currentPath, node);
            result.putAll(buildDepartmentPathMap(node.getChildren(), currentPath));
        }
        return result;
    }

    private String resolveDepartmentId(Map<String, OrgDepartmentNode> departmentPathMap, String departmentPath) {
        String normalizedPath = StringUtils.trimToNull(departmentPath);
        if (normalizedPath == null) {
            return null;
        }
        OrgDepartmentNode node = departmentPathMap.get(normalizedPath);
        if (node == null) {
            throw new ValidationException("Department path not found: " + normalizedPath);
        }
        return node.getDepartmentId();
    }

    private String buildDepartmentPath(String parentDepartmentPath, String departmentName) {
        String normalizedParentPath = StringUtils.trimToNull(parentDepartmentPath);
        return normalizedParentPath == null ? departmentName : normalizedParentPath + " / " + departmentName;
    }

    private int depth(String departmentPath) {
        if (StringUtils.isBlank(departmentPath)) {
            return 0;
        }
        return StringUtils.countMatches(departmentPath, "/") + 1;
    }

    private boolean isRowEmpty(Row row) {
        for (int index = 0; index < 7; index++) {
            if (StringUtils.isNotBlank(getCellText(row, index))) {
                return false;
            }
        }
        return true;
    }

    private String getCellText(Row row, int index) {
        if (row == null) {
            return null;
        }
        Cell cell = row.getCell(index, Row.MissingCellPolicy.RETURN_BLANK_AS_NULL);
        if (cell == null) {
            return null;
        }
        return StringUtils.trimToNull(dataFormatter.formatCellValue(cell));
    }

    private void writeRow(Sheet sheet, int rowIndex, String... values) {
        Row row = sheet.createRow(rowIndex);
        for (int index = 0; index < values.length; index++) {
            row.createCell(index).setCellValue(StringUtils.defaultString(values[index]));
        }
    }

    private byte[] toBytes(XSSFWorkbook workbook) throws IOException {
        try (ByteArrayOutputStream outputStream = new ByteArrayOutputStream()) {
            workbook.write(outputStream);
            return outputStream.toByteArray();
        }
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    private static class DepartmentExportRow {

        private String departmentName;
        private String parentDepartmentPath;
        private String adminConsumerName;
        private String departmentPath;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    private static class DepartmentSheetRow {

        private String departmentName;
        private String parentDepartmentPath;
        private String departmentPath;
        private String adminConsumerName;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    private static class AccountSheetRow {

        private String consumerName;
        private String displayName;
        private String email;
        private String departmentPath;
        private String userLevel;
        private String parentConsumerName;
        private Boolean departmentAdmin;
    }

    @Data
    @NoArgsConstructor
    @AllArgsConstructor
    private static class ImportPayload {

        private List<DepartmentSheetRow> departments;
        private List<AccountSheetRow> accounts;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ImportResult {

        private int createdDepartments;
        private int updatedDepartments;
        private int createdAccounts;
        private int updatedAccounts;
    }
}
