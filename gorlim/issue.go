package gorlim

import "time"

type Comment struct {
	Author string
	Text   string
	At     *time.Time
}

type Issue struct {
	Id          int        `yaml:"-"`
	Opened      bool       `yaml:"-"`
	Creator     string     `yaml:"-"`
	At          *time.Time `yaml:"-"`
	ClosedAt    *time.Time `yaml:"-"`
	Title       string
	Assignee    string
	Milestone   string
	PullRequest string `yaml:"patch"`
	Labels      []string
	Description string `yaml:"body"`
	Comments    []Comment
}

func (issue Issue) Equals(other Issue) bool {
	// TBD: ignored Creator, CloseAt, At because they are not git-saveable
	if issue.Id != other.Id {
		return false
	}
	if issue.Opened != other.Opened {
		return false
	}
	if issue.Assignee != other.Assignee {
		return false
	}
	if issue.Milestone != other.Milestone {
		return false
	}
	if issue.Title != other.Title {
		return false
	}
	if issue.Description != other.Description {
		return false
	}
	if issue.PullRequest != other.PullRequest {
		return false
	}
	if len(issue.Labels) != len(other.Labels) {
		return false
	}
	for i, label := range issue.Labels {
		if other.Labels[i] != label {
			return false
		}
	}
	if len(issue.Comments) != len(other.Comments) {
		return false
	}
	for i, comment := range issue.Comments {
		if other.Comments[i].Text != comment.Text ||
			other.Comments[i].Text != comment.Text {
			return false
		}
	}
	return true
}

func (v Comment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(v.Text)
}
func (v Comment) MarshalYAML() (interface{}, error) {
	return v.Text, nil
}
