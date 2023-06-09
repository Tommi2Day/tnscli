# Changelog tnscli

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