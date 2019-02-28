FROM golang:1.11-alpine3.8

RUN apk --no-cache add git gcc musl-dev
WORKDIR /go/src/github.com/bottleneckco/radio-clerk
ENV GOPATH /go
RUN go get -u github.com/bwmarrin/dca/cmd/dca
COPY . .
RUN go build -a -o app .

FROM alpine:3.8
RUN apk --update --no-cache add ca-certificates ffmpeg curl python
RUN curl -L https://yt-dl.org/downloads/latest/youtube-dl -o /usr/local/bin/youtube-dl
RUN chmod a+rx /usr/local/bin/youtube-dl
COPY --from=0 /go/bin/dca /usr/local/bin/dca
WORKDIR /root/
COPY --from=0 /go/src/github.com/bottleneckco/radio-clerk/app .
CMD "./app"
