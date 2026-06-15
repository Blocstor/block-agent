FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bloc-agent ./cmd/agent

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bloc-agent /bloc-agent
EXPOSE 8080
ENTRYPOINT ["/bloc-agent"]
