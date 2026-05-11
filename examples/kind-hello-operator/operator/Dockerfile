FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -tags ignore -ldflags="-s -w" -o hello-operator ./operator/main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /src/hello-operator /hello-operator
EXPOSE 8080
ENTRYPOINT ["/hello-operator"]
