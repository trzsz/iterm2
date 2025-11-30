- update

```sh
wget https://raw.githubusercontent.com/trzsz/iterm2/refs/heads/main/api/api.proto
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go generate
```
