package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ldapOrganisation = "TNS Ltd"
const LdapDomain = "oracle.local"
const LdapBaseDn = "dc=oracle,dc=local"
const LdapAdminUser = "cn=admin," + LdapBaseDn
const LdapAdminPassword = "admin"
const LdapConfigPassword = "config"

const tnsWrite = ` 
XE2.local =
	(DESCRIPTION =
		(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
		(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE2))
	)
XE.local =
	(DESCRIPTION =
		(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
		(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
	)
`

const ldapOra = `
DEFAULT_ADMIN_CONTEXT = "dc=oracle,dc=local"
DIRECTORY_SERVERS = (localhost:1389:1636, ldap:389)
DIRECTORY_SERVER_TYPE = OID
`
const ldapTimeout = 20

func TestOracleLdap(t *testing.T) {
	var err error
	var server string
	var sslport int
	var out = ""

	Testinit(t)
	err = os.Chdir(TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	ldapAdmin := TestData
	//nolint gosec
	err = os.WriteFile(ldapAdmin+"/ldap.ora", []byte(ldapOra), 0644)
	require.NoErrorf(t, err, "Create test ldap.ora failed")
	//nolint gosec
	err = os.WriteFile(ldapAdmin+"/ldap_file_write.ora", []byte(tnsWrite), 0644)
	require.NoErrorf(t, err, "Create test ldap.ora failed")

	// prepare or skip container based tests
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	ldapContainer, err = prepareLdapContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, ldapContainer, "Prepare failed")
	defer destroyLdapContainer(ldapContainer)
	server, sslport = getLdapHostAndPort(ldapContainer, "636/tcp")

	t.Run("Write TNS to Ldap", func(t *testing.T) {
		tnsAdmin := TestData
		filename := tnsAdmin + "/ldap_file_write.ora"
		args := []string{
			"ldap",
			"write",
			"--ldap.host", server,
			"--ldap.port", fmt.Sprintf("%d", sslport),
			"--ldap.tls", "true",
			"--ldap.insecure", "true",
			"--ldap.base", LdapBaseDn,
			"--ldap.binddn", LdapAdminUser,
			"--ldap.bindpassword", LdapAdminPassword,
			"--ldap.tnssource", filename,
			"--ldap.timeout", fmt.Sprintf("%d", ldapTimeout),
			"--info",
		}
		out, err = cmdTest(args)
		require.NoErrorf(t, err, "Command returned error: %s", err)
		t.Logf(out)
		assert.Containsf(t, out, "SUCCESS: ", "Output not as expected")
	})

	t.Run("Read TNS from Ldap", func(t *testing.T) {
		tnsAdmin := TestData
		filename := tnsAdmin + "/ldap_file_read.ora"
		_ = os.Remove(filename)
		args := []string{
			"ldap",
			"read",
			"--ldap.host", server,
			"--ldap.port", fmt.Sprintf("%d", sslport),
			"--ldap.tls", "true",
			"--ldap.insecure", "true",
			"--ldap.base", LdapBaseDn,
			"--ldap.binddn", LdapAdminUser,
			"--ldap.bindpassword", LdapAdminPassword,
			"--ldap.tnstarget", filename,
			"--info",
		}
		out, err = cmdTest(args)
		require.NoErrorf(t, err, "Command returned error:%s", err)
		t.Logf(out)
		assert.FileExistsf(t, filename, "Output File not created")
		assert.Containsf(t, out, "SUCCESS: ", "Output not as expected")
	})
}
