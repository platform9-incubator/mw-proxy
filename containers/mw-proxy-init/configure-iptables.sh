#!/bin/bash

set -ex

if [[ -n "${INTERNAL_SERVICES_CIDR}" ]] ; then
	iptables -t nat -A OUTPUT -d "${INTERNAL_SERVICES_CIDR}" -j ACCEPT
fi

iptables -t nat -A OUTPUT ! -d 127.0.0.1 \
	-p tcp -j REDIRECT --to-port ${PROXY_PORT} \
	-m owner --uid-owner ${APISERVER_UID}


