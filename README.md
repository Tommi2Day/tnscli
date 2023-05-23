# tnscli

Small Oracle TNS Service and Connect Test Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/tommi2day/tnscli)](https://goreportcard.com/report/github.com/tommi2day/tnscli)
![CI](https://github.com/tommi2day/tnscli/actions/workflows/main.yml/badge.svg)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tommi2day/tnscli)


## Features
- connect to a given service using real connect method. 
- Uses given credentials or default to raise an ORA-1017 error
- search or list tns entries
- load and write tnsnames.ora to ldap

## Setup db test user
**CAUTION**: Don't use anonymous checks for monitoring. Some security analysis systems are qualifying this as "Brute-Force-Attack" if the check are started too often. 
set $TNSCLI_USER and TNSCLI_PASSWORD env or use --user and --password flag in check command instead. this users needs only a connect privilege
### setup a common user within CDB$ROOT
```sql
alter session set container=cdb$root;
create user c##tcheck identified by "<MyCheckPassword>"
    default tablespace users temporary tablespace temp
    account unlock container=all;
grant connect to c##tcheck container=all;
alter user c##tcheck default role all container=all;
```
## Usage
```bash
tnscli – Small Oracle TNS Service and Connect Test Tool

Usage:                         
  tnscli [command]             
                               
Available Commands:            
  check       Check TNS Entries
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  ldap        LDAP TNS Entries
  list        list TNS Entries
  version     version print version string

Flags:
  -c, --config string      config file
      --debug              verbose debug output
  -f, --filename string    path to alternate tnsnames.ora
  -h, --help               help for tnscli
      --info               reduced info output
  -A, --tns_admin string   TNS_ADMIN directory (default "$TNS_ADMIN")

Use "tnscli [command] --help" for more information about a command.

tnscli check [flags]

Flags:
  -a, --all               check all entries
  -H, --dbhost            print current host:cdb:pdb
  -h, --help              help for check
  -p, --password string   Password for real connect or set TNSCLI_PASSWORD
  -s, --service string    service name to check
  -t, --timeout int       timeout in sec (default 15)
  -u, --user string       User for real connect or set TNSCLI_USER

tnscli list [flags]

Flags:
  -C, --complete        print complete entry
  -h, --help            help for list
  -s, --search string   service name to check

tnscli ldap [command]

Available Commands:
  read        prints ldap tns entries to stdout
  write       update ldap tns entries

Flags:
  -h, --help                       help for ldap
  -b, --ldap.base string            Base DN to search from
  -D, --ldap.binddn string         DN of user for LDAP bind, empty for anonymous access
  -w, --ldap.bindpassword string   password for LDAP Bind
  -H, --ldap.host string           Hostname of Ldap Server
  -I, --ldap.insecure              do not verify TLS
  -o, --ldap.oraclectx string       Base DN of Oracle Context
  -p, --ldap.port int              ldapport to connect, 0 means TLS flag will decide
      --ldap.timeout int           ldap timeout in sec (default 20)
  -T, --ldap.tls                   use secure ldap (ldaps)

```
### Return Codes
- 0 success
- 1 failed

## Examples

```bash
>tnscli -version

# list only service names for named tnsnames.ora
>tnscli list -f test/testdata/connect.ora
XE.LOCAL


# list entries with full description 
>tnscli list --complete -f test/testdata/connect.ora 
XE.LOCAL=  (DESCRIPTION=  (ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=21521)))  (CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=XEPDB1)))


# list nonexisting service
>tnscli list --search  mydb
Error: no alias with 'mydb' found

# check for unavailable service with tnsnames.ora in $TNS_ADMIN
>tnscli check xe.local
Error: service XE.LOCAL  NOT reached:dial tcp 127.0.0.1:1521: 

# check connect to service xe with dummy credentials, expect ORA-01017  
>tnscli check xe -f test/testdata/connect.ora --info
[Thu, 13 Apr 2023 21:25:27 CEST]  INFO use entry 
XE.LOCAL=(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=21521)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=XEPDB1)))


[Thu, 13 Apr 2023 21:25:28 CEST]  WARN Connect OK, but Login error, maybe expected
[Thu, 13 Apr 2023 21:25:28 CEST]  INFO service xe connected  in 1.06s

OK, service XE.LOCAL reachable


# use TNSCLI_USER/TNSCLI_PASSWORD variables for real login checks
>export TNSCLI_PASSWORD=supersecret
>tnscli check xe -f test/testdata/connect.ora --user system --info
[Thu, 13 Apr 2023 21:23:37 CEST]  INFO use entry 
XE.LOCAL=(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=21521)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=XEPDB1)))


[Thu, 13 Apr 2023 21:23:37 CEST]  INFO service xe connected using user 'system' in 69ms

OK, service XE.LOCAL reachable

# find host, CDB and PDB to the givven service
# this needs a proper login to the DB via --user/--password or TNSCLI_USER/TNSCLI_PASSWORD
>tnscli check -H -A test/testdata XEPDB1.local
XEPDB1.local -> localhost:XE:XEPDB1

# write tnsnames.ora to ldap server (openldap with oid* schema extensions), all parameters via commandline
>tnscli ldap write \
  --ldap.host=127.0.0.1 \
  --ldap.port=1636 -T -I \
  --ldap.base="dc=oracle,dc=local" \
  --ldap.oraclectx="dc=oracle,dc=local" \
  --ldap.binddn="cn=admin,dc=oracle,dc=local" \
  --ldap.bindpassword=admin  \
  --ldap.timeout=20 \
  --ldap.tnssource test/testdata/ldap_file_write.ora 
Finished successfully. For details run with --info or --debug

#read tnsnames.ora from ldap server with parameter via yaml and password via env
>export TNSCLI_LDAP_BINDPASSWORD=admin
>tnscli ldap read -T -I -c test/tnscli.yaml -A test/testdata 
[Tue, 23 May 2023 17:23:27 CEST]  INFO Return 2 TNS Ldap Entries
XE.LOCAL=  
        (DESCRIPTION =
                  (ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
                  (CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
        )
  
XE2.LOCAL=  
        (DESCRIPTION =
                  (ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
                  (CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE2))
        )
  
[Tue, 23 May 2023 17:23:27 CEST]  INFO SUCCESS: 2 LDAP entries found


```



