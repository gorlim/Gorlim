package gorlim

import "time"
import "fmt"

type IssuesUpdate struct {
	Uri    string
	Issues []Issue
}

// TBD: probably we want individual sync manager for each repo/web pair
type SyncManager struct {
	webIssues WebIssuesInterface
}

func CreateSyncManager(iWebIssues WebIssuesInterface) *SyncManager {
	return &SyncManager{
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
}

func (sm *SyncManager) EstablishSync(uri string, repo IssueRepositoryInterface) {
	// subscribe to web updates
	sm.listenToWebUpdateEvent(sm.webIssues.CreateIssuesUpdateChannel(uri), repo)
	// subscribe to pre-push event
	sm.subscribeToPrePushEvent(repo, uri)
}

func (sm *SyncManager) subscribeToPrePushEvent(repo IssueRepositoryInterface, webUri string) {
	// TODO: force sync
	repo.SetPrePushHook(
		func(commitDiff CommitDiff) error {
			for _, mod := range commitDiff.ModifiedIssues {
				err := sm.webIssues.UpdateIssue(webUri, mod.Old, mod.New)
				if err != nil {
					return err
				}
			}
			// TODO: new Issues
			return nil
		})
}

// Simple implementation of web-to-git updater - do not care that commit may come from the user in the same time for starters
func (sm *SyncManager) listenToWebUpdateEvent(webupdate <-chan IssuesUpdate, repo IssueRepositoryInterface) {
	go func() {
		for wupd := range webupdate {
			issues := wupd.Issues
			for _, issue := range issues {
				repo.Commit("webimport: "+issue.Title, []Issue{issue}, time.Now(), nil)
			}
		}
	}()
}
