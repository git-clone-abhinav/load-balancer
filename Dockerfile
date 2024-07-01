FROM golang:1.22 as builder

WORKDIR /src
COPY . .
# build for linux
RUN env GOOS=linux GOARCH=amd64 go build -o /bin/loadbalancer_linux main.go 



FROM alpine as runner

# use multi-stage build to reduce image size, and copy binary from builder to runner(highly optimised)
COPY --from=builder /bin/loadbalancer_linux /bin/loadbalancer_linux
RUN apk --no-cache add ca-certificates

EXPOSE ${PORT}
CMD ["/bin/loadbalancer_linux"]