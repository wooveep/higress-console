FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags='-s -w' -o /out/aigateway-console main.go

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata kubectl && \
    addgroup -S app && adduser -S -G app app

COPY --from=builder /out/aigateway-console /app/aigateway-console
COPY resource /app/resource
COPY manifest/config /app/manifest/config

EXPOSE 8080
USER app

CMD ["/app/aigateway-console"]
