# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/usermgmt ./cmd/usermgmt

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/usermgmt /app/usermgmt
EXPOSE 10100 10101 10102
USER nobody
ENTRYPOINT ["/app/usermgmt"]
