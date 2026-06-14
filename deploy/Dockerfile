FROM golang:1.26-alpine AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /atlas-server ./cmd/atlas-server


FROM node:22-alpine AS web-builder

RUN corepack enable && corepack prepare pnpm@11 --activate

WORKDIR /app/web

COPY web/package.json web/pnpm-lock.yaml web/.npmrc ./
RUN pnpm install --frozen-lockfile

COPY web/ .
RUN pnpm build


FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=go-builder /atlas-server .
COPY --from=web-builder /app/web/dist ./web/dist

EXPOSE 8080

CMD ["./atlas-server"]
