package gorlim

import "time"

type Comment struct {
	Author string
	Text   string
	At     time.Time
}

type Issue struct {
	Id          int
	Opened      bool
	Creator     string
	Assignee    string
	Milestone   string
	Title       string
	Description string
	Labels      []string
	Comments    []*Comment
}
