FROM golang:1.24.1

WORKDIR /app

COPY ../../go.mod ../../go.sum ./
RUN go mod download

COPY cmd/confidentiality_system/*.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /confidentiality-system

EXPOSE 9090

CMD ["/confidentiality-system"]
