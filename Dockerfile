FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

RUN echo "Building for $TARGETOS / $TARGETARCH / $TARGETVARIANT"

WORKDIR $GOPATH/src/github.com/viscerous/goplaxt/
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out/keystore

RUN CGO_ENABLED=0  \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOARM=$(echo $TARGETVARIANT | sed 's/v//') \
    go build -o /out/goplaxt-docker .

FROM scratch
LABEL maintainer="rviscerous@gmail.com"
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out .
COPY static ./static
VOLUME /app/keystore/
EXPOSE 8000
ENTRYPOINT ["/app/goplaxt-docker"]
