package gorlim

type Issue struct {
  Id int
  Opened bool
  Assignee string
  Milestone string
  Title string
  Description string
  Labels []string
  Comments []string
}

