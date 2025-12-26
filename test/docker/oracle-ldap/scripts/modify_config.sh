#!/bin/bash
export MSYS_NO_PATHCONV=1
WD=$(dirname $0)
CONTAINER=${CONTAINER:-"dblib-ldap"}
URL="ldap://localhost"
LDAP_USER="cn=config"
LDAP_PASSWORD=${LDAP_CONFIG_PASSWORD:-"config"}
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
for f in $WD/../ldif/*.ldif.schema; do
  apply_ldif
done
for f in $WD/../ldif/*.ldif.config; do
  apply_ldif
done