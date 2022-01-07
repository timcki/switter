from golang:rc-alpine as builder

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY cmd cmd
COPY pkg pkg
RUN go build ./cmd/conduit


FROM alpine
COPY --from=builder /app/conduit /bin/conduit
WORKDIR /app

CMD ["/bin/conduit"]