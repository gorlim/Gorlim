package gorlim

import "time"

type WebIssuesInterface interface {

	SetIssues(uri string, issues []Issue)
	GetIssues(uri string, date *time.Time) []Issue
	CreateIssuesUpdateChannel(uri string) <-chan IssuesUpdate
}