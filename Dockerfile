FROM golang:1.22.1-alpine AS builder

# Disables the use of CGo when building the go app; CGo is a feature in the Go that allows code to call C functions.
# By disabling CGo, we ensure that the Go binary does not depend on any C libraries, which provides a few benefits:
#   Static binary
#   Smaller Docker image
#   Improved security
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /build

# Adding a user and running the service as that user is a security best practice,
# which helps limit the potential damage in case the service is compromised.
# When the service runs as a non-root user, it has fewer privileges than the root user,
# which can mitigate the risk of unauthorized access and restrict the attacker's
# capabilities within the container.
RUN adduser -D stservice

RUN mkdir -p /var/gymstats && chown stservice:stservice /var/gymstats

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# -w: disable the generation of debugging information (DWARF) in the resulting binary
# -s: disable the generation of the symbol table in the resulting binary
RUN go build -o bin/service -ldflags="-w -s" cmd/service/main.go

FROM alpine

ARG OPEN_WEATHER_API_KEY_ARG=todo
ARG SERJ_TUBIN_COM_ADMIN_USERNAME_ARG=todo
ARG SERJ_TUBIN_COM_ADMIN_PASSWORD_HASH_ARG='$2a$14$tqM9zVV1bSeI3e4mBW/8DuunsnoBoQwKBXeKASl4AXpALge3/WsXi'
ARG SERJ_BROWSER_REQ_SECRET_ARG=todo
ARG SERJ_GYMSTATS_IOS_APP_SECRET_ARG=todo
ARG SERJ_REDIS_PASS_ARG=todo
ARG OTEL_SERVICE_NAME_ARG=serj-tubin-com-docker-dev
ARG HONEYCOMB_ENABLED_ARG="false"
ARG HONEYCOMB_API_KEY_ARG=""
ARG SENTRY_DSN_ARG=todo

ENV OPEN_WEATHER_API_KEY=${OPEN_WEATHER_API_KEY_ARG}
ENV SERJ_TUBIN_COM_ADMIN_USERNAME=${SERJ_TUBIN_COM_ADMIN_USERNAME_ARG}
ENV SERJ_TUBIN_COM_ADMIN_PASSWORD_HASH=${SERJ_TUBIN_COM_ADMIN_PASSWORD_HASH_ARG}
ENV SERJ_BROWSER_REQ_SECRET=${SERJ_BROWSER_REQ_SECRET_ARG}
ENV SERJ_GYMSTATS_IOS_APP_SECRET=${SERJ_GYMSTATS_IOS_APP_SECRET_ARG}
ENV SERJ_REDIS_PASS=${SERJ_REDIS_PASS_ARG}
ENV OTEL_SERVICE_NAME=${OTEL_SERVICE_NAME_ARG}
ENV HONEYCOMB_ENABLED=${HONEYCOMB_ENABLED_ARG}
ENV HONEYCOMB_API_KEY=${HONEYCOMB_API_KEY_ARG}
ENV SENTRY_DSN=${SENTRY_DSN_ARG}

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /build/bin/service /usr/bin/service
COPY --from=builder /build/bin/service /usr/bin/service
COPY --from=builder /build/config.toml config.toml
COPY --from=builder /build/assets assets
COPY --from=builder /var/gymstats /var/gymstats

RUN chown -R stservice:stservice /var/gymstats

USER stservice
CMD ["./usr/bin/service", "-env", "dockerdev"]
