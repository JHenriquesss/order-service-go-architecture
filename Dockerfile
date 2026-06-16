FROM golang:1.23-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/api ./cmd/api
RUN go build -o /bin/worker ./cmd/worker

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/api /app/api
COPY --from=build /bin/worker /app/worker

EXPOSE 8080

CMD ["/app/api"]
