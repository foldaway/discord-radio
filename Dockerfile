FROM golang:1.14-buster AS server_builder

WORKDIR /go/src/github.com/bottleneckco/discord-radio
ENV GOPATH /go
RUN go get -u github.com/bwmarrin/dca/cmd/dca

FROM debian:buster
RUN apt-get -yqq update && \
    apt-get install -yq --no-install-recommends ca-certificates ffmpeg curl python && \
    apt-get autoremove -y && \
    apt-get clean -y
RUN curl -L https://yt-dl.org/downloads/latest/youtube-dl -o /usr/local/bin/youtube-dl
RUN curl -L -o /usr/local/bin/discord-radio "https://github.com/bottleneckco/discord-radio/releases/latest/download/discord-radio-linux-x64"
RUN chmod a+rx /usr/local/bin/youtube-dl
RUN chmod +x /usr/local/bin/discord-radio
COPY --from=server_builder /go/bin/dca /usr/local/bin/dca
WORKDIR /root/
ENTRYPOINT ["discord-radio"]