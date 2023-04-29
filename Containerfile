FROM golang:latest AS builder

# Magic line, notice in use that the lib name is different!
#RUN apt-get update && apt-get install -y gcc-aarch64-linux-gnu
# Add your app and do what you need to for dependencies
ADD . /go/src/github.com/snafuprinzip/webapp
WORKDIR /go/src/github.com/snafuprinzip/webapp
#RUN CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64 go build -o app .
RUN go build -tags netgo -ldflags '-extldflags "-static"' -o app .

FROM scratch

COPY --from=builder /go/src/github.com/snafuprinzip/webapp/app /
#ADD app /
ADD assets /assets
ADD templates /templates
ADD config/config.yaml.template /config/config.yaml

VOLUME /data

CMD ["/app"]
