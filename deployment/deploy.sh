 #!/bin/sh
set -e

PROJECT=github.com/gorlim/Gorlim
go get github.com/tools/godep
godep restore
(cd $GOPATH/src/github.com/libgit2/git2go/ && git submodule update --init)
make -C $GOPATH/src/github.com/libgit2/git2go install
ALL="$PROJECT/gorlim_ssh $PROJECT/gorlim_hooks $PROJECT/gorlim_web $PROJECT/gorlim_github"
go clean $ALL 
go install $ALL

ansible-playbook --ask-vault-pass -i hosts.yml -vvvv deploy.yml $@
