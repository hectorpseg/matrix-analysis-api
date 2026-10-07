# Multi-stage build that packages the Go orchestrator API and the Node stats API
# into a single container. The Go API listens on $PORT; the Node API listens on
# localhost:3001. The Go service talks to Node via http://127.0.0.1:3001.

# --- Build the Go binary ---
FROM golang:1.27.1-alpine AS go-build
WORKDIR /src
COPY go-api/go.mod go-api/go.sum ./
RUN go mod download
COPY go-api/ .
RUN CGO_ENABLED=0 go build -o /bin/app .

# --- Install Node production dependencies ---
FROM node:22-alpine AS node-build
WORKDIR /app
COPY node-api/package*.json ./
RUN npm ci --omit=dev

# --- Final runtime image ---
FROM node:22-alpine
RUN apk add --no-cache wget

WORKDIR /app

COPY --from=go-build /bin/app /bin/app
COPY --from=node-build /app/node_modules ./node_modules
COPY node-api/src ./src
COPY start.sh /start.sh
RUN chmod +x /start.sh

ENV PORT=8080
ENV NODE_API_URL=http://127.0.0.1:3001

EXPOSE 8080

USER node

CMD ["/start.sh"]
