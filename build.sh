CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o webdav_Mac_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o webdav_Mac_arm64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o webdav_Win_amd64.exe