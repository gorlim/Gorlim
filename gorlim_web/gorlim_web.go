package main

import (
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"flag"
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
const SSH_FORMAT = "command=\"$GOPATH/bin/gorlim_ssh %v\",no-port-forwarding,no-X11-forwarding,no-pty ssh-rsa\n%v\n"

var db *storage.Storage

//var syncManager *gorlim.SyncManager = nil
var dbFile = flag.String("db", "gorlim.db", "SQLite file with keys")
var ghClient = flag.String("github-client", "", "GitHub Client Id for application")
var ghSecret = flag.String("github-secret", "", "GitHub Secret Id for application")
var staticDir = flag.String("static-dir", "", "Directory where all static files are")
var authorizedKeys = flag.String("authorized-keys", "~/.ssh/authorized_keys", "~/.ssh/authorized_keys to store ssh keys")

func main() {
	flag.Parse()
	http.Handle("/", http.FileServer(http.Dir(*staticDir)))
	http.HandleFunc(GH_SUFFIX, githubAuthHandler)
	var err error
	// go to listen and serve loop
	if err = http.ListenAndServe(":80", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	db, err = storage.Create(*dbFile)
	if err != nil {
		panic(err)
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
	data.Set("client_id", *ghClient)
	data.Set("client_secret", *ghSecret)
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
	f, err := os.OpenFile(*authorizedKeys, os.O_APPEND|os.O_WRONLY, 0600)
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

func (gwi *GithubWebIssuesInterface) CreateIssue(uri string, issue gorlim.Issue) (int, error) {
	owner, repo := gwi.uriToOwnerRepoPair(uri)
	access_token, err := (*db).GetGithubAuth(owner)
	if err != nil {
		panic(err)
	}
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: access_token},
	}
	return gorlim_github.CreateIssue(owner, repo, t.Client(), issue)
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
	return gorlim_github.UpdateIssue(owner, repo, t.Client(), oldValue, newValue)
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

/*
func getRepoPath(repo string) string {
	return  + "/" + repo + ".issues"
}

func createOurRepo(myType, user, repoName string) {
	key := user + "/" + repoName
	path := getRepoPath(key)
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
*/
func prettyError(w http.ResponseWriter, text string) {
	http.Error(w, "<b>Ooops.</b> "+text, http.StatusInternalServerError)
}
