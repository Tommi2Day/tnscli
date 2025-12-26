 #!/bin/bash
 export MSYS_NO_PATHCONV=1
 CONTAINER=${CONTAINER:-"dblib-ldap"}
 URL=${URL:-"ldap://localhost"}
 LDAP_BASE="cn=config"
 LDAP_USER="cn=config"
 LDAP_PASSWORD=${LDAP_CONFIG_PASSWORD:-"config"}
 docker exec -ti "$CONTAINER" \
  ldapsearch  -H "$URL" \
    -D "$LDAP_USER" -w "$LDAP_PASSWORD" \
    -b "$LDAP_BASE" "*"