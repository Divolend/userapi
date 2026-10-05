FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/api /app/api
RUN mkdir -p /app/logs
EXPOSE 8080
CMD ["/bin/sh", "-c", "/app/api 2>&1 | tee -a /app/logs/app.log"]
