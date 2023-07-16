ARG BIND_VERSION=9.18
ARG VENDOR_IMAGE_PREFIX=""
ARG BIND_IMAGE=${VENDOR_IMAGE_PREFIX}docker.io/internetsystemsconsortium/bind9:${BIND_VERSION}
FROM ${BIND_IMAGE}

#Install required tools and dependencies
RUN apt update && apt install -y \
        bind9-doc \
        dnsutils \
        net-tools

#Copy configuration files
COPY named.conf.options /etc/bind/
COPY named.conf.local /etc/bind/
COPY rac.lan.zone /etc/bind/zones/

# Expose Ports
EXPOSE 53/tcp
EXPOSE 53/udp
EXPOSE 953/tcp

# Start the Name Service
CMD ["/usr/sbin/named", "-g", "-c", "/etc/bind/named.conf", "-u", "bind"]