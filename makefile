server:
	go build -o bin/anytls-server ./cmd/server


client:
	go build -o bin/anytls-client ./cmd/client


linux:
	GOOS=linux GOARCH=amd64 go build -o bin/anytls-server-linux ./cmd/server
	GOOS=linux GOARCH=amd64 go build -o bin/anytls-client-linux ./cmd/client
	GOOS=linux GOARCH=amd64 go build -o bin/anytls-redirect-linux ./cmd/redirect

windows:
	GOOS=windows GOARCH=amd64 go build -o bin/anytls-server-windows.exe ./cmd/server
	GOOS=windows GOARCH=amd64 go build -o bin/anytls-client-windows.exe ./cmd/client

macos:
	GOOS=darwin GOARCH=amd64 go build -o bin/anytls-server-macos ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o bin/anytls-client-macos ./cmd/client