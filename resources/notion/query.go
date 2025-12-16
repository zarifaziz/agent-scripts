package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/samber/oops"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type QueryOptions struct {
	DatabaseID    string
	Filter        json.RawMessage
	Sorts         json.RawMessage
	PageSize      int
	Limit         int
	ResolvePeople bool
	Format        string
	DryRun        bool
}

type QueryResult struct {
	Pages      []PageResult `json:"pages"`
	TotalCount int          `json:"total_count"`
	HasMore    bool         `json:"has_more,omitempty"`
}

type PageResult struct {
	ID        string   `json:"id"`
	IssueID   string   `json:"issue_id,omitempty"`
	Title     string   `json:"title,omitempty"`
	Status    string   `json:"status,omitempty"`
	Priority  string   `json:"priority,omitempty"`
	Assign    []string `json:"assign,omitempty"`
	QA        []string `json:"qa,omitempty"`
	CreatedBy string   `json:"created_by,omitempty"`
	URL       string   `json:"url"`
}

func (c *Client) Query(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	if opts.DatabaseID == "" {
		opts.DatabaseID = TechIssuesDBID
	}
	if opts.PageSize == 0 {
		opts.PageSize = 100
	}
	if opts.Limit == 0 {
		opts.Limit = 1000
	}

	req := &notionapi.DatabaseQueryRequest{
		PageSize: opts.PageSize,
	}

	if len(opts.Filter) > 0 {
		// Load schema for transformation (use cache)
		schema, schemaErr := c.loadCachedSchema()
		if schemaErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: no cached schema (run 'notion schema --refresh'). Property/enum normalization disabled.\n")
		}

		// Transform LLM-friendly JSON to Notion format
		transformed, err := TransformFilter(opts.Filter, schema)
		if err != nil {
			return nil, oops.Wrapf(err, "filter transform failed")
		}

		filter, err := c.parseFilter(transformed, opts.ResolvePeople)
		if err != nil {
			return nil, oops.Wrapf(err, "invalid filter")
		}
		req.Filter = filter
	}

	if len(opts.Sorts) > 0 {
		var sorts []notionapi.SortObject
		if err := json.Unmarshal(opts.Sorts, &sorts); err != nil {
			return nil, oops.Wrapf(err, "invalid sorts JSON")
		}
		req.Sorts = sorts
	}

	if opts.DryRun {
		reqJSON, _ := json.MarshalIndent(map[string]any{
			"database_id": opts.DatabaseID,
			"filter":      json.RawMessage(opts.Filter),
			"sorts":       json.RawMessage(opts.Sorts),
			"page_size":   opts.PageSize,
			"limit":       opts.Limit,
		}, "", "  ")
		fmt.Printf("Dry run - would send:\n%s\n", reqJSON)
		return &QueryResult{}, nil
	}

	var allPages []notionapi.Page
	var cursor notionapi.Cursor

	for {
		req.StartCursor = cursor
		resp, err := c.client.Database.Query(ctx, notionapi.DatabaseID(opts.DatabaseID), req)
		if err != nil {
			return nil, oops.Wrapf(err, "query failed")
		}

		allPages = append(allPages, resp.Results...)

		if !resp.HasMore || len(allPages) >= opts.Limit {
			break
		}
		cursor = notionapi.Cursor(resp.NextCursor)
	}

	if len(allPages) > opts.Limit {
		allPages = allPages[:opts.Limit]
	}

	result := &QueryResult{
		Pages:      make([]PageResult, 0, len(allPages)),
		TotalCount: len(allPages),
	}

	for _, page := range allPages {
		result.Pages = append(result.Pages, c.pageToResult(&page))
	}

	return result, nil
}

func (c *Client) pageToResult(page *notionapi.Page) PageResult {
	return PageResult{
		ID:        string(page.ID),
		IssueID:   ExtractIssueID(page),
		Title:     ExtractTitle(page),
		Status:    ExtractStatus(page),
		Priority:  ExtractPriority(page),
		Assign:    ExtractPeople(page, "Assign"),
		QA:        ExtractPeople(page, "QA Engineer"),
		CreatedBy: ExtractCreatedBy(page),
		URL:       fmt.Sprintf("https://notion.so/%s", strings.ReplaceAll(string(page.ID), "-", "")),
	}
}

func (c *Client) parseFilter(raw json.RawMessage, resolvePeople bool) (notionapi.Filter, error) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	if resolvePeople {
		c.resolveUserNames(data)
	}

	if andFilters, ok := data["and"].([]any); ok {
		return c.buildCompoundFilter("and", andFilters, resolvePeople)
	}
	if orFilters, ok := data["or"].([]any); ok {
		return c.buildCompoundFilter("or", orFilters, resolvePeople)
	}

	return c.buildPropertyFilter(data)
}

func (c *Client) buildCompoundFilter(op string, filters []any, resolvePeople bool) (notionapi.Filter, error) {
	var result []notionapi.Filter

	for _, f := range filters {
		filterMap, ok := f.(map[string]any)
		if !ok {
			continue
		}

		if resolvePeople {
			c.resolveUserNames(filterMap)
		}

		if nested, ok := filterMap["and"].([]any); ok {
			compound, err := c.buildCompoundFilter("and", nested, resolvePeople)
			if err != nil {
				return nil, err
			}
			result = append(result, compound)
			continue
		}
		if nested, ok := filterMap["or"].([]any); ok {
			compound, err := c.buildCompoundFilter("or", nested, resolvePeople)
			if err != nil {
				return nil, err
			}
			result = append(result, compound)
			continue
		}

		pf, err := c.buildPropertyFilter(filterMap)
		if err != nil {
			return nil, err
		}
		result = append(result, pf)
	}

	if op == "or" {
		return notionapi.OrCompoundFilter(result), nil
	}
	return notionapi.AndCompoundFilter(result), nil
}

func (c *Client) buildPropertyFilter(data map[string]any) (notionapi.PropertyFilter, error) {
	pf := notionapi.PropertyFilter{}

	prop, ok := data["property"].(string)
	if !ok {
		return pf, oops.New("filter missing 'property' field")
	}
	pf.Property = prop

	if checkbox, ok := data["checkbox"].(map[string]any); ok {
		pf.Checkbox = &notionapi.CheckboxFilterCondition{}
		if v, ok := checkbox["equals"].(bool); ok {
			pf.Checkbox.Equals = v
		}
		if v, ok := checkbox["does_not_equal"].(bool); ok {
			pf.Checkbox.DoesNotEqual = v
		}
	}

	if status, ok := data["status"].(map[string]any); ok {
		pf.Status = &notionapi.StatusFilterCondition{}
		if v, ok := status["equals"].(string); ok {
			pf.Status.Equals = v
		}
		if v, ok := status["does_not_equal"].(string); ok {
			pf.Status.DoesNotEqual = v
		}
		if _, ok := status["is_empty"].(bool); ok {
			pf.Status.IsEmpty = true
		}
		if _, ok := status["is_not_empty"].(bool); ok {
			pf.Status.IsNotEmpty = true
		}
	}

	if sel, ok := data["select"].(map[string]any); ok {
		pf.Select = &notionapi.SelectFilterCondition{}
		if v, ok := sel["equals"].(string); ok {
			pf.Select.Equals = v
		}
		if v, ok := sel["does_not_equal"].(string); ok {
			pf.Select.DoesNotEqual = v
		}
		if _, ok := sel["is_empty"].(bool); ok {
			pf.Select.IsEmpty = true
		}
		if _, ok := sel["is_not_empty"].(bool); ok {
			pf.Select.IsNotEmpty = true
		}
	}

	if multiSel, ok := data["multi_select"].(map[string]any); ok {
		pf.MultiSelect = &notionapi.MultiSelectFilterCondition{}
		if v, ok := multiSel["contains"].(string); ok {
			pf.MultiSelect.Contains = v
		}
		if v, ok := multiSel["does_not_contain"].(string); ok {
			pf.MultiSelect.DoesNotContain = v
		}
		if _, ok := multiSel["is_empty"].(bool); ok {
			pf.MultiSelect.IsEmpty = true
		}
		if _, ok := multiSel["is_not_empty"].(bool); ok {
			pf.MultiSelect.IsNotEmpty = true
		}
	}

	if richText, ok := data["rich_text"].(map[string]any); ok {
		pf.RichText = &notionapi.TextFilterCondition{}
		if v, ok := richText["contains"].(string); ok {
			pf.RichText.Contains = v
		}
		if v, ok := richText["does_not_contain"].(string); ok {
			pf.RichText.DoesNotContain = v
		}
		if v, ok := richText["equals"].(string); ok {
			pf.RichText.Equals = v
		}
		if v, ok := richText["does_not_equal"].(string); ok {
			pf.RichText.DoesNotEqual = v
		}
		if v, ok := richText["starts_with"].(string); ok {
			pf.RichText.StartsWith = v
		}
		if v, ok := richText["ends_with"].(string); ok {
			pf.RichText.EndsWith = v
		}
		if _, ok := richText["is_empty"].(bool); ok {
			pf.RichText.IsEmpty = true
		}
		if _, ok := richText["is_not_empty"].(bool); ok {
			pf.RichText.IsNotEmpty = true
		}
	}

	if number, ok := data["number"].(map[string]any); ok {
		pf.Number = &notionapi.NumberFilterCondition{}
		if v, ok := number["equals"].(float64); ok {
			pf.Number.Equals = &v
		}
		if v, ok := number["does_not_equal"].(float64); ok {
			pf.Number.DoesNotEqual = &v
		}
		if v, ok := number["greater_than"].(float64); ok {
			pf.Number.GreaterThan = &v
		}
		if v, ok := number["less_than"].(float64); ok {
			pf.Number.LessThan = &v
		}
		if v, ok := number["greater_than_or_equal_to"].(float64); ok {
			pf.Number.GreaterThanOrEqualTo = &v
		}
		if v, ok := number["less_than_or_equal_to"].(float64); ok {
			pf.Number.LessThanOrEqualTo = &v
		}
		if _, ok := number["is_empty"].(bool); ok {
			pf.Number.IsEmpty = true
		}
		if _, ok := number["is_not_empty"].(bool); ok {
			pf.Number.IsNotEmpty = true
		}
	}

	if people, ok := data["people"].(map[string]any); ok {
		pf.People = &notionapi.PeopleFilterCondition{}
		if v, ok := people["contains"].(string); ok {
			pf.People.Contains = v
		}
		if v, ok := people["does_not_contain"].(string); ok {
			pf.People.DoesNotContain = v
		}
		if _, ok := people["is_empty"].(bool); ok {
			pf.People.IsEmpty = true
		}
		if _, ok := people["is_not_empty"].(bool); ok {
			pf.People.IsNotEmpty = true
		}
	}

	if uniqueID, ok := data["unique_id"].(map[string]any); ok {
		pf.UniqueId = &notionapi.UniqueIdFilterCondition{}
		if v, ok := uniqueID["equals"].(float64); ok {
			num := int(v)
			pf.UniqueId.Equals = &num
		}
		if v, ok := uniqueID["does_not_equal"].(float64); ok {
			num := int(v)
			pf.UniqueId.DoesNotEqual = &num
		}
	}

	return pf, nil
}

func (c *Client) resolveUserNames(data map[string]any) {
	if people, ok := data["people"].(map[string]any); ok {
		for key, val := range people {
			if strVal, ok := val.(string); ok && !isUUID(strVal) {
				if resolved, exists := userIDMap[strings.ToLower(strVal)]; exists {
					people[key] = resolved
				}
			}
		}
	}

	if createdBy, ok := data["created_by"].(map[string]any); ok {
		for key, val := range createdBy {
			if strVal, ok := val.(string); ok && !isUUID(strVal) {
				if resolved, exists := userIDMap[strings.ToLower(strVal)]; exists {
					createdBy[key] = resolved
				}
			}
		}
	}

	if lastEditedBy, ok := data["last_edited_by"].(map[string]any); ok {
		for key, val := range lastEditedBy {
			if strVal, ok := val.(string); ok && !isUUID(strVal) {
				if resolved, exists := userIDMap[strings.ToLower(strVal)]; exists {
					lastEditedBy[key] = resolved
				}
			}
		}
	}
}

func isUUID(s string) bool {
	return uuidRegex.MatchString(strings.ToLower(s))
}
