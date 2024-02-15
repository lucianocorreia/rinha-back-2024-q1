FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.22 as build

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src/app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /go/bin/app cmd/api/main.go

FROM --platform=${TARGETPLATFORM:-linux/amd64} gcr.io/distroless/static-debian11
COPY --from=build /go/bin/app /
CMD ["/app"] 
