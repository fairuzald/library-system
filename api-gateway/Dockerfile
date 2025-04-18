FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN echo "Contents of /app:" && ls -la /app && \
	echo "Contents of /app/api-gateway:" && ls -la /app/api-gateway && \
	echo "Contents of /app/api-gateway/cmd:" && ls -la /app/api-gateway/cmd

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/api-gateway-bin ./api-gateway/cmd/main.go

RUN ls -la /app && \
	echo "Binary exists at: " && \
	find /app -name "api-gateway-bin"

FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/api-gateway-bin /app/api-gateway

RUN ls -la /app && \
	echo "Binary check:" && \
	if [ -f /app/api-gateway ]; then echo "Binary exists"; else echo "Binary missing"; fi

EXPOSE 8000

CMD ["/bin/sh", "-c", "ls -la /app && exec /app/api-gateway"]
