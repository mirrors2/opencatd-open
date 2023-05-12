GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)
# 获取源码最近一次 git commit log，包含 commit sha 值，以及 commit message
GitCommitLog=$(shell git log)
# 检查源码在最近一次 git commit 基础上，是否有本地修改，且未提交的文件
GitStatus=$(shell git status -s)
# 获取当前时间
BuildTime=$(shell date +'%Y.%m.%d %H:%M:%S')
# 获取Go的版本
BuildGoVersion=$(shell go version)

LDFlags=" \
-X 'main.Version=$(VERSION)' \
-X 'main.GitCommitLog=$(GitCommitLog)' \
-X 'main.BuildTime=$(BuildTime)' \
-X 'main.BuildGoVersion=$(BuildGoVersion)'"

.PHONY: web
# web
web:
	cd web && npm install && npm run build && mv dist ..
	
.PHONY: build
# build
build:
# mkdir -p bin/ && go build -ldflags $(LDFlags) -o ./bin/ ./...
	rm -rf  bin 
	mkdir -p bin/  &&  go build -ldflags "-s -w" -o ./bin/opencatd .
	upx -9 bin/opencatd

.PHONY:docker
# build docker images
docker:
	docker run --privileged --rm tonistiigi/binfmt --install all
	docker buildx create --use --name xbuilder --driver docker-container
	docker buildx inspect xbuilder --bootstrap
	docker buildx build --platform linux/amd64,linux/arm64 -t mirrors2/opencatd:latest -f docker/Dockerfile . --push

.PHONY: clean
# clean
clean:
	rm -rf bin/

.PHONY: all
# generate all
all:
	make build;


# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
