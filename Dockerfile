FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

COPY . .

RUN GOPATH= GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /main .

FROM scratch

COPY --from=builder main main

ENTRYPOINT ["/main"]
