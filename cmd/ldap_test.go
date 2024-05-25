package cmd

import (
	"fmt"
	"os"
	"path"
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
const ldapora = `
DEFAULT_ADMIN_CONTEXT = "dc=oracle,dc=local"
DIRECTORY_SERVERS = (localhost:1389:1636, localhost:1389)
DIRECTORY_SERVER_TYPE = OID
`

func TestOracleLdap(t *testing.T) {
	var err error
	var server string
	var port int
	var out = ""
	var domain string
	var fileTnsEntries dblib.TNSEntries

	test.Testinit(t)
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	ldapAdmin := test.TestData
	tnsAdmin = test.TestData
	testConfig := path.Join(test.TestDir, "tnscli.yaml")
	tnsSource1 := path.Join(tnsAdmin, "/ldap_file_write1.ora")
	tnsSource2 := path.Join(tnsAdmin, "/ldap_file_write2.ora")

	//nolint gosec
	err = os.WriteFile(ldapAdmin+"/ldap.ora", []byte(ldapora), 0644)
	require.NoErrorf(t, err, "Create test ldap.ora failed")

	// prepare or skip container based tests
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	ldapContainer, err = prepareLdapContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, ldapContainer, "Prepare failed")
	defer common.DestroyDockerContainer(ldapContainer)
	server, port = common.GetContainerHostAndPort(ldapContainer, "1389/tcp")
	// create test file to load

	//nolint gosec
	err = os.WriteFile(tnsSource1, []byte(ldaptns), 0644)
	require.NoErrorf(t, err, "Create test %s failed", tnsSource1)

	//nolint gosec
	err = os.WriteFile(tnsSource2, []byte(ldaptns2), 0644)
	require.NoErrorf(t, err, "Create test %s failed", tnsSource2)

	t.Run("Ldap Config with ldap.ora", func(t *testing.T) {
		contextDN = ""
		lc, _ := ldapConnect()
		require.NotNil(t, lc, "Ldap Config is nil")
		if lc == nil {
			t.Fatalf("No valid Configuraction, terminate")
			return
		}
		t.Logf("Ldap Config: H:%s,P:%d,B:%s,T:%v", lc.Server, lc.Port, lc.BaseDN, lc.TLS)
		assert.Equal(t, ldapBaseDN, lc.BaseDN, "BaseDN not as expected")
		assert.Equal(t, 1389, lc.Port, "Port not as expected")
		assert.Equal(t, "localhost", lc.Server, "Host not as expected")
		assert.Equal(t, false, lc.TLS, "TLS flag not as expected")
	})
	base := LdapBaseDn
	lc := ldaplib.NewConfig(server, port, false, false, base, ldapTimeout)
	context := ""

	t.Run("Ldap Connect", func(t *testing.T) {
		t.Logf("Connect '%s' on port %d", LdapAdminUser, port)
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
		t.Logf("load from %s", tnsSource1)

		// read entries from file
		fileTnsEntries, domain, err := dblib.GetTnsnames(tnsSource1, true)
		require.NoErrorf(t, err, "Parsing %s failed: %s", tnsSource1, err)
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

		t.Logf("load from %s", tnsSource2)
		// read entries from file
		fileTnsEntries, domain, err = dblib.GetTnsnames(tnsSource2, true)
		require.NoErrorf(t, err, "Parsing %s failed: %s", tnsSource2, err)
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
		args := []string{
			"ldap",
			"write",
			"--ldap.oraclectx", LdapBaseDn,
			"--ldap.host", server,
			"--ldap.port", fmt.Sprintf("%d", port),
			"--ldap.base", LdapBaseDn,
			"--ldap.binddn", LdapAdminUser,
			"--ldap.bindpassword", LdapAdminPassword,
			"--ldap.tnssource", tnsSource1,
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		require.NoErrorf(t, err, "Command returned error: %s", err)
		t.Logf(out)
		assert.Containsf(t, out, "SUCCESS: ", "Output not as expected")
	})

	t.Run("Read TNS from Ldap with config file and env", func(t *testing.T) {
		tnsAdmin = test.TestData
		filename = tnsAdmin + "/ldap_file_read.ora"
		_ = os.Remove(filename)
		_ = os.Setenv("LDAP_BIND_PASSWORD", LdapAdminPassword)
		args := []string{
			"ldap",
			"read",
			"--ldap.host", server,
			"--ldap.port", fmt.Sprintf("%d", port),
			"--ldap.tnstarget", filename,
			"--config", testConfig,
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		require.NoErrorf(t, err, "Command returned error:%s", err)
		t.Logf(out)
		assert.FileExistsf(t, filename, "Output File not created")
		assert.Containsf(t, out, "SUCCESS: ", "Output not as expected")
	})

	t.Run("Clear TNS Entries from Ldap with config file prompt password", func(t *testing.T) {
		_ = os.Setenv("LDAP_BIND_PASSWORD", "")
		ldapBaseDN = ""
		ldapBindPassword = ""
		r, w, err := os.Pipe()
		require.NoErrorf(t, err, "Pipe failed")
		inputReader = r
		args := []string{
			"ldap",
			"clear",
			"--ldap.host", server,
			"--ldap.port", fmt.Sprintf("%d", port),
			"--config", testConfig,
			"--info",
			"--unit-test",
		}
		// write to Stdin
		_, _ = w.WriteString(fmt.Sprintf("%s\n", LdapAdminPassword))

		out, err = common.CmdRun(RootCmd, args)
		require.NoErrorf(t, err, "Command returned error:%s", err)
		t.Logf(out)
		assert.Containsf(t, out, "SUCCESS: ", "Output not as expected")
		inputReader = os.Stdin
		_ = w.Close()
	})
}
