CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o release/gitdork_rev_darwin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o release/gitdork_rev_amd_linux
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o release/gitdork_rev.exe
