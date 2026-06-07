FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGET=api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${TARGET}

FROM alpine:3.22

RUN apk add --no-cache ca-certificates docker-cli git && addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=build /out/app /app/app

USER app
EXPOSE 8080

ENTRYPOINT ["/app/app"]
