
build:
	GOOS=linux GOARCH=amd64 go build -o /workspaces/Merfolk/bin/merfolk /workspaces/Merfolk/cmd/Mermaidsrv/main.go

run:
	go run /workspaces/Merfolk/cmd/Mermaidsrv/main.go
