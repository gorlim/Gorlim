package gorlim

import "time"

type GitWebPair struct {
	repo IssueRepositoryInterface
	uri  string // TBD corresponding object
}

type SyncManager struct {
	idToReposMap       map[int]GitWebPair
	uriToReposMap      map[string]GitWebPair
	webUpdateTimestamp time.Time
}

func (sm *SyncManager) AddRepository(webIssuesUri string, repo IssueRepositoryInterface) {
	gwp := GitWebPair{repo: repo, uri: webIssuesUri}
	sm.idToReposMap[repo.Id()] = gwp
	sm.uriToReposMap[webIssuesUri] = gwp
}

func Create() *SyncManager {
	return &SyncManager{
		idToReposMap:  make(map[int]GitWebPair),
		uriToReposMap: make(map[string]GitWebPair),
	}
}

func (sm *SyncManager) InitGetRepoFromIssues(webIssuesUri string, repo IssueRepositoryInterface) {
	repo.Update("initial commit", make([]Issue, 0)) // TBD - place to fetch Issues from web
}

func (sm *SyncManager) SubscribeToPushEvent(pushevent <-chan int) {
	sm.webUpdateTimestamp = time.Unix(0, 0)
	go func() {
		for push := range pushevent {
			// TBD here we can simply send current repo state to web interface
			repo := sm.idToReposMap[push].repo
			issues, timestamps := repo.GetIssues()
			currentTime := time.Now()
			for index, tm := range timestamps {
				// if modified later than last sync
				if time.Since(tm) < time.Since(sm.webUpdateTimestamp) {
					_ = issues[index]
					// TODO : send issue to web
				}
			}
			sm.webUpdateTimestamp = currentTime
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
