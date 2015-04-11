package gorlim

import "time"

type WebIssuesInterface interface {
	UpdateIssue(uri string, oldValue Issue, newValue Issue) error
	GetIssues(uri string, date *time.Time) []Issue
	CreateIssuesUpdateChannel(uri string) <-chan IssuesUpdate
}