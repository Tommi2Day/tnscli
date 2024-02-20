# Changelog tnscli

## [v3.9.0 - 2024-02-20]
### New
- use now bitnami/openldap:2.6.7 as ldap test container
- add ldap bind password prompt
- Read Bind credentials also from LDAP_BIND_DN and *_PASSWORD environment variables
### Changed
- update gomodules to v1.11.4
- move docker provision to test folder
- update dependencies
- update pipeline
- use Oracle-Free 23.3 as test container, which causes to replace XE* with FREE*
- update dblib to v1.6.5
- remove tools.go and rely on preloaded tools in golang image
### Fixed
- fix goreleaser version


## [v3.8.2 - 2023-10-31]
### New
- service info jdbc: replace TRANSPORT_CONNECT_TIMEOUT in JDBC connect string if set <1000 
to be in milliseconds as required by Oracle JDBC >12.1. 
You may disable this with --noModifyTransportConnectTimeout
### Changed
- rename TCP portcheck result `UNKNOWN` to `PROBLEM` and add err message
- enhance tcp check output
- add ldap config test
### fixed
- service info jdbc: wrong jdbc prefix spelling

## [v3.8.0 - 2023-10-27]
### Changed
- use go 1.21
- update workflow
- use gomodules v1.10.0
- update testinit
- fix linter issues

## [v3.7.11 - 2023-09-08]
### Fixed
- jdbc connect string format error

## [v3.7.10 - 2023-08-15]
### Changed
- use gomodules v1.9.4
### Fixed
- dhost flag naming

## [v3.7.9 - 2023-08-09]
### New
- add new flag --unit-tests to redirect output for unit tests
### Changed
- use gomodules v1.9.3
- use common.CmdRun instead of cmdTest
- use common.CmdFlagChanged instead of cmd.Flags().Lookup().Changed

# \[v3.7.8\] - 2023-08-04
### New
- add version test
### Changed
- move tests to there packages

## \[v3.7.7\] - 2023-08-01
### New
- add no-color switch to explicit disable colored output
- add helper scripts
### Changed
- refactor config file handling
### Fixed
- service address test

## \[ v3.7.0 \] - 2023-07-16
### New
- add service info command  with ports, jdbc and tns subcommand
- add rac vip addresses
- add portcheck command
### Changed
- rename check command to service check subcommand
- update dependencies
- add more ldap info output
- refactor docker tests
- update gomodules to v1.9.0

## \[ v3.6.0 \] - 2023-06-19
### New
- add ldap clear command
### Changed
- use $HOME/etc and current dir as config search path
- update gomodules to v1.8.0
- enhance ldap tests
### Fixed
- ldap write and ldap OracleContext handling
- TNS list test

## \[ v3.5.0 \] - 2023-05-22
### New
- add --config option to use a config file
- use viper to handle parameter settings via config and environment variables
### Changed
- use golang 1.20
- rename -c short parameter for complete in ldap list to -C
### Fixed
- dbhost query

## \[ v3.4.2 \] - 2023-05-19
### Changed
- update gomodules to v1.7.4
- change goreleaser version date string and changelog

## \[ v3.4.1 \] - 2023-05-16
### Changed
- update gomodules to v1.7.2
- update list test
### Fixed
- fix SID parsing

## \[ v3.4.0 \] - 2023-05-15
### New
- add dbhost switch to check command for current host/CDB/PDB
### Changed
- split ldap connect and check function to make linter happy
- reduce gocyclo linter threshold

## \[ v3.3.0 \] - 2023-05-07
- first public release