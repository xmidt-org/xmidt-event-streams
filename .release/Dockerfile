# SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
# SPDX-License-Identifier: Apache-2.0

# GoReleaser builds the binary and places it in the docker build context.
FROM docker.io/library/alpine:3.20

RUN apk add --no-cache --no-progress ca-certificates \
    && mkdir /etc/eventstream \
    && touch /etc/eventstream/eventstream.yaml \
    && chmod 666 /etc/eventstream/eventstream.yaml

COPY eventstream /eventstream

USER nobody

# these are overriden in docker compose
EXPOSE 8080
EXPOSE 8090
EXPOSE 8181

WORKDIR /

ENTRYPOINT ["./eventstream"]
