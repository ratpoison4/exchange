#!/bin/bash

FILENAME="version"

TAG="`git tag | sort --version-sort | tail -1`"
VER="`git log --oneline | head -1 `"

if [[ -z "$TAG" ]]; then
    TAG="N/A"
fi
TS="`TZ=UTC date +\"%F_%T\"`UTC"

echo "-X main.Version=${TAG} -X main.Revision=git:${VER:0:7} -X main.Date=${TS}"
