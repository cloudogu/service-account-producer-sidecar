FROM golang:1.26.4-alpine AS builder
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY internal internal

RUN CGO_ENABLED=0 GOOS=linux go build -o target/service-account-producer-sidecar ./cmd/service-account-producer-sidecar

FROM registry.cloudogu.com/official/base:3.23.3-5
LABEL maintainer="hello@cloudogu.com" \
    NAME="k8s/service-account-producer-sidecar" \
    VERSION="0.1.0"

ENV USER=hooksidecar \
    USER_ID=1000 \
    GROUP=hooksidecar \
    GROUP_ID=1000

RUN apk update && apk upgrade \
    && apk add --no-cache bash curl jq \
    && addgroup -S "${GROUP}" -g ${GROUP_ID} \
    && adduser -S -h "/home/${USER}" -G "${GROUP}" -u ${USER_ID} -s /bin/bash "${USER}"

COPY --from=builder /workspace/target/service-account-producer-sidecar /service-account-producer-sidecar

USER ${USER}
EXPOSE 8080
ENTRYPOINT ["/service-account-producer-sidecar"]
