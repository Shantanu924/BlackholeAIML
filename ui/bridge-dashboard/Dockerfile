FROM golang:1.23-alpine

WORKDIR /app
COPY main.go .
COPY static/ ./static/

RUN go build -o ui-server main.go

EXPOSE 8103

CMD ["./ui-server"]
