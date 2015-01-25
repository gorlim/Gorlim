package gorlim_github

import (
	"github.com/google/go-github/github"
	"gorlim"
	"net/http"
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

func getGithubIssues(owner string, repo string, client *github.Client, date string) ([]github.Issue, error) {
	if date == "" {
		date = "Sat, 24 Jan 2015 00:00:00 GMT"
	}
	issuesService := client.Issues
	result := make([]github.Issue, 0, 100)
	opts := make([]github.IssueListByRepoOptions, 0, 100)
	opts = append(opts, github.IssueListByRepoOptions{Milestone: "none", Assignee: "none", State: "open"})
	opts = append(opts, github.IssueListByRepoOptions{Milestone: "*", Assignee: "none", State: "open"})
	tmp := make([]github.IssueListByRepoOptions, 0, len(opts))
	for _, opt := range opts {
		newOpt := opt
		newOpt.State = "closed"
		tmp = append(tmp, newOpt)
	}
	opts = append(opts, tmp...)
	tmp = make([]github.IssueListByRepoOptions, 0, len(opts))
	for _, opt := range opts {
		newOpt := opt
		newOpt.Assignee = "*"
		tmp = append(tmp, newOpt)
	}
	opts = append(opts, tmp...)
	for _, opt := range opts {
		issues, _, err := issuesService.ListByRepo(owner, repo, &opt)
		if err == nil {
			result = append(result, issues...)
		}
	}
	return result, nil
}

func getGithubIssueComments(owner string, repo string, client *github.Client, date string, gIssue github.Issue) ([]github.IssueComment, int) {
	if date == "" {
		date = "Sat, 24 Jan 2015 00:00:00 GMT"
	}
	clo := &github.IssueListCommentsOptions{}
	issuesService := client.Issues
	comments, resp, _ := issuesService.ListComments(owner, repo, *gIssue.Comments, clo)

	return comments, resp.StatusCode
}

func convertGithubIssue(gIssue github.Issue, gComments []github.IssueComment) gorlim.Issue {
	labelAmount := len(gIssue.Labels)
	labels := make([]string, labelAmount)
	for i := 0; i < labelAmount; i++ {
		labels[i] = *gIssue.Labels[i].Name
	}
	commentAmount := len(gComments)
	description := ""
	comments := make([]string, commentAmount)
	if commentAmount > 0 {
		description = *(gComments[0].Body)
		for i := 1; i < commentAmount; i++ {
			comments[i] = *gComments[i].Body
		}
	}
	id := *gIssue.Number
	opened := (*gIssue.State) == "opened"
	assignee := ""
	if user := gIssue.Assignee; user != nil {
		assignee = *user.Login
	}
	milestone := ""
	if mi := gIssue.Milestone; mi != nil {
		milestone = *mi.Title
	}
	creator := ""
	if author := gIssue.User; author != nil {
		creator = *author.Login
	}
	title := *gIssue.Title

	result := gorlim.Issue{
		Id:          id,
		Opened:      opened,
		Creator:     creator,
		Assignee:    assignee,
		Milestone:   milestone,
		Title:       title,
		Description: description,
		Labels:      labels,
		Comments:    comments,
	}
	return result
}

func GetIssues(owner string, repo string, client *http.Client, date string) []gorlim.Issue {
	gh := github.NewClient(client)
	gIssues, err := getGithubIssues(owner, repo, gh, date)
	if err != nil {
		panic(err)
	}
	iss := make([]gorlim.Issue, len(gIssues))
	for i, issue := range gIssues {
		comments, _ := getGithubIssueComments(owner, repo, gh, date, issue)
		iss[i] = convertGithubIssue(issue, comments)
	}
	return iss
}

func SetIssues(owner string, repo string, client *http.Client, date string, issues []gorlim.Issue) {

}
