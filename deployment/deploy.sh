#! /bin/sh
ansible-playbook --ask-vault-pass -i hosts.yml -vvvv deploy.yml $@
