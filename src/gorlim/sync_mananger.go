package gorlim

import "fmt"

type GitWebPair struct {
  repo IssueRepositoryInterface
  uri string // TBD corresponding object
}

type SyncManager struct {    
	idToReposMap map[int]GitWebPair
}

func (sm *SyncManager) AddRepository(webIssuesUri string, repo IssueRepositoryInterface) {
	sm.idToReposMap[repo.Id()] = GitWebPair{repo:repo, uri:webIssuesUri}
}

func (sm *SyncManager) InitGetRepoFromIssues(webIssuesUri string, repo IssueRepositoryInterface) {
	repo.Update("initial commit", make([]Issue, 0)) // TBD - place to fetch Issues from web

	// TBD: good optimization would be to ask repo for which issues were modified since the last known commit
}

func (sm *SyncManager) SubscribeToPushEvent(pushevent <-chan int) {
	go func () {
	  	for push := range pushevent {
  			fmt.Println("push to repo id ", push)
  			// TBD here we can simply send current repo state to web interface
			sm.idToReposMap[push].repo.GetIssues() 
  		}
	}()
}

