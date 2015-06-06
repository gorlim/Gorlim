package gorlim

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/libgit2/git2go"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// threshold for number of commits to repo
// before issuing next "garbage collection"
const gcThreshold = 2048

type issueRepository struct {
	path         string
	gcCounter    int
	repo         *git.Repository
	mutex        *sync.Mutex
	prePushHook  PrePushHook
	pendingMoves map[string]int
}

var mutex = &sync.Mutex{}
var id int = 0
var repoMap map[string]*issueRepository = make(map[string]*issueRepository)

type ExtendedCommitDiff struct {
	CommitDiff
	newIssuePathes []string
}

func NewGitRepo(repoPath string) IssueRepositoryInterface {
	repo := issueRepository{}
	repo.initializeNewRepo(repoPath)
	return &repo
}

func CreateFromExistingGitRepo(repoPath string) IssueRepositoryInterface {
	repo := issueRepository{}
	repo.initRepoObject(repoPath)
	return &repo
}

func (r *issueRepository) SetPrePushHook(pph PrePushHook) {
	r.prePushHook = pph
}

func (r *issueRepository) lock() {
	r.mutex.Lock()
}

func (r *issueRepository) unlock() {
	r.mutex.Unlock()
}

func (r *issueRepository) initRepoObject(repoPath string) {
	r.path = repoPath
	r.gcCounter = 0
	r.mutex = &sync.Mutex{}
	// save to repo map
	repoMap[repoPath] = r
}

func (r *issueRepository) initializeNewRepo(repoPath string) {
	r.initRepoObject(repoPath)
	// create physical repo
	repo, err := git.InitRepository(r.path, false)
	if err != nil {
		panic("Failed to create repo with path " + repoPath)
	}
	repo.Free()
	// configure
	setIgnoreDenyCurrentBranch(r.path) // allow push to non-bare repo
	// setup pre-receive hook
	pre, err := os.Create(r.path + "/.git/hooks/pre-receive")
	if err != nil {
		panic(err)
	}
	defer pre.Close()
	pre.Chmod(0777)
	pre.WriteString("#!/bin/sh\n")
	pre.WriteString("read oldrev newrev refname\n")
	pre.WriteString(os.Getenv("GOPATH") + "/gorlim_hook pre_push " + repoPath + " $oldrev $newrev\n")
	// setup post-receive hook
	post, err := os.Create(r.path + "/.git/hooks/post-receive")
	if err != nil {
		panic(err)
	}
	defer post.Close()
	post.Chmod(0777)
	post.WriteString("#!/bin/sh\n")
	post.WriteString("read oldrev newrev refname\n")
	post.WriteString(os.Getenv("GOPATH") + "/gorlim_hook post_push " + repoPath + " $oldrev $newrev\n")
	return
}

func blobToIssue(blob []byte) []string {
	// Well, that is potentially wrong because of different encodings
	str := string(blob[:len(blob)])
	return strings.Split(str, "\n")
}

func (r *issueRepository) diff(oldOid *git.Oid, newOid *git.Oid) ExtendedCommitDiff {
	repo := r.repo

	// find commits
	c1, err := repo.LookupCommit(oldOid)
	if err != nil {
		panic("Failed to find commit by oid")
	}
	c2, err := repo.LookupCommit(newOid)
	if err != nil {
		panic("Failed to find commit by oid")
	}
	// get commit trees
	t1, err := c1.Tree()
	if err != nil {
		panic(err)
	}
	t2, err := c2.Tree()
	if err != nil {
		panic(err)
	}
	// get diff
	diffOpts, _ := git.DefaultDiffOptions()
	diff, err := repo.DiffTreeToTree(t1, t2, &diffOpts)
	if err != nil {
		panic(err)
	}
	parseIssue := func(file git.DiffFile) (issue Issue, isNew bool) {
		blob, _ := repo.LookupBlob(file.Oid)
		issue = Issue{}
		if isNew = isNewIssueRepoPath(file.Path); !isNew {
			issue.Id, _ = parseIssueIdFromRepoPath(file.Path)
		}
		parseIssuePropertiesFromRepoPath(file.Path, &issue)
		parseIssuePropertiesFromText(blobToIssue(blob.Contents()), &issue)
		return
	}
	modifiedIssuesMap := make(map[int]struct {
		Old Issue
		New Issue
	})
	var newIssues []Issue
	var newIssuePathes []string
	callback := func(dd git.DiffDelta, f float64) (git.DiffForEachHunkCallback, error) {
		switch dd.Status {
		case git.DeltaModified:
			oldIssue, _ := parseIssue(dd.OldFile)
			newIssue, _ := parseIssue(dd.NewFile)
			modifiedIssuesMap[oldIssue.Id] = struct {
				Old Issue
				New Issue
			}{oldIssue, newIssue}
		case git.DeltaAdded:
			newIssue, isNew := parseIssue(dd.NewFile)
			mod, ok := modifiedIssuesMap[newIssue.Id]
			if isNew {
				newIssues = append(newIssues, newIssue)
				newIssuePathes = append(newIssuePathes, dd.NewFile.Path)
			} else {
				if ok {
					modifiedIssuesMap[newIssue.Id] = struct {
						Old Issue
						New Issue
					}{mod.Old, newIssue}
				} else {
					modifiedIssuesMap[newIssue.Id] = struct {
						Old Issue
						New Issue
					}{Issue{}, newIssue}
				}
			}
		case git.DeltaDeleted:
			oldIssue, _ := parseIssue(dd.OldFile)
			mod, ok := modifiedIssuesMap[oldIssue.Id]
			if ok {
				modifiedIssuesMap[oldIssue.Id] = struct {
					Old Issue
					New Issue
				}{oldIssue, mod.New}
			} else {
				modifiedIssuesMap[oldIssue.Id] = struct {
					Old Issue
					New Issue
				}{oldIssue, Issue{}}
			}
		}
		return nil, nil
	}
	if err = diff.ForEach(callback, git.DiffDetailFiles); err != nil {
		panic(err)
	}
	// return
	modifiedIssues := make([]struct {
		Old Issue
		New Issue
	}, len(modifiedIssuesMap), len(modifiedIssuesMap))
	index := 0
	for _, m := range modifiedIssuesMap {
		modifiedIssues[index] = m
		index++
	}
	return ExtendedCommitDiff{
		CommitDiff: CommitDiff{
			NewIssues:      newIssues,
			ModifiedIssues: modifiedIssues,
		},
		newIssuePathes: newIssuePathes,
	}
}

func setIgnoreDenyCurrentBranch(rpath string) {
	// this is an ugly hack to add config record - git.Config interfaces didn't work for me... TBD...
	cfgpath := rpath + "/.git/config"
	file, err := os.Open(cfgpath)
	if err != nil {
		panic(err)
	}
	content := readTextFile(file)
	file.Close()
	file, err = os.OpenFile(cfgpath, os.O_WRONLY, 0666)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	content = append(content, "\n[receive]")
	content = append(content, "        denyCurrentBranch = ignore\n")
	for _, str := range content {
		_, err := file.WriteString(str + "\n")
		if err != nil {
			panic(err)
		}
	}
}

// TODO: rewrite this method to avoid parsing all issues
func (r *issueRepository) GetIssue(id int) (Issue, bool) {
	issues, _ := r.GetIssues()
	for _, issue := range issues {
		if issue.Id == id {
			return issue, true
		}
	}
	return Issue{}, false
}

func (r *issueRepository) GetIssues() ([]Issue, []time.Time) {
	r.establishExclusiveRepoConnection()
	defer r.closeExclusiveRepoConnection()
	repo := r.repo

	idx, err := repo.Index()
	if err != nil {
		panic(err)
	}

	issuesCount := idx.EntryCount()
	issues := make([]Issue, issuesCount)
	timestamps := make([]time.Time, issuesCount)

	for i := 0; i < int(issuesCount); i++ {
		ientry, _ := idx.EntryByIndex(uint(i))
		path := ientry.Path
		id, _ := parseIssueIdFromRepoPath(path)
		issue := Issue{Id: id}
		parseIssuePropertiesFromRepoPath(path, &issue)
		file, err := os.OpenFile(r.path+"/"+path, os.O_RDONLY, 0666)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		status := parseIssuePropertiesFromText(readTextFile(file), &issue)
		if status == false {
			panic("Issue parse failed")
		}
		issues[i] = issue
		timestamps[i] = ientry.Mtime
	}
	return issues, timestamps
}

func readTextFile(file *os.File) []string {
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func isNewIssueRepoPath(path string) bool {
	split := strings.Split(path, "/")
	last := split[len(split)-1]
	return strings.HasPrefix(last, "new_")
}

func parseIssueIdFromRepoPath(path string) (int, error) {
	split := strings.Split(path, "/")
	last := split[len(split)-1]
	if last[0] == '#' {
		id, err := strconv.Atoi(last[1:])
		if err != nil {
			return -1, err
		}
		return id, nil
	} else {
		return -1, errors.New("Wrong issue id: " + last)
	}
}

func parseIssuePropertiesFromRepoPath(path string, issue *Issue) {
	split := strings.Split(path, "/")
	issue.Opened = split[0] == "open"
	splitIndex := 1
	if split[splitIndex][0] != '@' && split[splitIndex][0] != '#' {
		issue.Milestone = split[splitIndex]
		splitIndex++
	}
	if split[splitIndex][0] == '@' {
		issue.Assignee = split[splitIndex][1:]
		splitIndex++
	}
}

func parseIssuePropertiesFromText(text []string, issue *Issue) bool {
	join := strings.Join(text, "")
	err := yaml.Unmarshal([]byte(join), issue)
	return err == nil
}

var newLine = regexp.MustCompile("\r?\n")
var specificYaml = regexp.MustCompile("[-:#?,{}\r\n\\[\\]]")

func writeValue(b *bytes.Buffer, offset string, prefix string, value string) {
	b.WriteString(offset)
	b.WriteString(prefix)
	if specificYaml.FindString(value) == "" {
		b.WriteString(value)
	} else {
		b.WriteString("|")
		newOffset := offset + "  "
		for _, s := range newLine.Split(value, -1) {
			b.WriteString("\n")
			b.WriteString(newOffset)
			b.WriteString(s)
		}
	}
	b.WriteString("\n")
}

func writeValues(b *bytes.Buffer, offset string, prefix string, value []string, join string) {
	b.WriteString(prefix)
	if len(value) == 0 {
		b.WriteString("[]")
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
		newOffset := offset + "  "
		for _, v := range value {
			writeValue(b, newOffset, "- ", v)
			b.WriteString(join)
		}
	}
}

func issueToText(issue Issue) string {
	buffer := &bytes.Buffer{}
	writeValue(buffer, "", "title: ", issue.Title)
	writeValue(buffer, "", "assignee: ", issue.Assignee)
	writeValue(buffer, "", "milestone: ", issue.Milestone)
	writeValue(buffer, "", "patch: ", issue.PullRequest)
	writeValues(buffer, "", "labels: ", issue.Labels, "")
	delim := "############################################################\n"
	buffer.WriteString(delim)
	writeValue(buffer, "", "body: ", issue.Description)
	buffer.WriteString(delim)
	length := len(issue.Comments)
	comments := make([]string, length, length)
	index := 0
	for _, m := range issue.Comments {
		comments[index] = m.Text
		index++
	}
	writeValues(buffer, "", "comments: ", comments, delim)
	return buffer.String()
}

func getIssueDir(issue Issue) string {
	var buffer bytes.Buffer

	if issue.Opened {
		buffer.WriteString("open/")
	} else {
		buffer.WriteString("close/")
	}

	if issue.Milestone != "" {
		buffer.WriteString(issue.Milestone + "/")
	}

	if issue.Assignee != "" {
		buffer.WriteString("@" + issue.Assignee + "/")
	}

	return buffer.String()
}

func getIssueFileName(issue Issue) string {
	return "#" + strconv.Itoa(issue.Id) + ".yml"
}

func mkIssueIdToPathMap(idx *git.Index) map[int]string {
	issuesCount := idx.EntryCount()
	idToPathMap := make(map[int]string)
	for i := 0; i < int(issuesCount); i++ {
		ientry, _ := idx.EntryByIndex(uint(i))
		split := strings.Split(ientry.Path, "#")
		id, _ := strconv.Atoi(split[len(split)-1])
		idToPathMap[id] = ientry.Path
	}
	return idToPathMap
}

func (r *issueRepository) StartCommitGroup() {
	r.establishExclusiveRepoConnection()
}

func (r *issueRepository) EndCommitGroup() {
	defer r.closeExclusiveRepoConnection()
}

func (r *issueRepository) establishExclusiveRepoConnection() {
	r.lock()
	repo, err := git.OpenRepository(r.path)
	if err != nil {
		panic(err)
	}
	copts := &git.CheckoutOpts{Strategy: git.CheckoutForce}
	repo.CheckoutHead(copts) // sync local dir
	r.repo = repo
}

func (r *issueRepository) closeExclusiveRepoConnection() {
	r.repo.Free()
	r.repo = nil
	r.unlock()
}

func (r *issueRepository) createMoveCommitForNewIssues(moves map[string]int) {
	repo := r.repo
	idx, err := repo.Index()
	if err != nil {
		panic(err)
	}
	for oldPath, id := range moves {
		issue := Issue{Id: id}
		parseIssuePropertiesFromRepoPath(oldPath, &issue)
		newPath := getIssueDir(issue) + getIssueFileName(issue)
		err := os.Rename(r.path+"/"+oldPath, r.path+"/"+newPath)
		if err != nil {
			panic(err)
		}
		// update index
		if err := idx.RemoveByPath(oldPath); err != nil {
			panic(err)
		}
		err = idx.AddByPath(newPath)
		if err != nil {
			panic(err)
		}
	}
	// write index to filesystem
	treeId, err := idx.WriteTree()
	if err != nil {
		panic(err)
	}
	if err = idx.Write(); err != nil {
		panic(err)
	}
	tree, err := repo.LookupTree(treeId)
	if err != nil {
		panic(err)
	}
	// get head commit
	head, _ := repo.Head()
	headCommit, err := repo.LookupCommit(head.Target())
	if err != nil {
		panic(err)
	}
	// do move commit
	signature := &git.Signature{Name: "gorlim", Email: "none", When: time.Now()}
	_, err = repo.CreateCommit("refs/heads/master", signature, signature, "rename new issues", tree, headCommit)
	if err != nil {
		panic(err)
	}
}

func (r *issueRepository) Commit(message string, issues []Issue, tm time.Time, updateAuthor *string) {
	if r.repo == nil { // Can be non-nil if we are inside commit group
		r.establishExclusiveRepoConnection()
		defer r.closeExclusiveRepoConnection()
	}

	repo := r.repo

	idx, err := repo.Index()
	if err != nil {
		panic(err)
	}

	idToPathMap := mkIssueIdToPathMap(idx)

	for _, issue := range issues {
		dir := r.path + "/" + getIssueDir(issue)
		repopath := getIssueDir(issue) + getIssueFileName(issue)
		filepath := r.path + "/" + repopath
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			panic(err)
		}
		file, err := os.Create(filepath)
		if err != nil {
			panic(err)
		}
		file.WriteString(issueToText(issue))
		file.Close()
		err = idx.AddByPath(repopath)
		if err != nil {
			panic(err)
		}
		// if old path to issue was different, then we need to delete old version
		oldPath, ok := idToPathMap[issue.Id]
		if ok && (oldPath != repopath) {
			if err := os.Remove(r.path + "/" + oldPath); err != nil {
				panic(err)
			}
			if err := idx.RemoveByPath(oldPath); err != nil {
				panic(err)
			}
		}
	}
	treeId, err := idx.WriteTree()
	if err != nil {
		panic(err)
	}
	if err = idx.Write(); err != nil {
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
	// check if author is the same
	author := ""
	if updateAuthor != nil {
		author = *updateAuthor
	} else {
		singleAuthor := true
		for _, issue := range issues {
			if author == "" {
				author = issue.Creator
			} else if author != issue.Creator {
				singleAuthor = false
				break
			}
		}
		if singleAuthor == false {
			author = "multiple authors"
		}
	}
	signature := &git.Signature{Name: author, Email: "none", When: tm}
	if headCommit != nil {
		_, err = repo.CreateCommit("refs/heads/master", signature, signature, message, tree, headCommit)
	} else {
		_, err = repo.CreateCommit("refs/heads/master", signature, signature, message, tree)
	}
	if err != nil {
		panic(err)
	}

	r.gcCounter++
	if r.gcCounter >= gcThreshold {
		r.doGarbageCollection()
		fmt.Println("Do garbage collection")
		r.gcCounter = 0
	}
}

func (r *issueRepository) Path() string {
	return r.path
}

func (r *issueRepository) doGarbageCollection() {
	// libgit2 API does not inlude explicit "gc" or "repack" command
	// It (API) is too low-level, so we would need to implement our own repack algorithm
	// Let's do this "hack" instead
	_, err := exec.Command("git", "--git-dir="+r.path+"/.git", "gc").CombinedOutput()
	if err != nil {
		panic(err)
	}
}
