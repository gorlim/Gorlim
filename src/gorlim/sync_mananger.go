package gorlim

import "time"
import "fmt"

type GitWebPair struct {
	repo IssueRepositoryInterface
	uri  string // TBD corresponding object
	webUpdateTimestamp time.Time
}

type SyncManager struct {
	idToReposMap       map[int]GitWebPair
	uriToReposMap      map[string]GitWebPair	
}

// TBD: first parameter should be web issues interface
func (sm *SyncManager) AddRepository(webIssuesUri string, repo IssueRepositoryInterface) {
	gwp := GitWebPair{repo: repo, uri: webIssuesUri, webUpdateTimestamp:time.Now()}
	sm.idToReposMap[repo.Id()] = gwp
	sm.uriToReposMap[webIssuesUri] = gwp
}

func Create() *SyncManager {
	return &SyncManager{
		idToReposMap:  make(map[int]GitWebPair),
		uriToReposMap: make(map[string]GitWebPair),
	}
}

// TBD: idea is that we don't nee third parameter is first paramter will be real WebIssue interface with getIssues method
func (sm *SyncManager) InitGitRepoFromIssues(webIssuesUri string, repo IssueRepositoryInterface, issues []Issue) {
	for _, issue := range issues {
		repo.Update("import from web: "+issue.Title, []Issue{issue})
	}
	gwp := sm.idToReposMap[repo.Id()]
	gwp.webUpdateTimestamp = time.Now()
	sm.idToReposMap[repo.Id()] = gwp
}

func (sm *SyncManager) SubscribeToPushEvent(pushevent <-chan int) {
	go func() {
		for push := range pushevent {
			// TBD here we can simply send current repo state to web interface
			gwp := sm.idToReposMap[push]
			repo := gwp.repo
			issues, timestamps := repo.GetIssues()
			currentTime := time.Now()
			for index, tm := range timestamps {
				// if modified later than last sync
				if time.Since(tm) < time.Since(gwp.webUpdateTimestamp) {
					issue := issues[index]
					fmt.Println("Pushed issue", issue)
					// TODO : send issue to web
				}
			}
			gwp.webUpdateTimestamp = currentTime
			sm.idToReposMap[push] = gwp
		}
	}()
}

// Simple implementation of web-to-git updater - do not care that commit may come from the user in the same time for starters
func (sm *SyncManager) SubscribeToWebUpdateEvent(webupdate <-chan struct {
	string
	issues []Issue
}) {
	go func() {
		for wupd := range webupdate {
			uri := wupd.string
			issues := wupd.issues
			repo := sm.uriToReposMap[uri].repo
			for _, issue := range issues {
				singleIssueSlice := make([]Issue, 1)
				singleIssueSlice[0] = issue
				repo.Update("import from web: "+issue.Title, singleIssueSlice)
			}
		}
	}()
}
