package api

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"encore/internal/boards"
	"encore/pkg/types"

	"encore.dev/cron"
	"github.com/jomei/notionapi"
)

var notionClient *notionapi.Client

type Config struct {
	NotionDatabaseID string
	Jobs             []JobSource
}

type JobSource struct {
	Company  string
	Platform string
}

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
	Every:    1 * cron.Hour,
	Endpoint: UpdateNotionJobs,
})

//encore:api private
func UpdateNotionJobs(ctx context.Context) error {
	// Read configuration
	config, err := readConfig("config.txt")
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	databaseID := notionapi.DatabaseID(config.NotionDatabaseID)
	if databaseID == "" {
		return fmt.Errorf("Notion database ID is not set in config")
	}

	for _, job := range config.Jobs {
		err := processJobSource(ctx, databaseID, job.Company, job.Platform)
		if err != nil {
			fmt.Printf("Error processing %s jobs for %s: %v\n", job.Platform, job.Company, err)
			// Continue with the next job source instead of returning the error
			continue
		}
	}

	return nil
}

func processJobSource(ctx context.Context, databaseID notionapi.DatabaseID, company, platform string) error {
	boardFunc, ok := boards.GetBoards()[platform]
	if !ok {
		return fmt.Errorf("%s board not found", platform)
	}

	jobs, err := boardFunc(company)
	if err != nil {
		return fmt.Errorf("failed to fetch jobs: %v", err)
	}

	for _, job := range jobs {
		err := createNotionPage(ctx, databaseID, job, company, platform)
		if err != nil {
			return err
		}
	}

	return nil
}

func createNotionPage(ctx context.Context, databaseID notionapi.DatabaseID, job types.Job, company, platform string) error {
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
				{Text: &notionapi.Text{Content: company}},
			},
		},
		"Job platform": notionapi.RichTextProperty{
			RichText: []notionapi.RichText{
				{Text: &notionapi.Text{Content: platform}},
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

	return nil
}

func readConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	scanner := bufio.NewScanner(file)

	// Read Notion Database ID
	if scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")
		if len(parts) == 2 && parts[0] == "notion_database_id" {
			config.NotionDatabaseID = parts[1]
		} else {
			return nil, fmt.Errorf("invalid config format: expected 'notion_database_id' in first line")
		}
	}

	// Skip header line
	scanner.Scan()

	// Read job sources
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid config format: expected 'company,platform' in each line")
		}
		config.Jobs = append(config.Jobs, JobSource{
			Company:  strings.TrimSpace(parts[0]),
			Platform: strings.TrimSpace(parts[1]),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}
