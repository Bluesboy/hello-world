PROJECT_ID=dojo-gcp
DOCKER_REPO=bluesboy/hello-world
TAG := $(shell git describe HEAD --tags)
STORAGE=sql

.PHONY : build

build-dev :
				docker build \
				-f build/package/Dockerfile \
				-t ${DOCKER_REPO}:dev \
				--target dev .

dev : build-dev
				docker run -it --rm \
				-p 8080:8080 \
				-v $(shell pwd):/src \
				${DOCKER_REPO}:dev

dev-sql : build-dev
				docker run -it --rm \
				-p 8080:8080 \
				-v $(shell pwd):/src \
				-e STORAGE=sql \
				${DOCKER_REPO}:dev

build :
				docker build \
				-f build/package/Dockerfile \
				-t ${DOCKER_REPO}:${TAG} .

mkdir:
				mkdir -p /db

run-txt : build
				docker run -it --rm \
				-p 8080:8080 \
				-e STORAGE=file \
				-w /db \
				-v $(shell pwd)/db:/db \
				${DOCKER_REPO}:${TAG}

run-sql : build
				docker run -it --rm \
				-p 8080:8080 \
				-e STORAGE=sql \
				-w /db \
				-v $(shell pwd)/db:/db \
				${DOCKER_REPO}:${TAG}

push : build tag
				docker push ${DOCKER_REPO}:${TAG}
				docker push ${DOCKER_REPO}:latest

tag :
				docker tag ${DOCKER_REPO}:${TAG} ${DOCKER_REPO}:latest
