FROM ubuntu:xenial
RUN apt-get update && apt-get upgrade -y && apt-get install -y \
    iproute2 \
    iptables \
 && rm -rf /var/lib/apt/lists/*

ADD configure-iptables.sh /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/configure-iptables.sh"]
