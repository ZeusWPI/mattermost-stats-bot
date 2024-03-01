FROM golang:1.22-alpine3.19

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download -x

COPY *.go ./

RUN go build -o mm-bot

FROM alpine:3.19

WORKDIR /

COPY --from=0 /build/mm-bot .

ENTRYPOINT [ "./mm-bot" ]