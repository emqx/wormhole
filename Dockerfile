FROM golang:1.15.1-alpine as builder

COPY . /go/wormhole

WORKDIR /go/wormhole

RUN apk add make git \
    && make 

FROM alpine:3.12

COPY --from=builder /go/wormhole/_build/wormhole-* /wormhole/
   
WORKDIR /wormhole

CMD ["./agent", "client"] 
