package api

import (
	"context"
	"fmt"
	"os"
	"strings"

	"encore/internal/boards"

	"encore.dev/cron"
	"github.com/jomei/notionapi"
)

var notionClient *notionapi.Client

func init() {
	notionToken := os.Getenv("NOTION_TOKEN")
	if notionToken == "" {
		panic("NOTION_TOKEN environment variable is not set")
	}
	notionClient = notionapi.NewClient(notionapi.Token(notionToken))
}

// Set up the cron job to run every hour
var _ = cron.NewJob("update-notion-jobs", cron.JobConfig{
	Title:    "Update Notion with latest jobs",
	Every:    3 * cron.Minute,
	Endpoint: UpdateNotionJobs,
})

//encore:api private
func UpdateNotionJobs(ctx context.Context) error {
	// Fetch jobs using the boards package
	boardFunc, ok := boards.GetBoards()["ashby"]
	if !ok {
		return fmt.Errorf("ashby board not found")
	}

	jobs, err := boardFunc("pinecone")
	if err != nil {
		return fmt.Errorf("failed to fetch jobs: %v", err)
	}

	// Your Notion database ID
	databaseID := notionapi.DatabaseID("21ad46420f034d4496498a4747e38386")

	// Write each job to Notion
	for _, job := range jobs {
		properties := notionapi.Properties{
			"Name": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{Text: &notionapi.Text{Content: job.Title}},
				},
			},
			"Location": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{Text: &notionapi.Text{Content: job.Location}},
				},
			},
			"Link": notionapi.URLProperty{
				URL: job.Link,
			},
			"Job ID": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{Text: &notionapi.Text{Content: strings.Split(job.Link, "/")[4]}},
				},
			},
			"Company name": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{Text: &notionapi.Text{Content: "Pinecone"}},
				},
			},
			"Job platform": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{Text: &notionapi.Text{Content: "Ashby"}},
				},
			},
		}

		_, err := notionClient.Page.Create(ctx, &notionapi.PageCreateRequest{
			Parent: notionapi.Parent{
				Type:       notionapi.ParentTypeDatabaseID,
				DatabaseID: databaseID,
			},
			Properties: properties,
		})
		if err != nil {
			return fmt.Errorf("failed to create page for job %s: %v", job.Title, err)
		}
	}

	return nil
}
