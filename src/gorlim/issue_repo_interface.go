package gorlim

import "time"

type IssueRepositoryInterface interface {
	GetIssue(id int) (Issue, bool) // not-implemented yet
	GetIssues() ([]Issue, []time.Time)
	Update(string, []Issue) 
	Id() int
	Path() string
}

func CreateRepo (repoRoot string, id int) IssueRepositoryInterface {
  repo := issueRepository{}
  repo.initialize(repoRoot, id)
  return &repo
}