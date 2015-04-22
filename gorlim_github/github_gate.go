package gorlim_github

import (
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/gorlim/Gorlim/gorlim"
	"net/http"
	"strconv"
	"time"
)

var DEFAULT_DATE time.Time = time.Unix(0, 0)

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

func getGithubIssues(owner string, repo string, client *github.Client, date *time.Time) ([]github.Issue, error) {
	if date == nil {
		date = &DEFAULT_DATE
	}
	issuesService := client.Issues
	result := make([]github.Issue, 0, 100)
	opts := make([]github.IssueListByRepoOptions, 0, 100)
	none := github.IssueListByRepoOptions{Milestone: "none", Assignee: "none", State: "open", Since: *date}
	none.ListOptions = github.ListOptions{PerPage: 100}
	opts = append(opts, none)
	any := github.IssueListByRepoOptions{Milestone: "*", Assignee: "none", State: "open", Since: *date}
	any.ListOptions = github.ListOptions{PerPage: 100}
	opts = append(opts, any)
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
		for {
			issues, resp, err := issuesService.ListByRepo(owner, repo, &opt)
			if err == nil {
				result = append(result, issues...)
				resp.Body.Close()
			} else {
				fmt.Println(err)
				break
			}
			opt.ListOptions.Page = resp.NextPage
			if l := len(issues); l > 0 {
				fmt.Printf("issues(%#v) +%#v since %#v = %#v/%#v\n", repo, l, *date, resp.NextPage, resp.LastPage)
			}
			if resp.NextPage == 0 {
				break
			}
		}
	}
	return result, nil
}

func getGithubIssueComments(owner string, repo string, client *github.Client, issueId int) []github.IssueComment {
	date := &DEFAULT_DATE
	clo := &github.IssueListCommentsOptions{Since: *date}
	issuesService := client.Issues
	comments := make([]github.IssueComment, 0)
	for {
		pageComments, resp, err := issuesService.ListComments(owner, repo, issueId, clo)
		fmt.Printf("Number of fetched comments %d\n", len(pageComments))
		if err != nil {
			panic(err)
		}
		comments = append(comments, pageComments...)
		fmt.Printf("Number of updated comments %d\n", len(comments))
		clo.ListOptions.Page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}
	return comments
}

func getGithubIssuesComments(owner string, repo string, client *github.Client) map[string][]github.IssueComment {
	date := &DEFAULT_DATE
	clo := &github.IssueListCommentsOptions{Since: *date}
	clo.ListOptions = github.ListOptions{PerPage: 100}
	issuesService := client.Issues
	result := make(map[string][]github.IssueComment)
	for {
		comments, resp, err := issuesService.ListComments(owner, repo, 0, clo)
		if err != nil {
			break
		}
		for _, comment := range comments {
			key := *comment.IssueURL
			list := result[key]
			if list == nil {
				list = make([]github.IssueComment, 0, 5)
			}
			result[key] = append(list, comment)
		}
		clo.ListOptions.Page = resp.NextPage
		if l := len(comments); l > 0 {
			fmt.Printf("comments(%#v) +%#v since %#v %#v/%#v\n", repo, l, *date, clo.ListOptions.Page, resp.LastPage)
		}
		if resp.NextPage == 0 {
			break
		}
	}

	return result
}

func convertGithubIssue(gIssue github.Issue, gComments []github.IssueComment) gorlim.Issue {
	fmt.Printf("convert %#v\n", *gIssue.Number)
	labelAmount := len(gIssue.Labels)
	labels := make([]string, 0, labelAmount)
	for i := 0; i < labelAmount; i++ {
		labels = append(labels, *gIssue.Labels[i].Name)
	}
	commentAmount := len(gComments)
	comments := make([]gorlim.Comment, 0, commentAmount)
	description := ""
	if ref := gIssue.Body; ref != nil {
		description = *ref
	}
	if commentAmount > 0 {
		for i := 0; i < commentAmount; i++ {
			gComment := gComments[i]
			author := ""
			if user := gComment.User; user != nil {
				author = *user.Login
			}
			comments = append(comments, gorlim.Comment{Text: *gComment.Body, Author: author, At: gComment.UpdatedAt})
		}
	}
	id := *gIssue.Number
	opened := (*gIssue.State) == "open"
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
	title := ""
	if ref := gIssue.Title; ref != nil {
		title = *ref
	}
	pullRequest := ""
	if pr := gIssue.PullRequestLinks; pr != nil {
		pullRequest = *pr.PatchURL
	}

	result := gorlim.Issue{
		Id:          id,
		At:          gIssue.CreatedAt,
		ClosedAt:    gIssue.ClosedAt,
		Opened:      opened,
		Creator:     creator,
		Assignee:    assignee,
		Milestone:   milestone,
		Title:       title,
		Description: description,
		PullRequest: pullRequest,
		Labels:      labels,
		Comments:    comments,
	}
	return result
}

func GetIssues(owner string, repo string, client *http.Client, date *time.Time) []gorlim.Issue {
	gh := github.NewClient(client)
	gIssues, err := getGithubIssues(owner, repo, gh, date)
	if err != nil {
		panic(err)
	}
	iss := make([]gorlim.Issue, 0, len(gIssues))
	var comments map[string][]github.IssueComment
	if date == nil {
		comments = getGithubIssuesComments(owner, repo, gh)
	} else {
		comments = make(map[string][]github.IssueComment)
		for _, issue := range gIssues {
			comments[*issue.URL] = getGithubIssueComments(owner, repo, gh, *issue.Number)
		}
	}
	noComments := make([]github.IssueComment, 0)
	for _, issue := range gIssues {
		value := comments[*issue.URL]
		if value == nil {
			value = noComments
		}

		iss = append(iss, convertGithubIssue(issue, value))
	}
	return iss
}

func UpdateIssue(owner string, repo string, client *http.Client, date time.Time, oldValue, newValue gorlim.Issue) error {
	// TBD: should we handle Response or errors are enough?
	// TBD: Creation of new milestones
	// TBD: support for creation of new issues (now only editing works)
	fmt.Println("github_gate.SetIssues")

	gh := github.NewClient(client)
	issueService := gh.Issues

	gIssue, _, _ := issueService.Get(owner, repo, newValue.Id)
	gComments := getGithubIssueComments(owner, repo, gh, newValue.Id)
	issue := convertGithubIssue(*gIssue, gComments)
	if !issue.Equals(oldValue) {
		fmt.Println(len(gComments))
		fmt.Println(newValue.Id)
		fmt.Println(len(oldValue.Comments))
		return errors.New("Github issue is different from origin")
	}
	// Update main fields
	request := github.IssueRequest{}
	if oldValue.Title != newValue.Title {
		request.Title = &newValue.Title
	}
	if oldValue.Description != newValue.Description {
		request.Body = &newValue.Description
	}
	if oldValue.Assignee != newValue.Assignee {
		request.Assignee = &newValue.Assignee
	}
	if oldValue.Milestone != newValue.Milestone {
		milestone, err := strconv.Atoi(newValue.Milestone)
		if err != nil {
			panic(err)
		}
		request.Milestone = &milestone
	}
	if oldValue.Opened != newValue.Opened {
		state := "closed"
		request.State = &state
	}
	request.Labels = newValue.Labels
	fmt.Printf("Edit request for issue send to github.issues %d\n", issue.Id)
	_, _, err := issueService.Edit(owner, repo, newValue.Id, &request)
	if err != nil {
		return err
	}
	removeComment := func(i int) (err error) {
		comment := gComments[i]
		_, err = issueService.DeleteComment(owner, repo, *comment.ID)
		return
	}
	editComment := func(i int, text string) (err error) {
		comment := gComments[i]
		comment.Body = &text
		_, _, err = issueService.EditComment(owner, repo, *comment.ID, &comment)
		return
	}
	addComment := func(text string) (err error) {
		comment := github.IssueComment{Body: &text}
		_, _, err = issueService.CreateComment(owner, repo, newValue.Id, &comment)
		return
	}
	// Update comments
	oldCommentsCount := len(oldValue.Comments)
	newCommentsCount := len(newValue.Comments)
	i := 0
	ni := 0
	for ; (i < oldCommentsCount) && (ni < newCommentsCount); i++ {
		oldComment := oldValue.Comments[i]
		newComment := newValue.Comments[ni]
		if oldComment.Author != newComment.Author { // Old comment was removed
			if oldComment.Author != owner {
				return errors.New("Cannot remove others comments")
			}
			if err := removeComment(i); err != nil {
				return err
			}
			continue
		}
		if oldComment.Text != newComment.Text {
			if err := editComment(i, newComment.Text); err != nil {
				return err
			}
		}
		ni++
	}
	if i != oldCommentsCount {
		// Remove comments
		for ; i < oldCommentsCount; i++ {
			if err := removeComment(i); err != nil {
				return err
			}
		}

	} else if ni != newCommentsCount {
		for ; ni < newCommentsCount; ni++ {
			newComment := newValue.Comments[ni]
			if err := addComment(newComment.Text); err != nil {
				return err
			}
		}
	}

	return nil
}
