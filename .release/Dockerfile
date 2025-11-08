# SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
# SPDX-License-Identifier: Apache-2.0

FROM docker.io/library/golang:alpine3.18 as builder


WORKDIR /src

COPY . .

RUN apk add --no-cache --no-progress \
    ca-certificates \
    make \
    git \
    openssh \
    gcc \
    libc-dev 


RUN mkdir /etc/eventstream/ && touch /etc/eventstream/eventstream.yaml && chmod 666 /etc/eventstream/eventstream.yaml

FROM alpine:3.18

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/eventstream /eventstream

USER nobody

# these are overriden in docker compose
EXPOSE 8080
EXPOSE 8090
EXPOSE 8181

WORKDIR /

ENTRYPOINT ["./eventstream"]

