package gorlim

type IssueRepositoryInterface interface {
	GetIssue(id int) (Issue, bool)
	GetIssues() []Issue
	Update(string, []Issue) 
	Id() int
	Path() string
}

func CreateRepo (repoRoot string, id int) IssueRepositoryInterface {
  repo := issueRepository{}
  repo.initialize(repoRoot, id)
  return &repo
}