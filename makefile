all: puff
	@echo Making puff done.
puff: puff.go
	@go build -o puff
clean:
	@go clean -i


