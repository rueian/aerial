image: bin/linux bin/aerial
	docker build -t rueian/aerial:latest .
.PHONY: bin/linux
bin/linux:
	GOOS=linux go build -a -o bin/linux main.go
.PHONY: bin/aerial
bin/aerial:
	go build -a -o bin/aerial main.go