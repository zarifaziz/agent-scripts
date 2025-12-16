package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/samber/oops"
)

const (
	TechIssuesDBID = "0cabad237e624481bd8c48c8a4a08a7b"
	IssueIDProp    = "ID"
)

var issueRegex = regexp.MustCompile(`^(?:ISSUE|TASK)-(\d+)$`)

type Client struct {
	client *notionapi.Client
}

func NewClient(token string) *Client {
	return &Client{
		client: notionapi.NewClient(notionapi.Token(token)),
	}
}

func (c *Client) GetPageByIssueID(ctx context.Context, issueID string) (*notionapi.Page, error) {
	issueID = strings.ToUpper(strings.TrimSpace(issueID))

	matches := issueRegex.FindStringSubmatch(issueID)
	if matches == nil {
		return nil, oops.With("issueID", issueID).New("invalid issue ID format (expected ISSUE-xxx or TASK-xxx)")
	}

	issueNum, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, oops.With("issueID", issueID).Wrapf(err, "invalid issue number")
	}

	resp, err := c.client.Database.Query(ctx, notionapi.DatabaseID(TechIssuesDBID), &notionapi.DatabaseQueryRequest{
		Filter: notionapi.PropertyFilter{
			Property: IssueIDProp,
			UniqueId: &notionapi.UniqueIdFilterCondition{
				Equals: &issueNum,
			},
		},
	})
	if err != nil {
		return nil, oops.With("issueNum", issueNum).Wrapf(err, "failed to query database")
	}

	if len(resp.Results) == 0 {
		return nil, oops.With("issueID", issueID).New("no issue found")
	}

	return &resp.Results[0], nil
}

func (c *Client) SearchByTitle(ctx context.Context, query string) ([]notionapi.Page, error) {
	resp, err := c.client.Database.Query(ctx, notionapi.DatabaseID(TechIssuesDBID), &notionapi.DatabaseQueryRequest{
		Filter: notionapi.PropertyFilter{
			Property: "Issue Title",
			RichText: &notionapi.TextFilterCondition{
				Contains: query,
			},
		},
	})
	if err != nil {
		return nil, oops.Wrapf(err, "failed to search database")
	}

	return resp.Results, nil
}

func (c *Client) GetPageContent(ctx context.Context, pageID string) ([]notionapi.Block, error) {
	var allBlocks []notionapi.Block
	var cursor notionapi.Cursor

	for {
		resp, err := c.client.Block.GetChildren(ctx, notionapi.BlockID(pageID), &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    100,
		})
		if err != nil {
			return nil, oops.Wrapf(err, "failed to get page content")
		}

		allBlocks = append(allBlocks, resp.Results...)

		if !resp.HasMore {
			break
		}
		cursor = notionapi.Cursor(resp.NextCursor)
	}

	return allBlocks, nil
}

func ExtractTitle(page *notionapi.Page) string {
	prop, ok := page.Properties["Issue Title"]
	if !ok {
		return ""
	}

	titleProp, ok := prop.(*notionapi.TitleProperty)
	if !ok {
		return ""
	}

	if len(titleProp.Title) == 0 {
		return ""
	}

	return titleProp.Title[0].PlainText
}

func ExtractStatus(page *notionapi.Page) string {
	prop, ok := page.Properties["Status"]
	if !ok {
		return ""
	}

	statusProp, ok := prop.(*notionapi.StatusProperty)
	if !ok {
		return ""
	}

	if statusProp.Status.Name == "" {
		return ""
	}

	return statusProp.Status.Name
}

func ExtractPriority(page *notionapi.Page) string {
	prop, ok := page.Properties["Priority"]
	if !ok {
		return ""
	}

	selectProp, ok := prop.(*notionapi.SelectProperty)
	if !ok {
		return ""
	}

	if selectProp.Select.Name == "" {
		return ""
	}

	return selectProp.Select.Name
}

func ExtractPeople(page *notionapi.Page, propName string) []string {
	prop, ok := page.Properties[propName]
	if !ok {
		return nil
	}

	peopleProp, ok := prop.(*notionapi.PeopleProperty)
	if !ok {
		return nil
	}

	var names []string
	for _, p := range peopleProp.People {
		names = append(names, p.Name)
	}
	return names
}

func ExtractCreatedBy(page *notionapi.Page) string {
	prop, ok := page.Properties["Created by"]
	if !ok {
		return ""
	}

	createdByProp, ok := prop.(*notionapi.CreatedByProperty)
	if !ok {
		return ""
	}

	return createdByProp.CreatedBy.Name
}

var userIDMap = map[string]string{
	// Core team (first name)
	"hemanta":  "f3687710-691f-4827-8f92-19e4faa41d62",
	"vatsal":   "0ab71adc-ca8f-498f-9107-b78e740a13a5",
	"shubham":  "a552be6f-35e2-457d-93fa-9e257c13c772",
	"adam":     "ab026605-729a-4117-a4a7-83f9df072088",
	"oliver":   "a4d8fda8-a3e3-456a-9e66-43534d881c53",
	"daniel":   "0bfe4db2-2b43-4f54-8570-1727e8ca30a9",
	"tom":      "0e4024a6-cc66-43d0-bae1-e9c771f0798d",
	"sonny":    "d1e19aab-918e-4a32-9bbd-f8387570396c",
	"richard":  "5d455806-cc2a-461a-ad9f-d23a16c6e01f",
	"khushboo": "ea0e2934-def3-4adb-83d1-6c3d70575ae8",
	"keren":    "0a5b9e11-26da-4797-86f9-41d9072dcbb1",
	"samuel":   "5199d1ca-e144-4776-b2cb-35bfca62554f",
	// GitHub usernames
	"mayor04":         "5199d1ca-e144-4776-b2cb-35bfca62554f",
	"preet3001":       "a02c34a2-7ac9-40b4-bab0-4d65442b216e",
	"raqeebshafeeque": "0bfe4db2-2b43-4f54-8570-1727e8ca30a9",
	"utkarshshendge":  "0e4024a6-cc66-43d0-bae1-e9c771f0798d",
	"drabu":           "e8cc6fd3-f50a-412f-85b8-adccb520bed6",
	"r0shish":         "f826e187-5048-4a79-b87f-6a6da20879e9",
	"roshish":         "f826e187-5048-4a79-b87f-6a6da20879e9",
	"noman2002":       "69129717-302f-4dc5-9b78-853ec8115b98",
	"oliatienza":      "a4d8fda8-a3e3-456a-9e66-43534d881c53",
	"shubham030":      "a552be6f-35e2-457d-93fa-9e257c13c772",
	"lord-chris":      "5f477614-8e12-4258-a051-c3dbc7d6da8d",
	"ajinkyag911":     "cbea354e-d597-4c8d-97c3-38b001545ef1",
	"vatsalpatel":     "0ab71adc-ca8f-498f-9107-b78e740a13a5",
}

func (c *Client) SearchByPerson(ctx context.Context, propName, personName string) ([]notionapi.Page, error) {
	personName = strings.ToLower(personName)

	userID, ok := userIDMap[personName]
	if !ok {
		return nil, oops.With("name", personName).New("unknown user, available: hemanta, mark, adam, sam, daniel, tom, dominic")
	}

	resp, err := c.client.Database.Query(ctx, notionapi.DatabaseID(TechIssuesDBID), &notionapi.DatabaseQueryRequest{
		PageSize: 100,
		Filter: notionapi.PropertyFilter{
			Property: propName,
			People: &notionapi.PeopleFilterCondition{
				Contains: userID,
			},
		},
	})
	if err != nil {
		return nil, oops.Wrapf(err, "failed to search by %s", propName)
	}
	return resp.Results, nil
}

type SchemaProperty struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Options []string `json:"options,omitempty"` // For select/multi_select/status
}

type DatabaseSchema struct {
	ID         string                    `json:"id"`
	Title      string                    `json:"title"`
	Properties map[string]SchemaProperty `json:"properties"`
	Users      map[string]string         `json:"users"` // name -> id mapping for convenience
}

func (c *Client) loadCachedSchema() (*DatabaseSchema, error) {
	cacheFile := getCacheFilePath()
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}
	var schema DatabaseSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

func getCacheFilePath() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return xdg + "/notion/schema.json"
	}
	home, _ := os.UserHomeDir()
	return home + "/.cache/notion/schema.json"
}

func (c *Client) GetSchema(ctx context.Context, dbID string) (*DatabaseSchema, error) {
	db, err := c.client.Database.Get(ctx, notionapi.DatabaseID(dbID))
	if err != nil {
		return nil, oops.Wrapf(err, "failed to get database")
	}

	schema := &DatabaseSchema{
		ID:         string(db.ID),
		Properties: make(map[string]SchemaProperty),
		Users:      userIDMap,
	}

	// Extract title
	for _, rt := range db.Title {
		schema.Title += rt.PlainText
	}

	// Extract properties with their types and options
	for name, prop := range db.Properties {
		sp := SchemaProperty{
			Name: name,
			Type: string(prop.GetType()),
		}

		switch p := prop.(type) {
		case *notionapi.SelectPropertyConfig:
			for _, opt := range p.Select.Options {
				sp.Options = append(sp.Options, opt.Name)
			}
		case *notionapi.MultiSelectPropertyConfig:
			for _, opt := range p.MultiSelect.Options {
				sp.Options = append(sp.Options, opt.Name)
			}
		case *notionapi.StatusPropertyConfig:
			for _, opt := range p.Status.Options {
				sp.Options = append(sp.Options, opt.Name)
			}
		}

		schema.Properties[name] = sp
	}

	return schema, nil
}

func ExtractIssueID(page *notionapi.Page) string {
	prop, ok := page.Properties["ID"]
	if !ok {
		return ""
	}

	uniqueIDProp, ok := prop.(*notionapi.UniqueIDProperty)
	if !ok {
		return ""
	}

	prefix := "ISSUE"
	if uniqueIDProp.UniqueID.Prefix != nil && *uniqueIDProp.UniqueID.Prefix != "" {
		prefix = *uniqueIDProp.UniqueID.Prefix
	}
	return fmt.Sprintf("%s-%d", prefix, uniqueIDProp.UniqueID.Number)
}

func BlockToText(block notionapi.Block) string {
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return richTextToPlain(b.Paragraph.RichText)
	case *notionapi.Heading1Block:
		return "# " + richTextToPlain(b.Heading1.RichText)
	case *notionapi.Heading2Block:
		return "## " + richTextToPlain(b.Heading2.RichText)
	case *notionapi.Heading3Block:
		return "### " + richTextToPlain(b.Heading3.RichText)
	case *notionapi.BulletedListItemBlock:
		return "- " + richTextToPlain(b.BulletedListItem.RichText)
	case *notionapi.NumberedListItemBlock:
		return "1. " + richTextToPlain(b.NumberedListItem.RichText)
	case *notionapi.ToDoBlock:
		check := " "
		if b.ToDo.Checked {
			check = "x"
		}
		return fmt.Sprintf("[%s] %s", check, richTextToPlain(b.ToDo.RichText))
	case *notionapi.CodeBlock:
		lang := ""
		if b.Code.Language != "" {
			lang = string(b.Code.Language)
		}
		return fmt.Sprintf("```%s\n%s\n```", lang, richTextToPlain(b.Code.RichText))
	case *notionapi.QuoteBlock:
		return "> " + richTextToPlain(b.Quote.RichText)
	case *notionapi.DividerBlock:
		return "---"
	default:
		return ""
	}
}

func richTextToPlain(richText []notionapi.RichText) string {
	var parts []string
	for _, rt := range richText {
		parts = append(parts, rt.PlainText)
	}
	return strings.Join(parts, "")
}
