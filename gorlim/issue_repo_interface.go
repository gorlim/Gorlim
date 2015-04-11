package gorlim

import "time"

type CommitDiff struct {
	NewIssues []Issue
	ModifiedIssues []struct {
		Old Issue
	    New Issue
	}
} 

type PrePushHook func(CommitDiff) error

type IssueRepositoryInterface interface {
	GetIssue(id int) (Issue, bool) 
	GetIssues() ([]Issue, []time.Time)
	Commit(string, []Issue, time.Time, *string) 

	SetPrePushHook(PrePushHook)

	// StartCommitGroup/EndCommitGroup are used on import to avoid
	// multiple open/close of connection to repo
	// TODO: think if some more clear interface may be provided
	StartCommitGroup() 
	EndCommitGroup()

	Path() string
}
