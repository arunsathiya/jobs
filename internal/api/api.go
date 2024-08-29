package api

import (
	"context"

	"encore.app/internal/boards"
	"encore.app/pkg/types"
	"encore.dev/beta/errs"
)

//encore:service
type Service struct {
	Boards map[string]types.BoardFunc
}

func initService() (*Service, error) {
	return &Service{
		Boards: boards.GetBoards(),
	}, nil
}

//encore:api public
func (s *Service) GetJobs(ctx context.Context, params *GetJobsParams) (*GetJobsResponse, error) {
	getJobs, ok := s.Boards[params.Board]
	if !ok {
		return nil, &errs.Error{
			Code:    errs.InvalidArgument,
			Message: "Job board not supported",
		}
	}

	jobs, err := getJobs(params.Company)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Failed to fetch jobs",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}

	return &GetJobsResponse{Jobs: jobs}, nil
}

type GetJobsParams struct {
	Board   string `json:"board"`
	Company string `json:"company"`
}

type GetJobsResponse struct {
	Jobs []types.Job `json:"jobs"`
}
