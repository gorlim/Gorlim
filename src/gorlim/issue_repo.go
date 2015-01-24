package gorlim

import "github.com/libgit2/git2go"
import "strconv"
import "os"
import "syscall"
import "fmt"
import "bytes"
import "strings"
import "bufio"

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
   repo *git.Repository
}

func (r *IssueRepository) initialize (repoRoot string, id int) {
	r.id = id
	r.path = repoRoot + "/" + "issues" + strconv.Itoa(id)
	// create physical repo
	r.repo, _ = git.InitRepository(r.path,  false)
	// setup hooks pipe and subscribe to it
    pipeName := os.Getenv("HOME") + "/pushntfifo" // TODO hardcoded
	// setup pre-receive hook
	pre, _ := os.Create(r.path + ".git/hooks/pre-receive")
    defer pre.Close()
	pre.Chmod(0777)
	pre.WriteString("#!/bin/sh\n")
	pre.WriteString("exit 0\n")
	// setup post-receive hook
	post, _ := os.Create(r.path + ".git/hooks/post-receive")
    defer post.Close()
	post.Chmod(0777)
	post.WriteString("#!/bin/sh\n")
	post.WriteString("echo " + strconv.Itoa(id) + " >" + pipeName)
	return 
}

func (r *IssueRepository) GetIssue(id int) (Issue, bool) {
	panic("Repository:GetIssue not implemented")
}

func (r *IssueRepository) GetIssues() []Issue {
	repo := r.repo

    repo.CheckoutHead(nil) // sync local dir 

    idx, err := repo.Index()
    if err != nil {
		panic(err)
    }

    issuesCount := idx.EntryCount()

    issues := make([]Issue, issuesCount)
	
	for i := 0; i < int(issuesCount); i++ {
		ientry, _ := idx.EntryByIndex(uint(i))
		path := ientry.Path
		split := strings.Split(path, "/")
		issue := Issue { Opened : split[0] == "open", Milestone : split[1], Assignee : split[2][1:] }
		id := split[3][1:]
		issue.Id, _ = strconv.Atoi(id)
		file, err := os.OpenFile(r.path + "/" + path, os.O_RDONLY, 0666)
		if err != nil {
			panic (err)
		}	
		defer file.Close()
		status := parseIssuePropertiesFromText(readTextFile(file), &issue)
		if status == false {
			panic ("Issue parse failed")
		}
		issues[i] = issue
	}

	return issues
}

func readTextFile(file *os.File) []string {
	var lines []string
  	scanner := bufio.NewScanner(file)
  	for scanner.Scan() {
    	lines = append(lines, scanner.Text())
  	}
  	return lines
}

const delimiter string = "----------------------------------";

func parseIssuePropertiesFromText(text []string, issue *Issue) bool {
	i := 0
	textLength := len(text)
	// Parse Title
	for ; i < textLength; i++ {
		if (strings.Contains(text[i], "Title:")) {
			issue.Title = strings.TrimSpace(strings.Split(text[i], ":")[1])
			i++
			break;
		}
	}
	if i == textLength {
		panic("panic")
		return false
	}
	// Parse Labels
	for ; i < textLength; i++ {
		if (strings.Contains(text[i], "Labels:")) {
			split  := strings.Split(text[i], ":")
			labels := split[1]
			split   = strings.Split(labels, ",")
			for _, label := range split {
				issue.Labels = append(issue.Labels, strings.TrimSpace(label))
			}
			i++
			break;
		}
	}
	if i == textLength {
		panic("panic")
		return false
	}
	// Parse description
	if text[i] == delimiter {
		i++
	} else{
		panic("panic")
		return false
	}
	for ; i < textLength; i++ {
		if (text[i] == delimiter) {
			break
		}
		issue.Description = issue.Description + text[i]
	}
	if i == textLength {
		return true
	}
	// Parse comments
	if text[i] == delimiter {
		i++
	} else{
		panic("panic")
		return false
	}
	comment := ""
	for ; i < textLength; i++ {
		if text[i] == delimiter {
			issue.Comments = append(issue.Comments, comment)
			comment = ""
			continue
		}
		comment = comment + text[i]
	}
    return true
}

func issueToText(issue Issue) string {
   	var buffer bytes.Buffer

    buffer.WriteString("Title: " + issue.Title + "\n\n");

    buffer.WriteString("Labels: ")
    for i, label := range issue.Labels {
    	if i > 0 {
    		buffer.WriteString(", ")
    	}
     	buffer.WriteString(label)
    }
    buffer.WriteString("\n" + delimiter + "\n")
    
    buffer.WriteString(issue.Description)
    buffer.WriteString("\n" + delimiter + "\n")

    for i, comment := range issue.Comments {
    	if i > 0 {
    		buffer.WriteString("\n" + delimiter + "\n")
    	}
    	buffer.WriteString(comment)
    }

    buffer.WriteString("\n")

    return buffer.String()
}

func getIssueDir(issue Issue) string {
	var buffer bytes.Buffer

	if issue.Opened {
		buffer.WriteString("open/")	
	} else {
		buffer.WriteString("close/")	
	}

	buffer.WriteString(issue.Milestone + "/")

	buffer.WriteString("@" + issue.Assignee)

	return buffer.String()
}

func getIssueFileName(issue Issue) string {
	return "#" + strconv.Itoa(issue.Id)
}

func (r *IssueRepository) Update(message string, issues []Issue) {  
   repo := r.repo

   repo.CheckoutHead(nil) // sync local dir 

   idx, err := repo.Index()
   if err != nil {
		panic(err)
   }

   for _, issue := range issues {
      	dir := r.path + "/" + getIssueDir(issue)
      	repopath := getIssueDir(issue) + "/" + getIssueFileName(issue)
      	filepath := r.path + "/" + repopath
      	err := os.MkdirAll(dir, 0777)
      	if err != nil {
      		panic(err)
      	}
      	file, err := os.Create(filepath);
      	if err != nil {
      		panic(err)
      	}
      	file.WriteString(issueToText(issue))
      	file.Close();   

        err = idx.AddByPath(repopath)
   	    if err != nil {
       		panic(err)
   		}
   }   

   treeId, err := idx.WriteTree()
   if err != nil {
  	 panic(err)
   }

   err = idx.Write()
   if err != nil {
	 panic(err)
   }

   tree, err := repo.LookupTree(treeId)
   if err != nil {
	 panic(err)
   }

   head, _ := repo.Head()

   var headCommit *git.Commit

   if head != nil {
   		headCommit, err = repo.LookupCommit(head.Target())
   		if err != nil {
     		panic(err)
   		}
	}

    signature := &git.Signature{Name:"a", Email:"b"}
    if headCommit != nil {
   		_, err = repo.CreateCommit("refs/heads/master", signature, signature, message, tree, headCommit)
	} else {
		_, err = repo.CreateCommit("refs/heads/master", signature, signature, message, tree)	
	}
    if err != nil {
	    panic(err)
    }
}

func (r *IssueRepository) Id() int {
	return r.id
}

func (r *IssueRepository) Path() string {
	return r.path
}

func CreateRepo (repoRoot string, id int) IssueRepositoryInterface {
  repo := IssueRepository{}
  repo.initialize(repoRoot, id)
  return &repo
}

var pushevent chan int

func GetPushListener() <-chan int {
	if pushevent == nil {
	  	pushevent = make(chan int, 16) // TODO buffer size
      	pipeName := os.Getenv("HOME") + "/pushntfifo" // TODO hardcoded, TODO cleanup
	  	syscall.Mkfifo(pipeName, 0666)
      	SubscribeToPushEvent(pipeName, pushevent)
    }
    fmt.Println("FU GO")
    return pushevent

}

