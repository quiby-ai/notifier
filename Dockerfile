FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/app ./cmd/main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/app /app
COPY config.toml /
USER nonroot

EXPOSE 8888
ENTRYPOINT ["/app"]
