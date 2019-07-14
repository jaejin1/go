# Run command below to build binary.
#   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -s' -o main main.go

#ROM scratch
FROM golang:1.11
WORKDIR /usr/src/app
COPY ./gorm ./gorm
CMD ["./gorm"]