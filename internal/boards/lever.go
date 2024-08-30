package boards

import (
	"encoding/json"
	"encore/pkg/types"
	"fmt"
	"net/http"

	"encore.dev/beta/errs"
)

type leverResponse []struct {
	Text       string `json:"text"`
	HostedURL  string `json:"hostedUrl"`
	Categories struct {
		Location string `json:"location"`
	} `json:"categories"`
}

func Lever(company string) ([]types.Job, error) {
	url := fmt.Sprintf("https://api.lever.co/v0/postings/%s", company)
	resp, err := http.Get(url)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Unavailable,
			Message: "Failed to fetch jobs from Lever",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}
	defer resp.Body.Close()

	var leverResp leverResponse
	if err := json.NewDecoder(resp.Body).Decode(&leverResp); err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Failed to parse Lever response",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}

	if len(leverResp) == 0 {
		return nil, &errs.Error{
			Code:    errs.NotFound,
			Message: fmt.Sprintf("Company %s not found", company),
		}
	}

	var jobs []types.Job
	for _, posting := range leverResp {
		jobs = append(jobs, types.Job{
			Title:    posting.Text,
			Location: posting.Categories.Location,
			Link:     posting.HostedURL,
		})
	}

	return jobs, nil
}
