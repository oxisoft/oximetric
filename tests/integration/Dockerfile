FROM golang:1.25-alpine

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ENTRYPOINT ["go", "test", "-v", "-count=1", "./tests/integration/"]
