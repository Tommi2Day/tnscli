#!/usr/bin/env bash
export MSYS_NO_PATHCONV=1
CONTAINER=${CONTAINER:-"dblib-ldap"}
URL=${URL:-"ldap://localhost"}
LDAP_BASE=${LDAP_BASE:-'dc=oracle,dc=local'}
LDAP_USER=${LDAP_USER:-"cn=admin,$LDAP_BASE"}
LDAP_PASSWORD=${LDAP_PASSWORD:-"admin"}

docker exec -ti $CONTAINER \
   ldapsearch  -H "$URL" \
     -D "$LDAP_USER" -w "$LDAP_PASSWORD" \
     -b "$LDAP_BASE" "*"