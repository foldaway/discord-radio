FROM golang:1.13-alpine AS server_builder 

RUN apk --no-cache add git gcc musl-dev
WORKDIR /go/src/github.com/bottleneckco/discord-radio
ENV GOPATH /go
RUN go get -u github.com/bwmarrin/dca/cmd/dca

FROM alpine:3.8
RUN apk --update --no-cache add ca-certificates ffmpeg curl python
RUN curl -L https://yt-dl.org/downloads/latest/youtube-dl -o /usr/local/bin/youtube-dl
RUN curl -L -o /usr/local/bin/discord-radio "https://github.com/bottleneckco/discord-radio/releases/latest/discord-radio-linux-x64"
RUN chmod a+rx /usr/local/bin/youtube-dl
RUN chmod +x /usr/local/bin/discord-radio
COPY --from=server_builder /go/bin/dca /usr/local/bin/dca
WORKDIR /root/
CMD "discord-radio"
