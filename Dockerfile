# Build Stage
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# SQLITE requires CGO_ENABLED=1 to compile successfully 
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Run Stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
# Copy compiled binary and frontend assets
COPY --from=builder /app/main .
COPY --from=builder /app/db ./db
# If your UI files are in a local directory, ensure they are copied here too:
# COPY --from=builder /app/static ./static 

EXPOSE 8080
CMD ["./main"]
