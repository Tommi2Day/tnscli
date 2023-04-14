# tnscli
[![Go Report Card](https://goreportcard.com/badge/github.com/tommi2day/tnscli)](https://goreportcard.com/report/github.com/tommi2day/tnscli)
![CI](https://github.com/tommi2day/tnscli/actions/workflows/main.yml/badge.svg)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tommi2day/tnscli)
tool for oracle tns functions

- connect to a given service using real connect method. 
- Uses given credentials or default to raise an ORA-1017 error
- search or list tns entries
- load and write tnsnames.ora to ldap

**CAUTION**: Dont use anonymous checks for monitoring. Some security analysis systems are qualifying this as "Brute-Force-Attack" if the check are started too often
## Usage
```bash
tnscli – small TNS Service and Connect Tool

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
      --debug              verbose debug output
  -f, --filename string    path to alternate tnsnames.ora
  -h, --help               help for tnscli
      --info               reduced info output
  -A, --tns_admin string   TNS_ADMIN directory (default "$TNS_ADMIN")

Use "tnscli [command] --help" for more information about a command.

tnscli check [flags]

Flags:
  -a, --all               check all entries
  -h, --help              help for check
  -p, --password string   Password for real connect or set TNSCLI_PASSWORD
  -s, --service string    service name to check
  -t, --timeout int       timeout in sec (default 15)
  -u, --user string       User for real connect or set TNSCLI_USER

tnscli list [flags]

Flags:
  -c, --complete        print complete entry
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
tnscli version 3.0.1-next (snapshot - 2023-04-12T19:12:20Z)

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

>tnscli ldap write --ldap.host=127.0.0.1 --ldap.port=1636 -T -I --ldap.base="dc=oracle,dc=local" --ldap.binddn="cn=admin,dc=oracle,dc=local" --ldap.bindpassword=admin  --ldap.timeout=20 --ldap.tnssource test/testdata/ldap_file_write.ora 
Finished successfully. For details run with --info or --debug

>tnscli ldap read -T -I --ldap.binddn="cn=admin,dc=oracle,dc=local" --ldap.bindpassword=admin -A test/testdata --debug
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG ldapRead called
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG CTX: dc=oracle,dc=local, Servers [{localhost 1389 1636} {ldap 389 0}]
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG Try to use LDAP Server from ldap.ora
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG Try to connect to Ldap Server localhost, Port 1636
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG Connect to Ldap Server localhost, Port 1636
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG Found 2 TNS Ldap Entries
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG parsed Host: 127.0.0.1, Port 1521
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG found TNS Alias XE2.LOCAL
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG parsed Host: 127.0.0.1, Port 1521
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG found TNS Alias XE.LOCAL
[Thu, 13 Apr 2023 21:47:31 CEST]  INFO Return 2 TNS Ldap Entries
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG write to StdOut
[Thu, 13 Apr 2023 21:47:31 CEST] DEBUG list 2 entries
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

[Thu, 13 Apr 2023 21:47:31 CEST]  INFO SUCCESS: 2 LDAP entries found

```



