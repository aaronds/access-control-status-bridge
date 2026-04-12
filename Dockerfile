FROM docker.io/golang:1.26.1 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build

FROM gcr.io/distroless/static-debian12
COPY --from=build /go/src/app/access-control-status-bridge /

CMD ["/access-control-status-bridge"]
