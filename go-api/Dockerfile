FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY ./go-api/go.mod ./go-api/go.sum ./
RUN go mod download

COPY ./go-api .
RUN CGO_ENABLED=0 GOOS=linux go build -o allshop-api .

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/allshop-api .
COPY --from=builder /build/docs ./docs

EXPOSE 8000

CMD ["./allshop-api"]
