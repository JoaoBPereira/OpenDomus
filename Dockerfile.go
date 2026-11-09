# Same image layout as ./Dockerfile, with the Go backend instead of Node.
# docker build -f Dockerfile.go -t opendomus-app:latest .

# -----------------------------------------------------------------------------
# Stage 1: Build the Go backend
# -----------------------------------------------------------------------------
FROM golang:1.27-alpine AS build
WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/od ./cmd/od

# -----------------------------------------------------------------------------
# Stage 2: Final Production Runtime
# -----------------------------------------------------------------------------
FROM alpine:3.22

RUN apk add --no-cache nginx ca-certificates

WORKDIR /usr/src/app

COPY --from=build /out/od ./od

# Copy application builds
COPY frontend/dist /usr/share/nginx/html/frontend
COPY backoffice/dist/spa /usr/share/nginx/html/bo

# Configure Nginx
COPY nginx.conf /etc/nginx/http.d/default.conf

EXPOSE 80 4000

CMD ["sh", "-c", "./od & exec nginx -g 'daemon off;'"]
