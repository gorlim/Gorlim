package gorlim

import "time"

type HookArgs struct {
	RepoPath string
	NewSha   string
	OldSha   string
}

type HookResponse struct {
	Message string
}

type CheckRepoResponse struct {
	LastConvertedEventTime time.Time
	DoneRatio              float32
}

type CheckRepoArgs struct {
	RepoPath string
}
