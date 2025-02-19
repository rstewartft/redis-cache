FROM golang:1.23-alpine AS build

RUN apk add git

WORKDIR /app

COPY go.mod go.sum ./

# Download all the dependencies
RUN go mod download

COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

CMD ["./main"]