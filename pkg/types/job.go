package types

type Job struct {
	Title    string `json:"title"`
	Location string `json:"location"`
	Link     string `json:"link"`
}

type BoardFunc func(company string) ([]Job, error)
