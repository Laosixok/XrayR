FROM golang:1.27.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -buildid=" -o /out/XrayR .

FROM alpine:3.22
RUN apk add --no-cache tzdata ca-certificates \
    && mkdir -p /etc/XrayR /usr/share/licenses/XrayR
WORKDIR /etc/XrayR
ENV TZ=Asia/Shanghai
COPY --from=builder /out/XrayR /usr/local/bin/XrayR
COPY --from=builder /app/LICENSE /app/NOTICE.md /app/THIRD_PARTY_NOTICES.md /usr/share/licenses/XrayR/
ENTRYPOINT ["XrayR", "--config", "/etc/XrayR/config.yml"]
