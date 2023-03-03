############################
# STEP 1 build executable
############################
FROM golang:1.15 AS builder

WORKDIR $GOPATH/src/github.com/zhongjie-cai/image-processor

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ./bin/image-processor

RUN cp ./bin/image-processor /go/bin/image-processor

############################
# STEP 2 gen runner image
############################
FROM alpine:latest AS runner

COPY --from=builder /go/bin/image-processor /go/bin/image-processor

WORKDIR /go/bin

EXPOSE 8080

ENTRYPOINT ./image-processor