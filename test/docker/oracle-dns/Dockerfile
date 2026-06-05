ARG BIND_VERSION=9.21
ARG VENDOR_IMAGE_PREFIX=""
ARG BIND_IMAGE=${VENDOR_IMAGE_PREFIX}docker.io/internetsystemsconsortium/bind9:${BIND_VERSION}
FROM ${BIND_IMAGE}

#Copy configuration files
COPY ["named.conf", "/etc/bind/"]
COPY ["rac.lan.zone", "/var/lib/bind/"]

# Expose Ports
EXPOSE 9055/tcp
