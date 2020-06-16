all: puff
	@echo Making puff done.
	@cp puff $(GOPATH)/tools
puff: puff.go
	@go build -o puff
clean:
	@go clean -i


