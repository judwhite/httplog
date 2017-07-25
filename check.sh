#!/bin/bash

# go get -u github.com/kisielk/errcheck
# go get -u github.com/golang/lint/golint
# go get -u honnef.co/go/tools/cmd/gosimple
# go get -u honnef.co/go/tools/cmd/unused
# go get -u github.com/mdempsky/unconvert
# go get -u github.com/client9/misspell/cmd/misspell
# go get -u github.com/gordonklaus/ineffassign
# go get -u honnef.co/go/tools/cmd/staticcheck
# go get -u github.com/fzipp/gocyclo

errcho() {
  printf '%s\n' "$1" 1>&2
}

need_cmd() {
  if ! command -v "$1" > /dev/null 2>&1; then
    errcho "need '$1' (command not found)"
    exit 255
  fi
}

need_cmd shopt
need_cmd uname
need_cmd pwd
need_cmd git

if ! shopt -s extglob; then
  errcho "shopt failed"
  exit 255
fi

home=$(pwd)

# If we're not under a "src" directory we're (probably) on the CI server.
# export GOPATH and cd to the right location
if [[ ${home} != *"src"* ]]; then
  export GOPATH=${home}

  dir=$(git config --get remote.origin.url)
  dir=${dir#http://}   # remove leading http://
  dir=${dir#https://}  # remove leading https://
  dir=${dir%.git}      # remove trailing .git
  dir="src/${dir}"     # add src/ prefix

  if ! cd "${dir}"; then
    errcho "error cd'ing to ${dir}"
    exit 255
  fi
fi

# convert path to lowercase
# prevent windows/system32/find.exe from being the 'find' we use
uname=$(uname)
if [[ ${uname} == "MSYS_NT"* ]] || [[ ${uname} == "MINGW"* ]] || [[ ${uname} == "CYGWIN_NT"* ]]; then
  echo "Running on MinGW/Cygwin - ${uname}."
  PATH=${PATH,,}                         # convert path to all lowercase
  PATH=${PATH/\/c\/windows\/system32:/}  # remove /c/windows/system32:
else
  echo "Not running on MinGW/Cygwin - ${uname}."
fi

need_cmd go
need_cmd gofmt
need_cmd date
need_cmd find
need_cmd diff

need_cmd errcheck
need_cmd golint
need_cmd gosimple
need_cmd unused
need_cmd unconvert
need_cmd misspell
need_cmd ineffassign
need_cmd staticcheck
need_cmd gocyclo

mapfile -t FILES < <(find . -type f -iname "*.go"|grep -v '\/vendor\/')
mapfile -t DIRS < <(go list ./... | grep -v '\/vendor\/')

printf '\nCurrent directory:\n%s\n' "$(pwd)"

printf '\nGo files:\n'
printf '%s\n' "${FILES[@]}"

printf '\nGo dirs:\n'
printf '%s\n' "${DIRS[@]}"

if [ ${#FILES[@]} -eq 0 ]; then
  errcho "No Go files found."
  exit 255
fi

if [ ${#DIRS[@]} -eq 0 ]; then
  errcho "No Go dirs found."
  exit 255
fi

printf '\n'
echo "[$(date +%T)] Running static analysis..."

hasErr=0

echo "[$(date +%T)] - Checking gofmt..."
res=$(gofmt -l -s -d "${FILES[@]}")
if [ -n "${res}" ]; then
  errcho "gofmt checking failed: ${res}"
  hasErr=1
fi

echo "[$(date +%T)] - Checking errcheck..."

echo "
(*os.File).Close
(io.Closer).Close
(*text/tabwriter.Writer).Write
(*text/tabwriter.Writer).Flush
" > errcheck_excludes.txt

res=$(errcheck -blank -asserts -exclude errcheck_excludes.txt .)
if [ -n "${res}" ]; then
  errcho "errcheck checking failed:"
  errcho "${res}"
  hasErr=1
fi

rm errcheck_excludes.txt

echo "[$(date +%T)] - Checking govet..."

res=$(go tool vet -all -shadow "${FILES[@]}")
if [ -n "${res}" ]; then
  errcho "govet checking failed:"
  errcho "${res}"
  hasErr=1
fi

echo "[$(date +%T)] - Checking golint..."
for file in "${FILES[@]}"; do
  if [[ "$file" != *bindata.go ]]; then
    res=$(golint "${file}")
    if [ -n "${res}" ]; then
      errcho "golint checking ${file} failed:"
      errcho " ${res}"
      hasErr=1
    fi
  fi
done

echo "[$(date +%T)] - Checking gosimple..."
res=$(gosimple "${DIRS[@]}")
if [ -n "${res}" ]; then
  errcho "gosimple checking failed:"
  errcho "${res}"
  hasErr=1
fi

echo "[$(date +%T)] - Checking unused..."
res=$(unused "${DIRS[@]}")
if [ -n "${res}" ]; then
  errcho "unused checking failed:"
  errcho "${res}"
  hasErr=1
fi

echo "[$(date +%T)] - Checking unconvert..."
res=$(unconvert "${DIRS[@]}")
if [ -n "${res}" ]; then
  errcho "unconvert checking failed:"
  errcho "${res}"
  hasErr=1
fi

echo "[$(date +%T)] - Checking misspell..."
res=$(misspell "${FILES[@]}")
if [ -n "${res}" ]; then
  errcho "misspell checking failed:"
  errcho "${res}"
  hasErr=1
fi

echo "[$(date +%T)] - Checking ineffassign..."
for file in "${FILES[@]}"; do
  res=$(ineffassign "${file}")
  if [ -n "${res}" ]; then
    errcho "ineffassign checking failed:"
    errcho "${res}"
    hasErr=1
  fi
done

echo "[$(date +%T)] - Checking staticcheck..."
res=$(staticcheck "${DIRS[@]}")
if [ -n "${res}" ]; then
  errcho "staticcheck checking failed:"
  errcho "${res}"
  hasErr=1
fi

echo "[$(date +%T)] - Checking gocyclo..."
res=$(gocyclo -over 15 "${FILES[@]}")
if [ -n "${res}" ]; then
  errcho "gocyclo warning:"
  errcho "${res}"
fi

if [ ${hasErr} -ne 0 ]; then
  errcho ""
  errcho "[$(date +%T)] Lint errors."
  exit 1
fi

printf '\n[%s] Success.\n' "$(date +%T)"
