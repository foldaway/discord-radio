FROM golang:1.13-alpine AS build_base

RUN apk --no-cache add git gcc musl-dev
WORKDIR /go/src/github.com/bottleneckco/discord-radio
ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod download

FROM build_base AS server_builder
WORKDIR /go/src/github.com/bottleneckco/discord-radio
ENV GOPATH /go
RUN go get -u github.com/bwmarrin/dca/cmd/dca
COPY . .
RUN go build -a -o app .

FROM alpine:3.8
RUN apk --update --no-cache add ca-certificates ffmpeg curl python
RUN curl -L https://yt-dl.org/downloads/latest/youtube-dl -o /usr/local/bin/youtube-dl
RUN chmod a+rx /usr/local/bin/youtube-dl
COPY --from=server_builder /go/bin/dca /usr/local/bin/dca
WORKDIR /root/
COPY --from=server_builder /go/src/github.com/bottleneckco/discord-radio/app .
CMD "./app"
