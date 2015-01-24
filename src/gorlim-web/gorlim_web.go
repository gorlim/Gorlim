package main

import (
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"storage"
	"strconv"
)

const GH_SUFFIX = "/auth/github"
const PROJECTS_SUFFIX = "/projects"
const ADD_SUFFIX = "/add_project"

func main() {
	http.Handle("/", http.FileServer(http.Dir("./static/")))
	http.HandleFunc(GH_SUFFIX, githubAuthHandler)
	db, err := storage.Create("./test.db")
	if err != nil {
		log.Fatal("Create: ", err)
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
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func githubAuthHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if code := query.Get("code"); code != "" {
		data := url.Values{}
		data.Set("client_id", "a726527a9c585dfe4550")
		data.Set("client_secret", "a2c0edff50fcda34cf214684f3bf70d6ff1cb05f")
		data.Set("code", code)

		r, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBufferString(data.Encode()))
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

		resp, _ := http.PostForm("https://github.com/login/oauth/access_token", data)
		defer resp.Body.Close()
		contents, _ := ioutil.ReadAll(resp.Body)

		values, err := url.ParseQuery(string(contents))
		t := &oauth.Transport{
			Token: &oauth.Token{AccessToken: values.Get("access_token")},
		}
		client := github.NewClient(t.Client())
		user, _, err := client.Users.Get("")
		fmt.Println(err)
		fmt.Printf("%#v %#v\n", values.Get("access_token"), user)
	}
	http.Redirect(w, r, "/repositories.html", http.StatusFound)
}

func prettyError(w http.ResponseWriter, text string) {
	http.Error(w, "<b>Ooops.</b> "+text, http.StatusInternalServerError)
}
