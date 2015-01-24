package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

const GH_PREFIX = "/auth/github"

func main() {
	http.Handle("/", http.FileServer(http.Dir("./static/")))
	http.HandleFunc(GH_PREFIX, githubAuthHandler)
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

		w.Write(contents)
	}
}
