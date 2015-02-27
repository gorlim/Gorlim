package gorlim

import "time"

type IssueRepositoryInterface interface {
	GetIssue(id int) (Issue, bool) 
	GetIssues() ([]Issue, []time.Time)
	Commit(string, []Issue, time.Time, *string) 

	// StartCommitGroup/EndCommitGroup are used on import to avoid
	// multiple open/close of connection to repo
	// TODO: think if some more clear interface may be provided
	StartCommitGroup() 
	EndCommitGroup()

	Id() int
	Path() string
}

func CreateRepo (repoPath string) IssueRepositoryInterface {
  repo := issueRepository{}
  repo.initialize(repoPath)
  return &repo
}