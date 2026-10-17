# OSSAlot image (ADR-035): the UI build embedded in the Go binary, on a distroless base.
FROM node:20-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ossalot ./cmd/ossalot \
 && mkdir /out/data && OSSALOT_DB_PATH=/tmp/check.db /out/ossalot -check

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ossalot /ossalot
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV OSSALOT_ADDR=0.0.0.0:8080 OSSALOT_DB_PATH=/data/ossalot.db
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/ossalot"]
