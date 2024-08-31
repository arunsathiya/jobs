package api

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

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
	Platform string
	Company  string
}

var secrets struct {
	NotionToken string
}

func init() {
	notionClient = notionapi.NewClient(notionapi.Token(secrets.NotionToken))
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
		err := processJobSource(ctx, databaseID, job.Platform, job.Company)
		if err != nil {
			fmt.Printf("Error processing %s jobs for %s: %v\n", job.Platform, job.Company, err)
			// Continue with the next job source instead of returning the error
			continue
		}
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
			return nil, fmt.Errorf("invalid config format: expected 'platform,company' in each line")
		}
		config.Jobs = append(config.Jobs, JobSource{
			Platform: strings.TrimSpace(parts[0]),
			Company:  strings.TrimSpace(parts[1]),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

func processJobSource(ctx context.Context, databaseID notionapi.DatabaseID, platform, company string) error {
	boardFunc, ok := boards.GetBoards()[platform]
	if !ok {
		return fmt.Errorf("%s board not found", platform)
	}

	jobs, err := boardFunc(company)
	if err != nil {
		return fmt.Errorf("failed to fetch jobs: %v", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(jobs))

	for _, job := range jobs {
		wg.Add(1)
		go func(j types.Job) {
			defer wg.Done()
			err := createNotionPage(ctx, databaseID, j, platform, company)
			if err != nil {
				errChan <- err
			}
		}(job)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err // Return the first error encountered
		}
	}

	return nil
}

func createNotionPage(ctx context.Context, databaseID notionapi.DatabaseID, job types.Job, platform, company string) error {
	// Extract job ID from the link (last part after '/')
	parts := strings.Split(job.Link, "/")
	jobID := parts[len(parts)-1]

	// If the jobID contains a query parameter, extract only the ID part
	if strings.Contains(jobID, "?") {
		jobID = strings.Split(jobID, "?")[0]
	}

	// Check if the job already exists
	exists, err := jobExists(ctx, databaseID, jobID)
	if err != nil {
		return fmt.Errorf("failed to check if job exists: %v", err)
	}

	if exists {
		// Job already exists, skip creation
		return nil
	}

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
				{Text: &notionapi.Text{Content: jobID}},
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

	_, err = notionClient.Page.Create(ctx, &notionapi.PageCreateRequest{
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

func jobExists(ctx context.Context, databaseID notionapi.DatabaseID, jobID string) (bool, error) {
	query := &notionapi.DatabaseQueryRequest{
		Filter: &notionapi.PropertyFilter{
			Property: "Job ID",
			RichText: &notionapi.TextFilterCondition{
				Equals: jobID,
			},
		},
		PageSize: 1,
	}

	result, err := notionClient.Database.Query(ctx, databaseID, query)
	if err != nil {
		return false, err
	}

	return len(result.Results) > 0, nil
}
