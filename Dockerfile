# 1) React build
FROM node:22-alpine AS web
WORKDIR /app
COPY client/package.json client/package-lock.json ./
RUN npm ci
COPY client/ ./
RUN npm run build

# 2) Go build
FROM golang:1.25-alpine AS api
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# 3) Yakuniy image: bitta binary + frontend fayllari
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H app
ENV TZ=Asia/Tashkent \
    PORT=8080
WORKDIR /app
COPY --from=api /out/api /app/api
COPY --from=web /app/dist /app/web
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/health || exit 1
CMD ["/app/api"]
