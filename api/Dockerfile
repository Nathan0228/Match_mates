FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/matchmates .

FROM alpine:3.22
RUN addgroup -S app && adduser -S app -G app && mkdir /data && chown app:app /data
USER app
WORKDIR /data
COPY --from=build /out/matchmates /usr/local/bin/matchmates
ENV APP_ENV=production DB_PATH=/data/matchmates.db UPLOAD_DIR=/data/uploads PORT=8080
EXPOSE 8080
CMD ["matchmates"]
