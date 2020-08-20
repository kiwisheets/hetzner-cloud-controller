FROM golang:alpine AS build

WORKDIR /build

RUN apk update && apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o main .

WORKDIR /dist

RUN cp /build/main .

FROM alpine:latest

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /dist/main /

CMD [ "/main" ]
