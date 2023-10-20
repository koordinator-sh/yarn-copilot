FROM golang:1.19 as builder
WORKDIR /go/src/github.com/koordinator-sh/goyarn

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o koord-yarn-operator cmd/yarn-operator/main.go

FROM alpine:3.16
RUN apk add --update bash net-tools iproute2 logrotate less rsync util-linux lvm2
WORKDIR /
COPY --from=builder /go/src/github.com/koordinator-sh/goyarn .
ENTRYPOINT ["/koord-yarn-operator"]
