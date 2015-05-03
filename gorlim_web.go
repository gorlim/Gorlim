package main

import (
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/gorlim/Gorlim/gorlim"
	"github.com/gorlim/Gorlim/gorlim_github"
	"github.com/gorlim/Gorlim/storage"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const GH_SUFFIX = "/auth/github"
const PROJECTS_SUFFIX = "/projects"
const ADD_SUFFIX = "/add_project"
const SSH_FORMAT = "command=\"$GOPATH/bin/gorlim_ssh %v\",no-port-forwarding,no-X11-forwarding,no-pty ssh-rsa\n%v\n"

var db *storage.Storage

var syncManager *gorlim.SyncManager = nil
var conf configuration = configuration{}

type configuration struct {
	DbFile     string
	GitRoot    string
	ClientId   string
	SecretId   string
	KeyStorage string
}

func main() {
	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&conf)
	if err != nil {
		panic(err)
	}
	http.Handle("/", http.FileServer(http.Dir("./static/")))
	http.HandleFunc(GH_SUFFIX, githubAuthHandler)
	db, err = storage.Create(conf.DbFile)
	if err != nil {
		panic(err)
	}
	http.HandleFunc(ADD_SUFFIX, func(w http.ResponseWriter, r *http.Request) {
		text, err := ioutil.ReadAll(r.Body)
		if err != nil {
			prettyError(w, err.Error())
			return
		}
		values, err := url.ParseQuery(string(text))
		if err != nil {
			prettyError(w, err.Error())
			return
		}
		myType := values.Get("type")
		if myType != "github" {
			prettyError(w, "Please enter valid type")
			return
		}
		repo := values.Get("repo")
		if repo == "" {
			prettyError(w, "There is no such "+myType+" repository")
			return
		}
		split := strings.Split(repo, "/")
		if len(split) != 2 {
			prettyError(w, "Should be in user/repo format")
			return
		}
		if v, err := (*db).GetRepo(repo); err == nil && v != nil {
			prettyError(w, fmt.Sprintf("This GitHub:Issues is already extracted: %#v", repo))
			return
		}
		user := split[0]
		repoName := split[1]
		t := &github.UnauthenticatedRateLimitedTransport{
			ClientID:     conf.ClientId,
			ClientSecret: conf.SecretId,
		}
		gh := github.NewClient(t.Client())
		resp, _, err := gh.Repositories.Get(user, repoName)
		if err != nil || resp == nil {
			prettyError(w, fmt.Sprintf("No GitHub repository: %#v", repo))
			return
		}
		err = (*db).AddRepo(myType, repo, time.Now(), false)
		if err != nil {
			prettyError(w, err.Error())
			return
		}
		go createOurRepo(myType, user, repoName)
	})
	http.HandleFunc(PROJECTS_SUFFIX, func(w http.ResponseWriter, r *http.Request) {
		needle := ""
		if v := r.Form["needle"]; v != nil && len(v) > 0 {
			needle = v[0]
		}
		repos, err := (*db).GetRepos(needle)
		if err != nil {
			prettyError(w, err.Error())
			return
		}
		js, err := json.Marshal(repos)
		if err != nil {
			prettyError(w, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})
	// setup synchronization manager
	githubIssuesWeb := GithubWebIssuesInterface{
		clientId: conf.ClientId,
		secretId: conf.SecretId,
	}
	syncManager = gorlim.CreateSyncManager(&githubIssuesWeb)
	// init existing repos from database
	repos, err := (*db).GetAllRepos()
	if err != nil {
		panic(err) // TBD - show to user
	}
	if repos != nil {
		for _, repo := range repos {
			origin := *repo.Origin
			path := getRepoPath(origin)
			repo := gorlim.CreateFromExistingGitRepo(path)
			syncManager.EstablishSync(origin, repo)
		}
	}
	// go to listen and serve loop
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func githubAuthHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if code := query.Get("code"); code != "" {
		ch := make(chan error)
		go initUser(code, ch)
		err := <-ch
		fmt.Printf("err: %#v\n", err)
	}
	http.Redirect(w, r, "/repositories.html", http.StatusFound)
}

func initUser(code string, ch chan error) {
	defer close(ch)
	data := url.Values{}
	data.Set("client_id", conf.ClientId)
	data.Set("client_secret", conf.SecretId)
	data.Set("code", code)

	r, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		ch <- err
		return
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := http.PostForm("https://github.com/login/oauth/access_token", data)
	if err != nil {
		ch <- err
		return
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ch <- err
		return
	}

	values, err := url.ParseQuery(string(contents))
	if err != nil {
		ch <- err
		return
	}
	access_token := values.Get("access_token")
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: access_token},
	}
	client := github.NewClient(t.Client())
	user, _, err := client.Users.Get("")
	if err != nil {
		ch <- err
		return
	}
	login := *user.Login
	_, err = (*db).GetGithubAuth(login)
	f, err := os.OpenFile(conf.KeyStorage, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		ch <- err
		return
	}

	defer f.Close()

	options := &github.ListOptions{Page: 1, PerPage: 100}
	for {
		keys, resp, err := client.Users.ListKeys("", options)
		if err != nil {
			ch <- err
			return
		}
		for _, key := range keys {
			if _, err = fmt.Fprintf(f, SSH_FORMAT, login, (*key.Key)); err != nil {
				ch <- err
				return
			}
		}
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}
	(*db).SaveGithubAuth(login, access_token)
}

type GithubWebIssuesInterface struct {
	clientId string
	secretId string
}

func (gwi *GithubWebIssuesInterface) UpdateIssue(uri string, oldValue gorlim.Issue, newValue gorlim.Issue) error {
	owner, repo := gwi.uriToOwnerRepoPair(uri)
	access_token, err := (*db).GetGithubAuth(owner)
	if err != nil {
		panic(err)
	}
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: access_token},
	}
	return gorlim_github.UpdateIssue(owner, repo, t.Client(), time.Now(), oldValue, newValue)
}

func (gwi *GithubWebIssuesInterface) GetIssues(uri string, date *time.Time) []gorlim.Issue {
	t := &github.UnauthenticatedRateLimitedTransport{
		ClientID:     gwi.clientId,
		ClientSecret: gwi.secretId,
	}
	owner, repo := gwi.uriToOwnerRepoPair(uri)
	return gorlim_github.GetIssues(owner, repo, t.Client(), date)
}

func (gwi *GithubWebIssuesInterface) CreateIssuesUpdateChannel(uri string) <-chan gorlim.IssuesUpdate {
	fmt.Println("CreateIssuesUpdateChannel")
	ch := make(chan gorlim.IssuesUpdate)
	owner, repo := gwi.uriToOwnerRepoPair(uri)
	ticker := time.NewTicker(time.Minute)
	go func() {
		date := time.Now()
		for now := range ticker.C {
			fmt.Println("Tick")
			t := &github.UnauthenticatedRateLimitedTransport{
				ClientID:     gwi.clientId,
				ClientSecret: gwi.secretId,
			}
			issues := gorlim_github.GetIssues(owner, repo, t.Client(), &date)
			date = now
			ch <- gorlim.IssuesUpdate{Uri: uri, Issues: issues}
		}
	}()
	return ch
}

func (gwi *GithubWebIssuesInterface) uriToOwnerRepoPair(uri string) (string, string) {
	owner := strings.Split(uri, "/")[0]
	repo := strings.Split(uri, "/")[1]
	return owner, repo
}

func getRepoPath(repo string) string {
	return conf.GitRoot + "/" + repo + ".issues"
}

func createOurRepo(myType, user, repoName string) {
	key := user + "/" + repoName
	path := getRepoPath(key)
	fmt.Println(path)
	repo := gorlim.NewGitRepo(path)
	syncManager.InitGitRepoFromIssues(key, repo)
	syncManager.EstablishSync(key, repo)
	r, err := (*db).GetRepo(key)
	if err != nil {
		return
	}
	prev := *r
	(*db).AddRepo(*prev.Type, *prev.Origin, *prev.Last, true)
}

func prettyError(w http.ResponseWriter, text string) {
	http.Error(w, "<b>Ooops.</b> "+text, http.StatusInternalServerError)
}
