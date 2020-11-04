image: bin/linux bin/aerial
	docker build -t rueian/aerial:latest .
bin/linux:
	GOOS=linux go build -a -o bin/linux main.go
bin/aerial:
	go build -a -o bin/aerial main.go