from golang:rc-alpine as builder

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY cmd cmd
COPY internal internal
RUN go build ./cmd/switter


FROM alpine
COPY --from=builder /app/switter /bin/switter
WORKDIR /app

CMD ["/bin/switter"]