package gorlim

import (
	"fmt"
	"github.com/google/go-github/github"
	"net/http"
	"net/url"
)

type AuthenticatedTransport struct {
	AccessToken string
	Date        string
	Transport   http.RoundTripper
}

func (t *AuthenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// copy req
	r2 := new(http.Request)
	*r2 = *req
	r2.Header = make(http.Header)
	for k, s := range req.Header {
		r2.Header[k] = s
	}
	req = r2
	q := req.URL.Query()
	q.Set("access_token", t.AccessToken)
	req.URL.RawQuery = q.Encode()
	if t.Date != "" {
		req.Header.Add("If-Modified-Since", t.Date)
	}
	return t.transport().RoundTrip(req)
}

func (t *AuthenticatedTransport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func (t *AuthenticatedTransport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

func getGithubIssues(owner, repo, accessToken, date string) ([]github.Issue, int, error) {
	if date == "" {
		date = "Sat, 24 Jan 2015 00:00:00 GMT"
	}
	t := &AuthenticatedTransport{
		AccessToken: accessToken,
		Date:        date,
	}
	client := github.NewClient(t.Client())
	client.BaseURL, _ = url.Parse(fmt.Sprintf("https://api.github.com/repos/%v/%v", owner, repo))
	ilo := &github.IssueListOptions{}
	issuesService := client.Issues
	issues, resp, err := issuesService.List(true, ilo)

	return issues, resp.StatusCode, err
}

func getGithubIssueComments(owner, repo, accessToken, date string, gIssue github.Issue) ([]github.IssueComment, int, error) {
	if date == "" {
		date = "Sat, 24 Jan 2015 00:00:00 GMT"
	}
	t := &AuthenticatedTransport{
		AccessToken: accessToken,
		Date:        date,
	}
	client := github.NewClient(t.Client())
	client.BaseURL, _ = url.Parse("https://api.github.com/repos/" + repo)
	clo := &github.IssueListCommentsOptions{}
	issuesService := client.Issues
	comments, resp, err := issuesService.ListComments(owner, repo, *gIssue.Comments, clo)

	return comments, resp.StatusCode, err
}

func convertGithubIssue(gIssue github.Issue, gComments []github.IssueComment) gorlim.Issue {
	labelAmount := len(gIssue.Labels)
	labels := make([]string, labelAmount)
	for i := 0; i < labelAmount; i++ {
		labels[i] = gIssue.Labels[i].Name
	}
	commentAmount := len(gComments)
	comments := make([]string, commentAmount)
	for i := 0; i < commentAmount; i++ {
		comments[i] = gComments[i].Body
	}
	result := gorlim.Issue{
		Id:          gIssue.Number,
		Opened:      gIssue.State == "opened",
		Assignee:    gIssue.Assignee,
		Milestone:   gIssue.Milestone.Title,
		Title:       gIssue.Title,
		Description: gComments[0].Body,
		Labels:      labels,
		Comments:    comments,
	}
	return result
}

func GetIssues(owner, repo, accessToken, date string) []gorlim.Issue {
	gIssues, _, err := getGithubIssues(owner, repo, accessToken, date)
	if err != nil {
		panic(err)
	}
	iss := make([]gorlim.Issue, len(gIssues))
	for i := 0; i < len(gIssues); i++ {
		comments, _, err := getGithubIssueComments(owner, repo, accessToken, date, gIssues[i])
		if err != nil {
			panic(err)
		}
		iss[i] = convertGithubIssue(gIssues[i], comments)
	}
	return iss
}
