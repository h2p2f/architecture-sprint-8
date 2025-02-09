FROM golang:1.23.1-alpine


WORKDIR /

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Создаем .env файл с значениями по умолчанию
RUN echo "KEYCLOAK_URL=http://keycloak:8080" > .env && \
    echo "KEYCLOAK_REALM=reports-realm" >> .env && \
    echo "PORT=8000" >> .env

RUN go build -o main ./cmd/main.go

EXPOSE 8000

CMD ["./main"]
