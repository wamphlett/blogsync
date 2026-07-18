FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cfg ./cfg
COPY cmd ./cmd
COPY pkg ./pkg
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/blogsync ./cmd/blogsync

FROM alpine:3.20
RUN apk add --no-cache git ca-certificates openssh-client \
    && addgroup -S blogsync && adduser -S -G blogsync blogsync
COPY --from=build /out/blogsync /usr/local/bin/blogsync
USER blogsync
ENTRYPOINT ["/usr/local/bin/blogsync"]
