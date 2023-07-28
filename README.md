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
- generates JDBC String for a service
- list all affected RAC Hosts/Ports for a given service using DNS SRV entries or racinfo file
- run a portcheck(TCP connect test) on all needed ports

## Setup
### recommanded: setup db test user
**CAUTION**: Don't use anonymous checks for permanent monitoring. Some security analysis systems are qualifying this as "Brute-Force-Attack" if the check are started too often. 
Instead, set $TNSCLI_USER and TNSCLI_PASSWORD env or use --user and --password flag in check command to connect an existing user. This user needs only a connect privilege.
Replace `c##tcheck`, `tcheck` and `<MyCheckPassword>` with your own secrets
-   **sample for set up a common user within CDB$ROOT**
 
    ```sql
    alter session set container=cdb$root;
    create user c##tcheck identified by "<MyCheckPassword>"
        default tablespace users temporary tablespace temp
        account unlock container=all;
    grant connect to c##tcheck container=all;
    alter user c##tcheck default role all container=all;
    ```
-   **sample for set up a traditional (non-cdb) user **
    ```sql
    create user tcheck identified by "<MyCheckPassword>"
        default tablespace users temporary tablespace temp
        account unlock;
    grant connect to tcheck;
    alter user c##tcheck default role all container=all;
    ```
- export user secrets to environment
  ```bash
  export TNSCLI_USER="c##tcheck" # or 
  # export TNSCLI_USER="tcheck"
  export TNSCLI_PASSWORD="<MyCheckPassword>"
  ``` 
### optional: setup RAC Address info
ORACLE address info can be provided with DNS SRV entries or a racinfo.ini file in $TNS_ADMIN directory.

-   *DNS SRV format:*
    ```
    _myrac._tcp.rac.lan.  IN SRV 10 5 1521 myrac.rac.lan.
    _myrac._tcp.rac.lan.  IN SRV 10 5 1521 vip1.rac.lan.
    _myrac._tcp.rac.lan.  IN SRV 10 5 1521 vip2.rac.lan.
    _myrac._tcp.rac.lan.  IN SRV 10 5 1521 vip3.rac.lan.
    ```

-   *racinfo.ini format*
    ```
    [RAC DNS Name as in tnsnames HOST Entry]
    san=scan-address:port
    vip1=vip-address1:port
    vip2=vip-address2:port
    ...
    Example:
    [MYRAC.RAC.LAN]
    scan=myrac.rac.lan:1521
    vip1=vip1.rac.lan:1521
    vip2=vip2.rac.lan:1521
    vip3=vip3.rac.lan:1521
    ``` 


## Usage
```bash
tnscli – Small Oracle TNS Service and Connect Test Tool

Usage:                         
  tnscli [command]             
                               
Available Commands:            
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  ldap        LDAP TNS Entries
  list        list TNS Entries
  service     Service sub command
  version     version print version string

Flags:
  -c, --config string      config file
      --debug              verbose debug output
  -f, --filename string    path to alternate tnsnames.ora
  -h, --help               help for tnscli
      --info               reduced info output
      --no-color           disable colored log output
  -A, --tns_admin string   TNS_ADMIN directory (default "$TNS_ADMIN")

Use "tnscli [command] --help" for more information about a command.

tnscli service [command]

Available Commands:
  check       Check TNS Entries
  info        give details for the given service
  portcheck   try to connect each service and report if it is open or not

Flags:
  -h, --help             help for service
  -s, --service string   service name to check


tnscli service check [flags]
Check all TNS Entries or one with real connect to database
Flags:
  -a, --all               check all entries
  -H, --dbhost            print actual connected host:cdb:pdb
  -h, --help              help for check
  -p, --password string   Password for real connect or set TNSCLI_PASSWORD
  -t, --timeout int       timeout in sec (default 15)
  -u, --user string       User for real connect or set TNSCLI_USER

Global Flags:
  -c, --config string      config file
      --debug              verbose debug output
  -f, --filename string    path to alternate tnsnames.ora
      --info               reduced info output
  -s, --service string     service name to check
  -A, --tns_admin string   TNS_ADMIN directory (default "$TNS_ADMIN")


tnscli service portcheck [flags]
list defined host:port and checks if requested. If racinfo.ini or SRV info given,  addresses will be checked as well
Flags:
      --dnstcp              Use TCP to resolve DNS names
  -h, --help                help for portcheck
      --ipv4                resolve only IPv4 addresses
  -n, --nameserver string   alternative nameserver to use for DNS lookup (IP:PORT)
      --nodns               do not use DNS to resolve hostnames
  -r, --racinfo string      path to racinfo.ini to resolve all RAC TCP Adresses, default $TNS_ADMIN/racinfo.ini
  -t, --timeout int         timeout for tcp ping (default 5)


tnscli service info [command]

Available Commands:
  jdbc        print tns entry as jdbc string
  ports       list service addresses and ports
  tns         print tns entry for the given service

Flags:
  -h, --help   help for info

Global Flags:
  -c, --config string      config file
      --debug              verbose debug output
  -f, --filename string    path to alternate tnsnames.ora
      --info               reduced info output
  -s, --service string     service name to check
  -A, --tns_admin string   TNS_ADMIN directory (default "$TNS_ADMIN")

tnscli service info ports [flags]
list defined host:port and checks if requested. If racinfo.ini given, it will be listed as well
Flags:
      --dnstcp              Use TCP to resolve DNS names
  -h, --help                help for ports
      --ipv4                resolve only IPv4 addresses
  -n, --nameserver string   alternative nameserver to use for DNS lookup (IP:PORT)
      --nodns               do not use DNS to resolve hostnames
  -r, --racinfo string      path to racinfo.ini to resolve all RAC TCP Adresses, default $TNS_ADMIN/racinfo.ini



tnscli list [flags]
list all TNS Entries or search one
Flags:
  -C, --complete        print complete entry
  -h, --help            help for list
  -s, --search string   service name to check

tnscli ldap [command]
handle TNS entries stored in LDAP Server
Available Commands:
  clear       clear ldap tns entries
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
# Location: test/testdata/connect.ora Line: 1
XE.LOCAL=  (DESCRIPTION=  (ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=21521)))  (CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=XEPDB1)))


# list nonexisting service
>tnscli list --search  mydb
Error: no alias with 'mydb' found

# give tNS String for a service
>tnscli service info tns xe -A test/testdata/
# Location: ifile.ora Line: 6 
XE.LOCAL=  (DESCRIPTION =
          (ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
          (CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
  )

#give jdbc string for a service
>tnscli service info jdbc xe -A test/testdata/
jdbc:Oracle:thin@(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=XE)))

#give target host and port for a service
>tnscli service info server xe -A test/testdata/
127.0.0.1:1521

# check if port is open for a service
>tnscli service portcheck xe.local -A test/testdata
127.0.0.1 (127.0.0.1:1521) is OPEN

# give ALL target host and port for a service with RAC and DNS SRV resolution
>tnscli service info ports myrac -f test/testdata/rac.ora --nameserver 127.0.0.1
vip1.rac.lan (172.24.0.13:1521)
vip3.rac.lan (172.24.0.15:1521)
vip2.rac.lan (172.24.0.14:1521)
myrac.rac.lan (172.24.0.3:1521)
myrac.rac.lan (172.24.0.4:1521)
myrac.rac.lan (172.24.0.5:1521)

# static resolution with racinfo.ini without DNS. With DNS access hostnames will be resolved to IP addresses as above
service info ports myrac -f test/testdata/rac.ora -r test/testdata/racinfo.ini --nodns
myrac.rac.lan (myrac.rac.lan:1521)
vip1.rac.lan (vip1.rac.lan:1521)
vip2.rac.lan (vip2.rac.lan:1521)
vip3.rac.lan (vip3.rac.lan:1521)


# login check for unavailable service with tnsnames.ora in $TNS_ADMIN
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
## Virus Warnings

some engines are reporting a virus in the binaries. This is a false positive. You may check the binaries with meta engines such as [virustotal.com](https://www.virustotal.com/gui/home/upload) or build your own binary from source. I have no glue why this happens.



