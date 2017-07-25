#!/bin/bash

shopt -s extglob
home=$(pwd)

# If we're not under a "src" directory we're (probably) on the CI server.
# export GOPATH and cd to the right location
if [[ $home != *"src"* ]]; then
  export GOPATH=${home}

  dir=$(git config --get remote.origin.url)
  dir=${dir#http://}   # remove leading http://
  dir=${dir#https://}  # remove leading https://
  dir=${dir%.git}      # remove trailing .git
  dir="src/${dir}"     # add src/ prefix

  cd ${dir}
  if [ $? -ne 0 ]; then
    exit 255
  fi
fi

DIRS=$(go list ./... | grep -v '\/vendor\/')

printf "\nGo dirs:\n${DIRS}\n\n"

if [[ -z $DIRS ]]; then
  echo "No Go dirs found."
  exit 255
fi

go test -v -cpu 1,2,4 --race -timeout 10s $DIRS
if [ $? -ne 0 ]; then
  printf "\nTest failed.\n"
  exit 255
fi

printf "\nSuccess.\n"
