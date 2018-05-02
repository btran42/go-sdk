PREFIX			?= $(shell pwd)
PKGS 			:= $(shell go list ./... | grep -v /vendor/)
SHASUMCMD 		:= $(shell command -v sha1sum || command -v shasum; 2> /dev/null)
TARCMD 			:= $(shell command -v tar || command -v tar; 2> /dev/null)
GIT_REF 		:= $(shell git log --pretty=format:'%h' -n 1)
CURRENT_USER 	:= $(shell whoami)
VERSION 		:= $(shell cat ./.version)

# this is to allow local go-sdk/db tests to pass
DB_SSLMODE		?= disable

# coverage stuff
CIRCLE_ARTIFACTS 	?= "."
COVERAGE_OUT 		:= "$(CIRCLE_ARTIFACTS)/coverage.html"

export GIT_REF
export VERSION
export DB_SSLMODE

all: format vet profanity test

ci: vet profanity cover 

new-install:
	@go get -u github.com/lib/pq
	@go get -u golang.org/x/oauth2
	@go get -u golang.org/x/oauth2/google
	@go get -u golang.org/x/lint/golint

format:
	@echo "$(VERSION)/$(GIT_REF) >> formatting code"
	@go fmt $(PKGS)

vet:
	@echo "$(VERSION)/$(GIT_REF) >> vetting code"
	@go vet $(PKGS)

lint:
	@echo "$(VERSION)/$(GIT_REF) >> linting code"
	@golint $(PKGS)

.PHONY: profanity
profanity:
	@echo "$(VERSION)/$(GIT_REF) >> profanity"
	@go run _bin/profanity/main.go -rules PROFANITY --exclude="_bin/*"

test:
	@echo "$(VERSION)/$(GIT_REF) >> tests"
	@go test -race -timeout 15s $(PKGS)

test-verbose:
	@echo "$(VERSION)/$(GIT_REF) >> tests"
	@go test -v $(PKGS)

cover:
	@echo "$(VERSION)/$(GIT_REF) >> coverage"
	@go run _bin/coverage/main.go

cover-enforce:
	@echo "$(VERSION)/$(GIT_REF) >> coverage"
	@go run _bin/coverage/main.go -enforce
	
cover-update:
	@echo "$(VERSION)/$(GIT_REF) >> coverage"
	@go run _bin/coverage/main.go -update

increment-patch:
	@echo "Current Version $(VERSION)"
	@go run _bin/increment_version/main.go patch ./.version > ./NEW_VERSION
	@mv ./NEW_VERSION ./.version
	@cat ./.version

increment-minor:
	@echo "Current Version $(VERSION)"
	@go run _bin/increment_version/main.go minor ./.version > ./NEW_VERSION
	@mv ./NEW_VERSION ./.version
	@cat ./.version

increment-major:
	@echo "Current Version $(VERSION)"
	@go run _bin/increment_version/main.go major ./.version > ./NEW_VERSION
	@mv ./NEW_VERSION ./.version
	@cat ./.version

tag-version:
	git tag -f $(VERSION)

push-tags:
	git push -f --tags
