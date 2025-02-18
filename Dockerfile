FROM golang:1.23-alpine AS build

RUN apk add git

# Add Maintainer Info
LABEL maintainer="<>"

#RUN mkdir /app
#ADD . /app
WORKDIR /app

COPY go.mod go.sum ./

# Download all the dependencies
RUN go mod download

COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

CMD ["./main"]

# GO Repo base repo
#FROM alpine:latest
#
#RUN apk --no-cache add ca-certificates curl
#
#RUN mkdir /app
#
#WORKDIR /app
#
## Copy the Pre-built binary file from the previous stage
#COPY --from=build /app/main .
#
#COPY --from=build /usr/local/go/ /go
#ENV GOROOT=/go
#ENV PATH=$PATH:$GOROOT/bin
#
## Expose port 8000
#EXPOSE 8000
#
## Run Executable
#CMD ["./main"]