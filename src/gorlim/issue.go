package gorlim

type Issue struct {
  id int
  state bool
  assignee string
  milestone string
  title string
  description string
  labels []string
  comments []string
}

