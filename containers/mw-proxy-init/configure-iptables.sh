#!/bin/bash

set -ex

iptables -t nat -A OUTPUT ! -d 127.0.0.1 -p tcp -j REDIRECT --to-port ${PROXY_PORT} -m owner --uid-owner ${APISERVER_UID}


