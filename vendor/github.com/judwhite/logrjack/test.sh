FILES=$(ls *.go)

echo "Checking gofmt..."
fmtRes=$(gofmt -l -s -d $FILES)
if [ -n "${fmtRes}" ]; then
  echo "gofmt checking failed: ${fmtRes}"
  exit 255
fi

echo "Checking errcheck..."
# buffer.WriteString always returns nil error; panics if buffer too large
errRes=$(errcheck -blank -ignore 'Write[String|Rune|Byte],os:Close')
# TODO: add -asserts flag (maybe)
if [ $? -ne 0 ]; then
  echo "errcheck checking failed: ${errRes}!"
  exit 255
fi
if [ -n "${errRes}" ]; then
  echo "errcheck checking failed: ${errRes}"
  exit 255
fi

echo "Checking govet..."
go vet $FILES
if [ $? -ne 0 ]; then
  exit 255
fi

echo "Checking govet -shadow..."
for path in $FILES; do
  go tool vet -shadow ${path}
  if [ $? -ne 0 ]; then
    exit 255
  fi
done

echo "Checking golint..."
lintError=0
for path in $FILES; do
  lintRes=$(golint ${path})
  if [ -n "${lintRes}" ]; then
    echo "golint checking ${path} failed: ${lintRes}"
    lintError=1
  fi
done

if [ ${lintError} -ne 0 ]; then
  exit 255
fi

echo "Running tests..."
if [ -f cover.out ]; then
  rm cover.out
fi

go test -timeout 3m --race -cpu 1
if [ $? -ne 0 ]; then
  exit 255
fi

go test -timeout 3m --race -cpu 2
if [ $? -ne 0 ]; then
  exit 255
fi

go test -timeout 3m --race -cpu 4
if [ $? -ne 0 ]; then
  exit 255
fi

go test -timeout 3m -coverprofile cover.out
if [ $? -ne 0 ]; then
  exit 255
fi

echo "Success"