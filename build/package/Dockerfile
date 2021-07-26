FROM golang:1.16.6-buster as dev
ARG STORAGE
WORKDIR /src
COPY . /src
RUN go install github.com/mitranim/gow@latest && go get
ENTRYPOINT ["gow", "-v", "run", "."]

FROM dev as build
WORKDIR /src
COPY . /src
RUN go build -o build/hello-world &&\
  chmod +x build/hello-world

FROM debian:buster-slim as runtime
COPY --from=build /src/build/hello-world /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["hello-world"]
