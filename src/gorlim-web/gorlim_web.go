package main

import (
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"gorlim"
	"gorlim_github"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"storage"
	"strconv"
	"strings"
)

const GH_SUFFIX = "/auth/github"
const PROJECTS_SUFFIX = "/projects"
const ADD_SUFFIX = "/add_project"

var db *storage.Storage

var syncManager gorlim.SyncManager = *gorlim.Create()
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
	db, err := storage.Create(conf.DbFile)
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
		err = (*db).AddRepo(myType, repo, "lalala")
		if err != nil {
			prettyError(w, err.Error())
			return
		}
		go createOurRepo(myType, repo)
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
	listener := gorlim.GetPushSocketListener()
	defer listener.Free()
	syncManager.SubscribeToPushEvent(listener.GetSocketWriteEvent())
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
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: values.Get("access_token")},
	}
	client := github.NewClient(t.Client())
	user, _, err := client.Users.Get("")
	if err != nil {
		ch <- err
		return
	}
	login := *user.Login
	st, err := storage.Create(conf.DbFile)
	if err != nil {
		ch <- err
		return
	}
	_, err = (*st).GetGithubAuth(login)
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
			if _, err = f.WriteString((*key.Key) + "\n"); err != nil {
				ch <- err
				return
			}
		}
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}
	(*st).SaveGithubAuth(login, code)
}

func createOurRepo(myType, path string) {
	split := strings.Split(path, "/")
	user := split[0]
	repoName := split[1]
	t := &github.UnauthenticatedRateLimitedTransport{
		ClientID:     conf.ClientId,
		ClientSecret: conf.SecretId,
	}
	fmt.Println(user + " " + repoName)
	issues := gorlim_github.GetIssues(user, repoName, t.Client(), "")
	fmt.Println(issues)
	repo := gorlim.CreateRepo(conf.GitRoot + "/" + path)
	syncManager.AddRepository("???", repo)
	initRepo(&repo, issues)
}

func initRepo(repo *gorlim.IssueRepositoryInterface, issues []gorlim.Issue) {
	for _, issue := range issues {
		(*repo).Update("import from web: "+issue.Title, []gorlim.Issue{issue})
	}
}

func prettyError(w http.ResponseWriter, text string) {
	http.Error(w, "<b>Ooops.</b> "+text, http.StatusInternalServerError)
}
