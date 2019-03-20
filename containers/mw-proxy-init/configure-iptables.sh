#!/bin/bash

set -ex

if [[ -n "${INTERNAL_SERVICES_CIDR}" ]] ; then
	FILTER_INTERNAL_SERVICES="! -d ${INTERNAL_SERVICES_CIDR}"
fi

if [[ -n "${INTERNAL_PODS_CIDR}" ]] ; then
	FILTER_INTERNAL_PODS="! -d ${INTERNAL_PODS_CIDR}"
fi

iptables -t nat -A OUTPUT ! -d 127.0.0.1 \
	${FILTER_INTERNAL_SERVICES} ${FILTER_INTERNAL_PODS} \
	-p tcp -j REDIRECT --to-port ${PROXY_PORT} \
	-m owner --uid-owner ${APISERVER_UID}


