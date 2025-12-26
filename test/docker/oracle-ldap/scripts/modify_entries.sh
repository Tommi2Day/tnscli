#!/bin/bash
export MSYS_NO_PATHCONV=1
WD=$(dirname $0)
CONTAINER=${CONTAINER:-"dblib-ldap"}
URL="ldap://localhost"
LDAP_BASE="dc=oracle,dc=local"
LDAP_USER=${LDAP_ADMIN_USER:-"cn=admin,$LDAP_BASE"}
LDAP_PASSWORD=${LDAP_ADMIN_PASSWORD:-"admin"}
function apply_ldif() {
    echo "apply $f"
    docker cp $f $CONTAINER:/etc/openldap/
    F=$(basename $f)
    docker exec -ti $CONTAINER ldapmodify -x \
        -H $URL \
        -D "$LDAP_USER" \
        -w "$LDAP_PASSWORD" \
        -f /etc/openldap/$F
 }
 for f in $WD/../ldif/*.ldif; do
   apply_ldif
 done