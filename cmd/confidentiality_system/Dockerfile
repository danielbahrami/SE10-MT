FROM golang:1.24.1

WORKDIR /app

COPY ../../go.mod ../../go.sum ./
RUN go mod download

COPY ../../. ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /confidentiality-system ./cmd/confidentiality_system

EXPOSE 9090

CMD ["/confidentiality-system"]
