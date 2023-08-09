package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/tommi2day/tnscli/test"

	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/dblib"
	"github.com/tommi2day/gomodules/ldaplib"
)

/*
const (

	sOK   = "ok"
	sNew  = "new"
	sMod  = "mod"
	sDel  = "del"
	sSkip = "skip"

)
const ldapTimeout = 20
*/
const ldapOrganisation = "TNS Ltd"
const LdapDomain = "oracle.local"
const LdapBaseDn = "dc=oracle,dc=local"
const LdapAdminUser = "cn=admin," + LdapBaseDn
const LdapAdminPassword = "admin"
const LdapConfigPassword = "config"

const ldaptns = ` 
XE.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)
XE1.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE1))
)
XE2 =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE2.local))
)
`

const ldaptns2 = `
#equal, but lower
xe.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)
#modified desc
XE1 =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE1.local))
)
new =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = new))
)
`
const ldapOra = `
DEFAULT_ADMIN_CONTEXT = "dc=oracle,dc=local"
DIRECTORY_SERVERS = (localhost:1389:1636, ldap:389)
DIRECTORY_SERVER_TYPE = OID
`

func TestOracleLdap(t *testing.T) {
	var err error
	var server string
	var sslport int
	var out = ""

	test.Testinit(t)
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	ldapAdmin := test.TestData
	//nolint gosec
	err = os.WriteFile(ldapAdmin+"/ldap.ora", []byte(ldapOra), 0644)
	require.NoErrorf(t, err, "Create test ldap.ora failed")

	// prepare or skip container based tests
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	ldapContainer, err = prepareLdapContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, ldapContainer, "Prepare failed")
	defer common.DestroyDockerContainer(ldapContainer)
	server, sslport = common.GetContainerHostAndPort(ldapContainer, "636/tcp")
	// create test file to load
	tnsAdmin := test.TestData
	filename1 := tnsAdmin + "/ldap_file_write1.ora"
	//nolint gosec
	err = os.WriteFile(filename1, []byte(ldaptns), 0644)
	require.NoErrorf(t, err, "Create test %s failed", filename1)
	filename2 := tnsAdmin + "/ldap_file_write2.ora"
	//nolint gosec
	err = os.WriteFile(filename2, []byte(ldaptns2), 0644)
	require.NoErrorf(t, err, "Create test %s failed", filename2)

	base := LdapBaseDn
	lc := ldaplib.NewConfig(server, sslport, true, true, base, ldapTimeout)
	context := ""

	t.Run("Ldap Connect", func(t *testing.T) {
		t.Logf("Connect '%s' using SSL on port %d", LdapAdminUser, sslport)
		err = lc.Connect(LdapAdminUser, LdapAdminPassword)
		require.NoErrorf(t, err, "admin Connect returned error %v", err)
		assert.NotNilf(t, lc.Conn, "Ldap Connect is nil")
		assert.IsType(t, &ldap.Conn{}, lc.Conn, "returned object ist not ldap connection")
		if lc.Conn == nil {
			t.Fatalf("No valid Connection, terminate")
			return
		}
	})

	t.Run("Get Oracle Context", func(t *testing.T) {
		context, err = dblib.GetOracleContext(lc, base)
		expected := "cn=OracleContext," + LdapBaseDn
		assert.NotEmptyf(t, context, "Oracle Context not found")
		assert.Equal(t, expected, context, "OracleContext not as expected")
		if context == "" {
			t.Fatalf("Oracle Context object not found, terminate")
			return
		}
		t.Logf("Oracle Context: %s", context)
	})
	t.Run("Write Ldap function", func(t *testing.T) {
		err = os.Chdir(test.TestDir)
		require.NoErrorf(t, err, "ChDir failed")
		t.Logf("load from %s", filename1)

		// read entries from file
		fileTnsEntries, domain, err := dblib.GetTnsnames(filename1, true)
		require.NoErrorf(t, err, "Parsing %s failed: %s", filename1, err)
		if err != nil {
			t.Fatalf("tns load returned error: %s ", err)
			return
		}

		// write entries to ldap
		var workstatus TWorkStatus
		workstatus, err = WriteLdapTns(lc, fileTnsEntries, domain, context)
		require.NoErrorf(t, err, "Write TNS to Ldap failed: %s", err)
		expected := len(fileTnsEntries)
		actual := workstatus[sNew]
		require.Equal(t, expected, actual, "Not all Records has been added")
		t.Logf("%d Entries added", actual)
	})

	if err != nil {
		t.Fatalf("need Write TNS to proceed")
		return
	}

	t.Run("Modify Ldap function", func(t *testing.T) {
		err = os.Chdir(test.TestDir)
		require.NoErrorf(t, err, "ChDir failed")

		t.Logf("load from %s", filename2)
		// read entries from file
		fileTnsEntries, domain, err := dblib.GetTnsnames(filename2, true)
		require.NoErrorf(t, err, "Parsing %s failed: %s", filename2, err)
		if err != nil {
			t.Fatalf("tns load returned error: %s ", err)
			return
		}
		require.Equal(t, 3, len(fileTnsEntries), "update TNS should have 3 entries")
		// write entries to ldap
		var workstatus TWorkStatus
		workstatus, err = WriteLdapTns(lc, fileTnsEntries, domain, context)
		require.NoErrorf(t, err, "Write TNS to Ldap failed: %s", err)
		o := workstatus[sOK]
		n := workstatus[sNew]
		m := workstatus[sMod]
		d := workstatus[sDel]
		s := workstatus[sSkip]
		assert.Equal(t, 1, o, "One OK expected")
		assert.Equal(t, 1, n, "One Adds expected")
		assert.Equal(t, 1, m, "One mod expected")
		assert.Equal(t, 1, d, "One del expected")
		assert.Equal(t, 0, s, "No skip expected")
	})
	t.Run("Clear Ldap function", func(t *testing.T) {
		_, f := ClearLdapTns(lc, context)
		require.Equalf(t, 0, f, "Clearing TNS Ldap had %d failures", f)
	})
	t.Run("Write TNS to Ldap", func(t *testing.T) {
		tnsAdmin := test.TestData
		filename := tnsAdmin + "/ldap_file_write1.ora"
		args := []string{
			"ldap",
			"write",
			"--ldap.oraclectx", LdapBaseDn,
			"--ldap.host", server,
			"--ldap.port", fmt.Sprintf("%d", sslport),
			"--ldap.tls", "true",
			"--ldap.insecure", "true",
			"--ldap.base", LdapBaseDn,
			"--ldap.binddn", LdapAdminUser,
			"--ldap.bindpassword", LdapAdminPassword,
			"--ldap.tnssource", filename,
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		require.NoErrorf(t, err, "Command returned error: %s", err)
		t.Logf(out)
		assert.Containsf(t, out, "SUCCESS: ", "Output not as expected")
	})

	t.Run("Read TNS from Ldap with config file and env", func(t *testing.T) {
		tnsAdmin := test.TestData
		filename := tnsAdmin + "/ldap_file_read.ora"
		_ = os.Remove(filename)
		_ = os.Setenv("TNSCLI_LDAP_BINDPASSWORD", LdapAdminPassword)
		args := []string{
			"ldap",
			"read",
			"--ldap.host", server,
			"--ldap.port", fmt.Sprintf("%d", sslport),
			"--ldap.tnstarget", filename,
			"--config", test.TestDir + "/tnscli.yaml",
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		require.NoErrorf(t, err, "Command returned error:%s", err)
		t.Logf(out)
		assert.FileExistsf(t, filename, "Output File not created")
		assert.Containsf(t, out, "SUCCESS: ", "Output not as expected")
	})

	t.Run("Clear TNS Entries from Ldap with config file and env", func(t *testing.T) {
		_ = os.Setenv("TNSCLI_LDAP_BINDPASSWORD", LdapAdminPassword)
		args := []string{
			"ldap",
			"clear",
			"--ldap.host", server,
			"--ldap.port", fmt.Sprintf("%d", sslport),
			"--config", test.TestDir + "/tnscli.yaml",
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		require.NoErrorf(t, err, "Command returned error:%s", err)
		t.Logf(out)
		assert.Containsf(t, out, "SUCCESS: ", "Output not as expected")
	})
}
