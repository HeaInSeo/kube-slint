FROM golang:1.25 AS builder
WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o hello-operator .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /src/hello-operator /hello-operator
EXPOSE 8080
ENTRYPOINT ["/hello-operator"]
