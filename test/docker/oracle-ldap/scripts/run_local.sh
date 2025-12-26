#!/usr/bin/env bash
export MSYS_NO_PATHCONV=1
export CONTAINER=dblib-ldap
docker stop $CONTAINER
docker rm $CONTAINER
DIR=$(dirname $0)
cd ..
WD=$(realpath .)

docker run -d \
  --name $CONTAINER \
  --hostname ldap \
  -v $WD/certs:/certs \
  -v $WD/ldif:/ldif \
  -v $WD/etc/slapd.conf:/etc/openldap/slapd.conf \
  -p 1389:389 -p 1636:636 \
  cleanstart/openldap:2.6.10
scripts/modify_config.sh
scripts/modify_entries.sh
scripts/show_entries.sh