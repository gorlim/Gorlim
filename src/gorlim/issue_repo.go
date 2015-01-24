package gorlim

import "github.com/libgit2/git2go"
import "fmt"
import "strconv"
import "os"

type IssueRepositoryInterface interface {
	GetIssue(id int) (Issue, bool)
	GetIssues() []Issue
	Update(string, []Issue) 
	Id() int
	Path() string
}

type IssueRepository struct {
   id int
   path string
}

func (r IssueRepository) initialize (repoRoot string, id int) {
	r.id = id
	r.path = repoRoot + "/" + "issues" + strconv.Itoa(id)
	// create physical repo
	_, error := git.InitRepository(r.path,  true)
	fmt.Println("error", error)  
	// setup hooks
	file, _ := os.Create(r.path + "/hooks/pre-receive")
    defer file.Close()
	file.Chmod(0777)
	file.WriteString("#!/bin/sh\n")
	file.WriteString("exit 1\n")
	return 
}

func (r IssueRepository) GetIssue(id int) (Issue, bool) {
	panic("Repository:GetIssue not implemented")
}

func (r IssueRepository) GetIssues() []Issue {
	panic("Repository:GetIssues not implemented")
}

func (r IssueRepository) Update(string, []Issue) {
   panic("Repository:Update not implemented")	
}

func (r IssueRepository) Id() int {
	return r.id
}

func (r IssueRepository) Path() string {
	return r.path
}

func CreateRepo (repoRoot string, id int) IssueRepositoryInterface {
  repo := IssueRepository{}
  repo.initialize(repoRoot, id)
  return repo
}
