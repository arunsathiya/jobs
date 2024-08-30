package boards

import (
	"bytes"
	"encoding/json"
	"encore/pkg/types"
	"fmt"
	"net/http"

	"encore.dev/beta/errs"
)

type workableResponse struct {
	NextPage string `json:"nextPage"`
	Results  []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Location struct {
			City        string `json:"city"`
			Country     string `json:"country"`
			CountryCode string `json:"countryCode"`
			Region      string `json:"region"`
		} `json:"location"`
	} `json:"results"`
	Total int `json:"total"`
}

func Workable(company string) ([]types.Job, error) {
	url := fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", company)
	var jobs []types.Job
	var token string

	for {
		body := map[string]string{"token": token}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, &errs.Error{
				Code:    errs.Unavailable,
				Message: "Failed to fetch jobs from Workable",
				Meta:    map[string]interface{}{"error": err.Error()},
			}
		}
		defer resp.Body.Close()

		var workableResp workableResponse
		if err := json.NewDecoder(resp.Body).Decode(&workableResp); err != nil {
			return nil, &errs.Error{
				Code:    errs.Internal,
				Message: "Failed to parse Workable response",
				Meta:    map[string]interface{}{"error": err.Error()},
			}
		}

		for _, result := range workableResp.Results {
			location := result.Location.City
			if result.Location.Region != "" {
				location += ", " + result.Location.Region
			} else {
				location += ", " + result.Location.Country
			}
			jobs = append(jobs, types.Job{
				Title:    result.Title,
				Location: location,
				Link:     fmt.Sprintf("https://%s.workable.com/j/%s", company, result.ID),
			})
		}

		if workableResp.NextPage == "" {
			break
		}
		token = workableResp.NextPage
	}

	if len(jobs) == 0 {
		return nil, &errs.Error{
			Code:    errs.NotFound,
			Message: fmt.Sprintf("Company %s not found", company),
		}
	}

	return jobs, nil
}
