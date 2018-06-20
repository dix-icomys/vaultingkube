FROM golang:1.9-alpine AS BUILD

WORKDIR /go/src/github.com/dix-icomys/vaultingkube

COPY . /go/src/github.com/dix-icomys/vaultingkube

RUN apk update && \
    apk add curl ca-certificates git && \
    update-ca-certificates && \
    go get github.com/Masterminds/glide && \
    glide i && \
    go build -ldflags="-s -w" .

FROM alpine:3.7

RUN apk --no-cache add ca-certificates && \
    update-ca-certificates

COPY --from=BUILD /go/src/github.com/dix-icomys/vaultingkube/vaultingkube /usr/bin

CMD ["vaultingkube"]
