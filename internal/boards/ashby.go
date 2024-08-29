package boards

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"encore.app/pkg/types"
	"encore.dev/beta/errs"
)

type ashbyResponse struct {
	Data struct {
		JobBoard struct {
			JobPostings []struct {
				ID                 string `json:"id"`
				Title              string `json:"title"`
				LocationName       string `json:"locationName"`
				EmploymentType     string `json:"employmentType"`
				SecondaryLocations []struct {
					LocationName string `json:"locationName"`
				} `json:"secondaryLocations"`
			} `json:"jobPostings"`
		} `json:"jobBoard"`
	} `json:"data"`
}

func Ashby(company string) ([]types.Job, error) {
	url := "https://jobs.ashbyhq.com/api/non-user-graphql?op=ApiBoardWithTeams"
	body := map[string]interface{}{
		"operationName": "ApiBoardWithTeams",
		"variables": map[string]string{
			"organizationHostedJobsPageName": company,
		},
		"query": `
			query ApiBoardWithTeams($organizationHostedJobsPageName: String!) {
				jobBoard: jobBoardWithTeams(
					organizationHostedJobsPageName: $organizationHostedJobsPageName
				) {
					jobPostings {
						id
						title
						locationName
						employmentType
						secondaryLocations {
							locationName
						}
					}
				}
			}
		`,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Unavailable,
			Message: "Failed to fetch jobs from Ashby",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}
	defer resp.Body.Close()

	var ashbyResp ashbyResponse
	if err := json.NewDecoder(resp.Body).Decode(&ashbyResp); err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Failed to parse Ashby response",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}

	if ashbyResp.Data.JobBoard.JobPostings == nil {
		return nil, &errs.Error{
			Code:    errs.NotFound,
			Message: fmt.Sprintf("Company %s not found", company),
		}
	}

	var jobs []types.Job
	for _, posting := range ashbyResp.Data.JobBoard.JobPostings {
		locations := []string{posting.LocationName}
		for _, secondary := range posting.SecondaryLocations {
			locations = append(locations, secondary.LocationName)
		}

		jobs = append(jobs, types.Job{
			Title:    posting.Title,
			Location: strings.Join(locations, ", "),
			Link:     fmt.Sprintf("https://jobs.ashbyhq.com/%s/%s", company, posting.ID),
		})
	}

	return jobs, nil
}
