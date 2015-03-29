package gorlim

import "time"
import "fmt"

type IssuesInfoObject struct {
	GitRepo            IssueRepositoryInterface
	Uri                string
	WebUpdateTimestamp time.Time
}

type IssuesUpdate struct {
	Uri    string
	Issues []Issue
}

// TBD: probably we want individual sync manager for each repo/web pair
type SyncManager struct {
	idToInfoMap  map[int]IssuesInfoObject
	uriToInfoMap map[string]IssuesInfoObject
	webIssues WebIssuesInterface
}

func CreateSyncManager(iWebIssues WebIssuesInterface) *SyncManager {
	return &SyncManager{
		idToInfoMap:  make(map[int]IssuesInfoObject),
		uriToInfoMap: make(map[string]IssuesInfoObject),
		webIssues: iWebIssues,
	}
}

// TBD: we should be able to init repo info from existing db!
func (sm *SyncManager) InitGitRepoFromIssues(uri string, emptyGitRepo IssueRepositoryInterface) {
	issues := sm.webIssues.GetIssues(uri, nil)
	repo := emptyGitRepo
	repo.StartCommitGroup()
	importStartTime := time.Now()
	for _, issue := range issues {
		fmt.Printf("Started import of issue %d\n", issue.Id)
		issueImportStartTime := time.Now()

		issue1 := issue
		issue1.Comments = []Comment{}
		issue1.Opened = true
		repo.Commit("webimport Opened issue: " + issue.Title + " " + issue.Description, []Issue{issue1}, *issue.At, &issue.Creator)
		for i := 0; i < len(issue.Comments); i++ {
			issue1.Comments = issue.Comments[0:i] 
			repo.Commit(fmt.Sprintf("webimport: #%v", issue.Comments[i].Text), []Issue{issue1}, *issue.Comments[i].At, &issue.Comments[i].Author)
		}
		if issue.Opened == false {
			if issue.Assignee == "" {
				repo.Commit("webimport Closed issue: " + issue.Title, []Issue{issue}, *issue.ClosedAt, nil)
			} else {
				repo.Commit("webimport Closed issue: " + issue.Title, []Issue{issue}, *issue.ClosedAt, &issue.Assignee)
			}
		}

        issueImportEndTime := time.Now()
        timePassed := issueImportEndTime.Sub(issueImportStartTime)
		fmt.Printf("Finished import of issue %d ms %d\n", issue.Id, int64(timePassed/time.Millisecond))
	}
	importEndTime := time.Now()
	timePassed := importEndTime.Sub(importStartTime)
	fmt.Printf("Finished import of issues: sec %d\n", int64(timePassed/time.Second))
	repo.EndCommitGroup()
	// add web <-> repo connection
	info := IssuesInfoObject {
		GitRepo: emptyGitRepo,
		Uri: uri,
		WebUpdateTimestamp: time.Now(),
	}
	sm.idToInfoMap[emptyGitRepo.Id()] = info
	sm.uriToInfoMap[uri] = info
	// subscribe to web updates
	sm.listenToWebUpdateEvent(sm.webIssues.CreateIssuesUpdateChannel(uri))
}

func (sm *SyncManager) SubscribeToPrePushEvent(prePushEvent <-chan RepoPrePushMessage, reply chan<- RepoPrePushReply) {
	go func() {
		for prePush := range prePushEvent {
			info := sm.idToInfoMap[prePush.RepoId]
			repo := info.GitRepo
			fmt.Printf("Calling repo compare %d\n", prePush.RepoId)
			if repo == nil {
				panic("NIL!")
			}
			repo.Compare(prePush.Sha)
			/*
			issues, timestamps := repo.GetIssues()
			currentTime := time.Now()
			for index, tm := range timestamps {
				// if modified later than last sync
				if time.Since(tm) < time.Since(info.WebUpdateTimestamp) {
					issue := issues[index]
					fmt.Println("Pushed issue", issue)
					sm.webIssues.SetIssues(info.Uri, []Issue{issue})
				}
			}
			info.WebUpdateTimestamp = currentTime
			sm.idToInfoMap[push] = info*/
		}
	}()
}

// Simple implementation of web-to-git updater - do not care that commit may come from the user in the same time for starters
func (sm *SyncManager) listenToWebUpdateEvent(webupdate <-chan IssuesUpdate) {
	go func() {
		for wupd := range webupdate {
			uri := wupd.Uri
			issues := wupd.Issues
			repo := sm.uriToInfoMap[uri].GitRepo
			fmt.Println(uri)
			for _, issue := range issues {
				repo.Commit("webimport: "+issue.Title, []Issue{issue}, time.Now(), nil)
			}
		}
	}()
}
