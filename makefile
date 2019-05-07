all: vascgen
	@echo Making vascgen done.
	@cp vascgen $(GOPATH)/bin
vascgen: vascgen.go
	@go build -o vascgen
clean:
	@go clean -i


