package boards

import (
	"encore/pkg/types"
	"fmt"
	"net/http"
	"strings"

	"encore.dev/beta/errs"
	"golang.org/x/net/html"
)

func Greenhouse(company string) ([]types.Job, error) {
	url := fmt.Sprintf("https://boards.greenhouse.io/embed/job_board?for=%s", company)
	resp, err := http.Get(url)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Unavailable,
			Message: "Failed to fetch jobs from Greenhouse",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Failed to parse Greenhouse HTML",
			Meta:    map[string]interface{}{"error": err.Error()},
		}
	}

	var jobs []types.Job
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "opening" {
					job := parseJobNode(n, company)
					if job != nil {
						jobs = append(jobs, *job)
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if len(jobs) == 0 {
		return nil, &errs.Error{
			Code:    errs.NotFound,
			Message: fmt.Sprintf("Company %s not found", company),
		}
	}

	return jobs, nil
}

func parseJobNode(n *html.Node, company string) *types.Job {
	var title, location, link string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "a" {
			for _, a := range c.Attr {
				if a.Key == "href" {
					link = a.Val
				}
			}
			title = c.FirstChild.Data
		}
		if c.Type == html.ElementNode && c.Data == "span" {
			for _, a := range c.Attr {
				if a.Key == "class" && a.Val == "location" {
					location = c.FirstChild.Data
				}
			}
		}
	}
	if !strings.HasPrefix(link, "http") {
		link = fmt.Sprintf("https://boards.greenhouse.io%s", link)
	}
	return &types.Job{
		Title:    title,
		Location: location,
		Link:     link,
	}
}
