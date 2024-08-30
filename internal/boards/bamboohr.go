package boards

import (
	"encoding/json"
	"encore/pkg/types"
	"fmt"
	"net/http"

	"encore.dev/beta/errs"
)

type bambooHrResponse struct {
	Result []struct {
		ID             string `json:"id"`
		JobOpeningName string `json:"jobOpeningName"`
		Location       struct {
			City  string `json:"city"`
			State string `json:"state"`
		} `json:"location"`
	} `json:"result"`
}

func BambooHR(company string) ([]types.Job, error) {
	url := fmt.Sprintf("https://%s.bamboohr.com/careers/list", company)
	resp, err := http.Get(url)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Unavailable,
			Message: "Failed to fetch jobs from BambooHR",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}
	defer resp.Body.Close()

	var bambooResp bambooHrResponse
	if err := json.NewDecoder(resp.Body).Decode(&bambooResp); err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Failed to parse BambooHR response",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}

	if len(bambooResp.Result) == 0 {
		return nil, &errs.Error{
			Code:    errs.NotFound,
			Message: fmt.Sprintf("Company %s not found", company),
		}
	}

	var jobs []types.Job
	for _, result := range bambooResp.Result {
		jobs = append(jobs, types.Job{
			Title:    result.JobOpeningName,
			Location: fmt.Sprintf("%s, %s", result.Location.City, result.Location.State),
			Link:     fmt.Sprintf("https://%s.bamboohr.com/careers/%s", company, result.ID),
		})
	}

	return jobs, nil
}
