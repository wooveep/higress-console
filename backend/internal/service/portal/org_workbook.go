package portal

import (
	"bytes"
	"context"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

type OrgImportResult struct {
	CreatedDepartments int `json:"createdDepartments"`
	UpdatedDepartments int `json:"updatedDepartments"`
	CreatedAccounts    int `json:"createdAccounts"`
	UpdatedAccounts    int `json:"updatedAccounts"`
}

var (
	orgDepartmentSheet = "Departments"
	orgAccountSheet    = "Accounts"
	orgDepartmentHead  = []string{"departmentId", "name", "parentDepartmentId", "adminConsumerName"}
	orgAccountHead     = []string{"consumerName", "displayName", "email", "status", "userLevel", "departmentId", "parentConsumerName"}
)

func (s *Service) DownloadOrgTemplate(ctx context.Context) ([]byte, error) {
	workbook := excelize.NewFile()
	workbook.SetSheetName("Sheet1", orgDepartmentSheet)
	if err := writeWorkbookSheet(workbook, orgDepartmentSheet, orgDepartmentHead, [][]string{}); err != nil {
		return nil, err
	}
	if _, err := workbook.NewSheet(orgAccountSheet); err != nil {
		return nil, err
	}
	if err := writeWorkbookSheet(workbook, orgAccountSheet, orgAccountHead, [][]string{}); err != nil {
		return nil, err
	}
	return workbookBytes(workbook)
}

func (s *Service) ExportOrganizationWorkbook(ctx context.Context) ([]byte, error) {
	departments, err := s.ListDepartmentTree(ctx)
	if err != nil {
		return nil, err
	}
	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}

	workbook := excelize.NewFile()
	workbook.SetSheetName("Sheet1", orgDepartmentSheet)
	if err := writeWorkbookSheet(workbook, orgDepartmentSheet, orgDepartmentHead, flattenDepartmentWorkbookRows(departments)); err != nil {
		return nil, err
	}
	if _, err := workbook.NewSheet(orgAccountSheet); err != nil {
		return nil, err
	}
	if err := writeWorkbookSheet(workbook, orgAccountSheet, orgAccountHead, flattenAccountWorkbookRows(accounts)); err != nil {
		return nil, err
	}
	return workbookBytes(workbook)
}

func (s *Service) ImportOrganizationWorkbook(ctx context.Context, content []byte) (*OrgImportResult, error) {
	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	defer workbook.Close()

	result := &OrgImportResult{}

	departmentRows, err := workbook.GetRows(orgDepartmentSheet)
	if err != nil {
		return nil, err
	}
	existingDepartments, err := s.ListDepartmentTree(ctx)
	if err != nil {
		return nil, err
	}
	departmentsByID, departmentsByName := indexDepartments(existingDepartments)
	for _, row := range skipHeaderRows(departmentRows) {
		record := normalizeWorkbookRow(row, len(orgDepartmentHead))
		name := strings.TrimSpace(record[1])
		if name == "" {
			continue
		}
		departmentID := strings.TrimSpace(record[0])
		parentDepartmentID := strings.TrimSpace(record[2])
		adminConsumerName := strings.TrimSpace(record[3])

		current := findDepartmentForImport(departmentID, name, departmentsByID, departmentsByName)
		if current == nil {
			created, err := s.CreateDepartment(ctx, DepartmentMutation{
				Name:               name,
				ParentDepartmentID: parentDepartmentID,
				AdminConsumerName:  adminConsumerName,
			})
			if err != nil {
				return nil, err
			}
			departmentsByID[created.DepartmentID] = created
			departmentsByName[created.Name] = created
			result.CreatedDepartments++
			continue
		}
		if _, err := s.UpdateDepartment(ctx, current.DepartmentID, DepartmentMutation{
			Name:              name,
			AdminConsumerName: adminConsumerName,
		}); err != nil {
			return nil, err
		}
		if strings.TrimSpace(current.ParentDepartmentID) != parentDepartmentID {
			if _, err := s.MoveDepartment(ctx, current.DepartmentID, parentDepartmentID); err != nil {
				return nil, err
			}
		}
		result.UpdatedDepartments++
	}

	accountRows, err := workbook.GetRows(orgAccountSheet)
	if err != nil {
		return nil, err
	}
	existingAccounts, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	accountsByName := map[string]OrgAccountRecord{}
	for _, item := range existingAccounts {
		accountsByName[item.ConsumerName] = item
	}
	for _, row := range skipHeaderRows(accountRows) {
		record := normalizeWorkbookRow(row, len(orgAccountHead))
		consumerName := strings.TrimSpace(record[0])
		if consumerName == "" {
			continue
		}
		mutation := AccountMutation{
			ConsumerName:       consumerName,
			DisplayName:        strings.TrimSpace(record[1]),
			Email:              strings.TrimSpace(record[2]),
			Status:             strings.TrimSpace(record[3]),
			UserLevel:          strings.TrimSpace(record[4]),
			DepartmentID:       strings.TrimSpace(record[5]),
			ParentConsumerName: strings.TrimSpace(record[6]),
		}
		if _, ok := accountsByName[consumerName]; ok {
			if _, err := s.UpdateAccount(ctx, consumerName, mutation); err != nil {
				return nil, err
			}
			result.UpdatedAccounts++
			continue
		}
		if _, err := s.CreateAccount(ctx, mutation); err != nil {
			return nil, err
		}
		result.CreatedAccounts++
	}

	return result, nil
}

func writeWorkbookSheet(workbook *excelize.File, sheet string, headers []string, rows [][]string) error {
	for index, header := range headers {
		cell, err := excelize.CoordinatesToCellName(index+1, 1)
		if err != nil {
			return err
		}
		if err := workbook.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
	}
	for rowIndex, row := range rows {
		for colIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			if err != nil {
				return err
			}
			if err := workbook.SetCellValue(sheet, cell, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func workbookBytes(workbook *excelize.File) ([]byte, error) {
	buffer, err := workbook.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func flattenDepartmentWorkbookRows(nodes []*OrgDepartmentNode) [][]string {
	rows := make([][]string, 0)
	var walk func(items []*OrgDepartmentNode)
	walk = func(items []*OrgDepartmentNode) {
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})
		for _, item := range items {
			rows = append(rows, []string{
				item.DepartmentID,
				item.Name,
				item.ParentDepartmentID,
				item.AdminConsumerName,
			})
			walk(item.Children)
		}
	}
	walk(nodes)
	return rows
}

func flattenAccountWorkbookRows(accounts []OrgAccountRecord) [][]string {
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].ConsumerName < accounts[j].ConsumerName
	})
	rows := make([][]string, 0, len(accounts))
	for _, item := range accounts {
		rows = append(rows, []string{
			item.ConsumerName,
			item.DisplayName,
			item.Email,
			item.Status,
			item.UserLevel,
			item.DepartmentID,
			item.ParentConsumerName,
		})
	}
	return rows
}

func indexDepartments(nodes []*OrgDepartmentNode) (map[string]*OrgDepartmentNode, map[string]*OrgDepartmentNode) {
	byID := map[string]*OrgDepartmentNode{}
	byName := map[string]*OrgDepartmentNode{}
	var walk func(items []*OrgDepartmentNode)
	walk = func(items []*OrgDepartmentNode) {
		for _, item := range items {
			clone := *item
			clone.Children = nil
			byID[item.DepartmentID] = &clone
			byName[item.Name] = &clone
			walk(item.Children)
		}
	}
	walk(nodes)
	return byID, byName
}

func findDepartmentForImport(
	departmentID string,
	name string,
	byID map[string]*OrgDepartmentNode,
	byName map[string]*OrgDepartmentNode,
) *OrgDepartmentNode {
	if departmentID != "" {
		if item := byID[departmentID]; item != nil {
			return item
		}
	}
	return byName[name]
}

func skipHeaderRows(rows [][]string) [][]string {
	if len(rows) <= 1 {
		return [][]string{}
	}
	return rows[1:]
}

func normalizeWorkbookRow(row []string, width int) []string {
	result := make([]string, width)
	copy(result, row)
	for index := range result {
		result[index] = strings.TrimSpace(result[index])
	}
	return result
}
