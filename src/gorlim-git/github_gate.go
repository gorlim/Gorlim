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

func getIssues(accessToken, date, repo string) []github.issues {
	if date == "" {
		date = "Sat, 24 Jan 2015 00:00:00 GMT"
	}
	t := &AuthenticatedTransport{
		AccessToken: accessToken,
		Date:        date,
	}
	client := github.NewClient(t.Client())
	client.BaseURL, _ = url.Parse("https://api.github.com/repos/" + repo)
	ilo := &github.IssueListOptions{}
	issuesService := client.Issues
	issues, resp, err := issuesService.List(true, ilo)

	if err != nil {
		fmt.Printf("error: %#v %#v\n\n", err, resp)
	} else {
		return issues
	}
}
