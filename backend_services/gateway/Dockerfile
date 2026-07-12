# --- Dev stage (with Air hot-reload) ---
FROM golang:1.25-alpine AS dev
RUN apk add --no-cache git && go install github.com/air-verse/air@v1.61.7
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["air"]

# --- Build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

# --- Production stage ---
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /server /server
COPY migrations/ /migrations/
EXPOSE 8080
CMD ["/server"]
