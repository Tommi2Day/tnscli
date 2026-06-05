# tnscli

Small Oracle TNS Service and Connect Test Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/tommi2day/tnscli)](https://goreportcard.com/report/github.com/tommi2day/tnscli)
![CI](https://github.com/tommi2day/tnscli/actions/workflows/main.yml/badge.svg)
[![codecov](https://codecov.io/gh/Tommi2Day/tnscli/branch/main/graph/badge.svg?token=C1IP9AMBUM)](https://codecov.io/gh/Tommi2Day/tnscli)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tommi2day/tnscli)


## Features

- connect to a given service using real connect method.
- Uses given credentials or default to raise an ORA-1017 error
- List and search TNS entries from `tnsnames.ora` and LDAP
- Check TNS service connectivity via TCP (and optionally a real database login)
- Port check all addresses of a service, including RAC VIPs
- RAC address resolution via DNS SRV records or `racinfo.ini`
- Service detail queries: address list, JDBC connection string, raw TNS descriptor
- LDAP TNS entry management (read, write, clear) via OpenLDAP with OID schema
- Configurable via YAML config file, environment variables, or CLI flags
- Addon scripts: `dbhost`, `gotodb`, `tnslookup`

## Contents

- [Installation](#installation)
- [Configuration](#configuration)
  - [DB test user](#db-test-user)
  - [RAC address info](#rac-address-info)
- [list — List TNS entries](#list--list-tns-entries)
- [service check — Check TNS entries](#service-check--check-tns-entries)
- [service portcheck — Port check](#service-portcheck--port-check)
- [service info — Service details](#service-info--service-details)
  - [service info ports](#service-info-ports--list-addresses-and-ports)
  - [service info jdbc](#service-info-jdbc--print-jdbc-string)
  - [service info tns](#service-info-tns--print-tns-entry)
- [ldap — LDAP TNS entries](#ldap--ldap-tns-entries)
  - [ldap read](#ldap-read--read-tns-entries-from-ldap)
  - [ldap write](#ldap-write--write-tns-entries-to-ldap)
  - [ldap clear](#ldap-clear--clear-ldap-tns-entries)
- [Addon scripts](#addon-scripts)
- [Global flags](#global-flags)
- [version](#version--print-version-information)

---

## Installation

Download the latest release binaries from [GitLab](https://gitlab.intern.tdressler.net/goproj/tnscli/-/releases)
or install with Go:

```sh
go install gitlab.intern.tdressler.net/goproj/tnscli@latest
```

---

## Configuration

Settings are resolved in this order (later sources override earlier ones):

1. YAML config file — searched in order, first found wins:
   - `./tnscli.yaml` (current directory)
   - `$HOME/.config/tnscli.yaml`
   - `$HOME/etc/tnscli.yaml`
   - `/etc/tnscli.yaml`
   - or set explicitly with `--config` / `-c`
2. Environment variables — prefix `TNSCLI_`, dots replaced by underscores (e.g. `TNSCLI_LDAP_HOST`)
3. CLI flags

### DB test user

**CAUTION**: Do not use anonymous checks for permanent monitoring. Some security analysis systems treat frequent anonymous connect attempts as a brute-force attack. Instead, set `TNSCLI_USER` and `TNSCLI_PASSWORD` env vars or use `--user` / `--password` flags in the `service check` command. The user only needs a `connect` privilege.

Replace `c##tcheck`, `tcheck`, and `<MyCheckPassword>` with your own values.

- **Common user in CDB$ROOT:**

  ```sql
  alter session set container=cdb$root;
  create user c##tcheck identified by "<MyCheckPassword>"
      default tablespace users temporary tablespace temp
      account unlock container=all;
  grant connect to c##tcheck container=all;
  alter user c##tcheck default role all container=all;
  ```

- **Traditional (non-CDB) user:**

  ```sql
  create user tcheck identified by "<MyCheckPassword>"
      default tablespace users temporary tablespace temp
      account unlock;
  grant connect to tcheck;
  alter user tcheck default role all;
  ```

- **Export credentials to the environment:**

  ```bash
  export TNSCLI_USER="c##tcheck"
  export TNSCLI_PASSWORD="<MyCheckPassword>"
  ```

### RAC address info

RAC address details can be provided via DNS SRV records or a `racinfo.ini` file placed in `$TNS_ADMIN`.

- **DNS SRV format:**

  ```
  _myrac._tcp.rac.lan.  IN SRV 10 5 1521 myrac.rac.lan.
  _myrac._tcp.rac.lan.  IN SRV 10 5 1521 vip1.rac.lan.
  _myrac._tcp.rac.lan.  IN SRV 10 5 1521 vip2.rac.lan.
  _myrac._tcp.rac.lan.  IN SRV 10 5 1521 vip3.rac.lan.
  ```

- **racinfo.ini format** (`$TNS_ADMIN/racinfo.ini`):

  ```ini
  [MYRAC.RAC.LAN]
  scan=myrac.rac.lan:1521
  vip1=vip1.rac.lan:1521
  vip2=vip2.rac.lan:1521
  vip3=vip3.rac.lan:1521
  ```

---

## list — List TNS entries

```sh
tnscli list [flags]
```

Lists all TNS aliases found in the configured `tnsnames.ora`, or searches for a specific alias.

| Flag | Description |
|------|-------------|
| `--complete` / `-C` | Print the full TNS descriptor for each entry |
| `--search` / `-s` | Filter output to aliases matching this string |

**Examples:**

```sh
# List all alias names
tnscli list -f test/testdata/connect.ora

# List entries with full descriptor
tnscli list --complete -f test/testdata/connect.ora

# Search for a specific alias
tnscli list --search mydb
```

---

## service check — Check TNS entries

```sh
tnscli service check [flags]
```

Performs a real TCP connect (and optionally a database login) to verify a TNS entry is reachable. Without `--user`/`--password`, connects with the built-in dummy credentials and expects an `ORA-01017` login error — confirming the port is open without requiring a real account.

| Flag | Description |
|------|-------------|
| `--service` / `-s` | Service alias to check **(required unless `--all`)** |
| `--all` / `-a` | Check all entries in the TNS file |
| `--user` / `-u` | Username for real connect (or set `TNSCLI_USER`) |
| `--password` / `-p` | Password for real connect (or set `TNSCLI_PASSWORD`) |
| `--timeout` / `-t` | Connect timeout in seconds (default 15) |
| `--dbhost` / `-H` | Print the actual connected host, CDB, and PDB from `sys_context` |

**Examples:**

```sh
# Check with dummy credentials — confirms port is open, expects ORA-01017
tnscli service check -s xe.local -f test/testdata/connect.ora --info

# Check with real credentials
tnscli service check -s xe.local -f test/testdata/connect.ora \
  --user system --password supersecret

# Use env vars for credentials
export TNSCLI_USER="c##tcheck"
export TNSCLI_PASSWORD="<MyCheckPassword>"
tnscli service check -s xe.local

# Print connected host:CDB:PDB (requires valid login)
tnscli service check -s XEPDB1.local -H -A test/testdata

# Check all entries in a file
tnscli service check --all -f test/testdata/connect.ora
```

---

## service portcheck — Port check

```sh
tnscli service portcheck [flags]
```

Performs a TCP connect test on all host:port addresses for a service. If RAC info is available (DNS SRV or `racinfo.ini`), all RAC addresses are checked as well.

| Flag | Description |
|------|-------------|
| `--service` / `-s` | Service alias to check |
| `--nodns` | Do not resolve hostnames via DNS |
| `--nameserver` / `-n` | Alternative nameserver for DNS lookups (`IP:PORT`) |
| `--dnstcp` | Use TCP instead of UDP for DNS queries |
| `--ipv4` | Resolve IPv4 addresses only |
| `--racinfo` / `-r` | Path to `racinfo.ini` (default `$TNS_ADMIN/racinfo.ini`) |
| `--timeout` / `-t` | TCP connect timeout in seconds (default 5) |

**Examples:**

```sh
# Check if all ports for a service are open
tnscli service portcheck -s xe.local -A test/testdata

# Check with an alternative nameserver
tnscli service portcheck -s myrac -f test/testdata/rac.ora \
  --nameserver 127.0.0.1:53
```

---

## service info — Service details

```sh
tnscli service info <subcommand> [flags]
```

Parent command for service detail subcommands. All subcommands share these flags:

| Flag | Description |
|------|-------------|
| `--service` / `-s` | Service alias |
| `--filename` / `-f` | Path to `tnsnames.ora` |
| `--tns_admin` / `-A` | `TNS_ADMIN` directory |

### service info ports — List addresses and ports

```sh
tnscli service info ports [flags]
```

Lists all host:port addresses for a service. Resolves RAC addresses from DNS SRV records or `racinfo.ini` when available.

| Flag | Description |
|------|-------------|
| `--nodns` | Do not resolve hostnames via DNS |
| `--nameserver` / `-n` | Alternative nameserver (`IP:PORT`) |
| `--dnstcp` | Use TCP for DNS queries |
| `--ipv4` | Resolve IPv4 addresses only |
| `--racinfo` / `-r` | Path to `racinfo.ini` (default `$TNS_ADMIN/racinfo.ini`) |

**Examples:**

```sh
# List all addresses (with DNS SRV resolution)
tnscli service info ports -s myrac -f test/testdata/rac.ora --nameserver 127.0.0.1

# List addresses from racinfo.ini without DNS resolution
tnscli service info ports -s myrac -f test/testdata/rac.ora \
  -r test/testdata/racinfo.ini --nodns
```

### service info jdbc — Print JDBC string

```sh
tnscli service info jdbc [flags]
```

Prints the JDBC thin connection string for a service.

| Flag | Description |
|------|-------------|
| `--noModifyTransportConnectTimeout` | Keep `TRANSPORT_CONNECT_TIMEOUT` value as-is (default: convert from seconds to milliseconds) |

**Examples:**

```sh
tnscli service info jdbc -s xe -A test/testdata/
# jdbc:oracle:thin:@(DESCRIPTION=(...))
```

### service info tns — Print TNS entry

```sh
tnscli service info tns [flags]
```

Prints the raw TNS descriptor for a service alias.

**Examples:**

```sh
tnscli service info tns -s xe -A test/testdata/
```

---

## ldap — LDAP TNS entries

```sh
tnscli ldap <subcommand> [flags]
```

Reads and writes TNS entries stored in an OpenLDAP server with OID schema extensions. All `ldap` subcommands share these connection flags:

| Flag | Description |
|------|-------------|
| `--ldap.host` / `-H` | LDAP server hostname |
| `--ldap.port` / `-p` | LDAP port (0 = derive from `--ldap.tls`) |
| `--ldap.tls` / `-T` | Use LDAPS (implicit TLS) |
| `--ldap.insecure` / `-I` | Skip TLS certificate verification |
| `--ldap.binddn` / `-D` | Bind DN for LDAP authentication (empty = anonymous) |
| `--ldap.bindpassword` / `-w` | Bind password (or set `TNSCLI_LDAP_BINDPASSWORD`) |
| `--ldap.base` / `-b` | Base DN to search from |
| `--ldap.oraclectx` / `-o` | Base DN of the Oracle Context |
| `--ldap.timeout` | LDAP operation timeout in seconds (default 20) |

### ldap read — Read TNS entries from LDAP

```sh
tnscli ldap read [flags]
```

Reads TNS entries from the LDAP server and prints them to stdout or writes them to a file.

**Examples:**

```sh
# Read with config file and password from env
export TNSCLI_LDAP_BINDPASSWORD=admin
tnscli ldap read -T -I -c test/tnscli.yaml -A test/testdata
```

### ldap write — Write TNS entries to LDAP

```sh
tnscli ldap write [flags]
```

Reads a `tnsnames.ora` file and writes the entries to the LDAP server.

| Flag | Description |
|------|-------------|
| `--ldap.tnssource` | Path to the `tnsnames.ora` source file |

**Examples:**

```sh
tnscli ldap write \
  --ldap.host=127.0.0.1 --ldap.port=1636 -T -I \
  --ldap.base="dc=oracle,dc=local" \
  --ldap.oraclectx="dc=oracle,dc=local" \
  --ldap.binddn="cn=admin,dc=oracle,dc=local" \
  --ldap.bindpassword=admin \
  --ldap.timeout=20 \
  --ldap.tnssource test/testdata/ldap_file_write.ora
```

### ldap clear — Clear LDAP TNS entries

```sh
tnscli ldap clear [flags]
```

Removes all TNS entries from the LDAP Oracle Context.

**Examples:**

```sh
tnscli ldap clear \
  --ldap.host=127.0.0.1 --ldap.port=1636 -T -I \
  --ldap.base="dc=oracle,dc=local" \
  --ldap.oraclectx="dc=oracle,dc=local" \
  --ldap.binddn="cn=admin,dc=oracle,dc=local" \
  --ldap.bindpassword=admin
```

---

## Addon scripts

Helper scripts are available in the `/scripts` directory.

### dbhost

Shortcut for `tnscli service check <service> --dbhost`. Connects to the given service and prints `host:CDB:PDB` from `sys_context`.

```bash
export TNSCLI_USER="c##tcheck"
export TNSCLI_PASSWORD="<MyCheckPassword>"
dbhost MYPDB1
# racnode1:MYCDB:PDB1
```

### gotodb

Uses `dbhost` to extract the server hostname and opens an SSH session to that host. Requires both `tnscli` and `dbhost` in `PATH`. Use `~/.ssh/config` if the returned hostname is not directly resolvable:

```
Host racnode1
    HostName racnode1.rac.lan
    User oracle
    IdentityFile ~/.ssh/id_ora
```

```bash
gotodb MYPDB1
# oracle@racnode1 ~>
```

### tnslookup

Shortcut for `tnscli list --search <service> --complete`.

```bash
tnslookup mypdb1
# Location: /etc/oracle/tnsnames.ora Line: 6
# MYPDB1.LOCAL=  (DESCRIPTION = ...)
```

---

## Global flags

These flags apply to every command:

| Flag | Description |
|------|-------------|
| `--config` / `-c` | Path to config file (overrides auto-discovery) |
| `--filename` / `-f` | Path to alternate `tnsnames.ora` |
| `--tns_admin` / `-A` | `TNS_ADMIN` directory (default `$TNS_ADMIN`) |
| `--debug` | Verbose debug output |
| `--info` | Reduced info output |
| `--no-color` | Disable coloured log output |

---

## version — Print version information

```sh
tnscli version
```

Prints the build version, commit hash, and build date injected at release time.

---

