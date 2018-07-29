FROM golang:latest AS build

WORKDIR /go/src/github.com/fvdveen/mu2
ADD . .

WORKDIR /go/src/github.com/fvdveen/mu2
RUN CGO_ENABLED=0 go build -o mu2 main.go

FROM alpine:latest AS RUN

WORKDIR /app/
COPY --from=build /go/src/github.com/fvdveen/mu2/mu2 mu2
RUN apk add --no-cache ca-certificates

CMD ["./mu2"]