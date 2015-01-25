package gorlim

import "time"

type GitWebPair struct {
  repo IssueRepositoryInterface
  uri string // TBD corresponding object
}

type SyncManager struct {    
	idToReposMap map[int]GitWebPair
	webUpdateTimestamp time.Time
}

func (sm *SyncManager) AddRepository(webIssuesUri string, repo IssueRepositoryInterface) {
	sm.idToReposMap[repo.Id()] = GitWebPair{repo:repo, uri:webIssuesUri}
}

func (sm *SyncManager) InitGetRepoFromIssues(webIssuesUri string, repo IssueRepositoryInterface) {
	repo.Update("initial commit", make([]Issue, 0)) // TBD - place to fetch Issues from web
}

func (sm *SyncManager) SubscribeToPushEvent(pushevent <-chan int) {
	sm.webUpdateTimestamp = time.Unix(0, 0)
	go func () {
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

