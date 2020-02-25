#!/usr/bin/env bash

DOCUMENT_ROOT=$1

cd ${DOCUMENT_ROOT}

make build && ./build/mac_grabber migrate -c config.json
