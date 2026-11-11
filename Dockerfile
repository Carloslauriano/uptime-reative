FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/uptime-reative .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/uptime-reative /uptime-reative
ENTRYPOINT ["/uptime-reative"]
