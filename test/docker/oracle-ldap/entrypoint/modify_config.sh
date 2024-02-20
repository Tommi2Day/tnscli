#!/bin/bash
. /opt/bitnami/scripts/libopenldap.sh

#start ldap to modify ACL
ldap_start_bg

ldapmodify -Y EXTERNAL -H ldapi:/// -D cn=config <<EOF
dn: olcDatabase={0}config,cn=config
changetype: modify
replace: olcAccess
olcAccess: to * by dn.base="gidNumber=0+uidNumber=1001,cn=peercred,cn=external,cn=auth" manage by dn.base="cn=admin,dc=example,dc=local" manage by * none
EOF

ldapmodify -Y EXTERNAL -H ldapi:/// -D cn=config <<EOF
dn: olcDatabase={2}mdb,cn=config
changetype: modify
replace: olcAccess
olcAccess: to attrs=userPassword,shadowLastChange,sshPublicKey by self write by dn.base="cn=admin,dc=example,dc=local" write by anonymous auth by * none
olcAccess: to * by self write by dn.base="gidNumber=0+uidNumber=1001,cn=peercred,cn=external,cn=auth" manage by dn.base="cn=admin,dc=example,dc=local" manage by * read
EOF

ldapmodify -Y EXTERNAL -H ldapi:/// -D cn=config <<EOF
dn: olcDatabase={-1}frontend,cn=config
changetype: modify
replace: olcSizeLimit
olcSizeLimit: 2000
EOF

# stop ldap again
ldap_stop