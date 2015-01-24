package gorlim

import "github.com/libgit2/git2go"
import "strconv"
import "os"
import "syscall"

type IssueRepositoryInterface interface {
	GetIssue(id int) (Issue, bool)
	GetIssues() []Issue
	Update(string, []Issue) 
	Id() int
	Path() string
	//PushEvent() <-chan string // TODO: maybe it would be better to have push event for every repo instead of global one
}

type IssueRepository struct {
   id int
   path string
}

func (r IssueRepository) initialize (repoRoot string, id int) {
	r.id = id
	r.path = repoRoot + "/" + "issues" + strconv.Itoa(id)
	// create physical repo
	git.InitRepository(r.path,  true)
	// setup hooks pipe and subscribe to it
    pipeName := os.Getenv("HOME") + "/pushntfifo" // TODO hardcoded
	// setup pre-receive hook
	pre, _ := os.Create(r.path + "/hooks/pre-receive")
    defer pre.Close()
	pre.Chmod(0777)
	pre.WriteString("#!/bin/sh\n")
	pre.WriteString("exit 0\n")
	// setup post-receive hook
	post, _ := os.Create(r.path + "/hooks/post-receive")
    defer post.Close()
	post.Chmod(0777)
	post.WriteString("#!/bin/sh\n")
	post.WriteString("echo " + strconv.Itoa(id) + " >" + pipeName)
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

var pushevent chan int

func GetPushListener() <-chan int {
	if pushevent == nil {
	  	pushevent = make(chan int, 16) // TODO buffer size
      	pipeName := os.Getenv("HOME") + "/pushntfifo" // TODO hardcoded, TODO cleanup
	  	syscall.Mkfifo(pipeName, 0666)
      	SubscribeToPushEvent(pipeName, pushevent)
    }
    return pushevent
}

