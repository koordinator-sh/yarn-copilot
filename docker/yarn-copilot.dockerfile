FROM golang:1.17 as builder
WORKDIR /go/src/github.com/koordinator-sh/goyarn

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN GOOS=linux GOARCH=amd64 go build -a -o koord-yarn-copilot cmd/yarn-copilot/main.go

FROM nvidia/cuda:11.2.2-base-ubuntu20.04
RUN apt-get add --update bash net-tools iproute2 logrotate less rsync util-linux lvm2
WORKDIR /
COPY --from=builder /go/src/github.com/koordinator-sh/goyarn .
ENTRYPOINT ["/koord-yarn-copilot"]
