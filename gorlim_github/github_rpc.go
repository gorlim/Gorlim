package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/gorlim/Gorlim/gorlim"
	"github.com/gorlim/Gorlim/storage"
	"io/ioutil"
	"log/syslog"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type GithubService struct{}

var converted = make(map[string]time.Time)
var inProgress = make(map[string]gorlim.CheckRepoResponse)
var mutex = new(sync.Mutex)

var root = flag.String("root", "/git", "Root for git repositories")
var port = flag.Int("port", 9999, "Port for listening")
var ghClient = flag.String("github-client", "", "GitHub Client Id for application")
var ghSecret = flag.String("github-secret", "", "GitHub Secret Id for application")
var dbFile = flag.String("db", "gorlim.db", "SQLite file with keys")
var syncManager *gorlim.SyncManager
var logger *syslog.Writer

func main() {
	flag.Parse()
	var err error
	logger, err = syslog.New(syslog.LOG_ERR|syslog.LOG_LOCAL0, "gorlim_github")
	if err != nil {
		panic(err)
	}
	var database *storage.Storage
	if database, err = storage.Create(*dbFile); err != nil {
		panic(err)
	}

	syncManager = gorlim.CreateSyncManager(&GithubWebIssuesInterface{
		db:       *database,
		clientId: *ghClient,
		secretId: *ghSecret,
	})
	users, _ := ioutil.ReadDir(*root)
	for _, user := range users {
		if !user.IsDir() {
			continue
		}
		prefix := path.Join(*root, user.Name())
		projects, _ := ioutil.ReadDir(prefix)
		for _, project := range projects {
			if !project.IsDir() {
				continue
			}
			path := path.Join(prefix, project.Name())
			uri := toUri(path)
			repo := gorlim.NewGitRepo(path)
			syncManager.EstablishSync(uri, repo)
			inProgress[path] = gorlim.CheckRepoResponse{
				LastConvertedEventTime: time.Unix(0, 0),
				DoneRatio:              1,
			}
		}
	}

	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterService(new(GithubService), "")
	http.Handle("/", s)
	if err := http.ListenAndServe(":"+strconv.Itoa(*port), nil); err != nil {
		logger.Err(fmt.Sprintf("error(%v) on ListenAndServe", err.Error))
	}
}

func (h *GithubService) CheckRepo(r *http.Request, args *gorlim.CheckRepoArgs, reply *gorlim.CheckRepoResponse) error {
	mutex.Lock()
	defer mutex.Unlock()
	key := args.RepoPath
	if t, ok := converted[key]; ok {
		reply.LastConvertedEventTime = t
		reply.DoneRatio = 1
		return nil
	}
	if val, ok := inProgress[key]; ok {
		reply.LastConvertedEventTime = val.LastConvertedEventTime
		reply.DoneRatio = val.DoneRatio
		return nil
	}
	reply.LastConvertedEventTime = time.Unix(0, 0)
	reply.DoneRatio = 0
	logger.Info(fmt.Sprintf("starts creating new repo for %v", args))
	inProgress[key] = gorlim.CheckRepoResponse{
		LastConvertedEventTime: time.Unix(0, 0),
		DoneRatio:              1,
	}
	go func() {
		uri := toUri(key)
		repo := gorlim.NewGitRepo(args.RepoPath)
		syncManager.InitGitRepoFromIssues(uri, repo)
		syncManager.EstablishSync(uri, repo)
	}()
	return nil
}

func toUri(key string) string {
	prefix, project := path.Split(key)
	_, user := path.Split(filepath.Clean(prefix))
	result := user + "/" + project
	return result
}
