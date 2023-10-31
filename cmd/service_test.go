package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/dblib"
)

const DBUSER = "system"
const DBPASSWORD = "XE-manager21"
const TIMEOUT = 5

var dbhost = common.GetEnv("DB_HOST", "127.0.0.1")
var connectora = fmt.Sprintf("XE.local=(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%s)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=XEPDB1)))", dbhost, port)
var target string

func TestOracleConnect(t *testing.T) {
	if os.Getenv("SKIP_ORACLE") != "" {
		t.Skip("Skipping ORACLE testing in CI environment")
	}
	const alias = "XE.local"
	// t.Skip()
	filename := tnsAdminDir + "/connect.ora"
	//_ = os.Chdir(tnsAdminDir)
	//nolint gosec
	_ = os.WriteFile(filename, []byte(connectora), 0644)

	t.Logf("load from %s", filename)
	tnsEntries, domain, err := dblib.GetTnsnames(filename, true)
	t.Logf("Default Domain: '%s'", domain)
	t.Run("Parse TNSNames.ora", func(t *testing.T) {
		require.NoErrorf(t, err, "Parsing %s failed: %s", filename, err)
	})
	if err != nil {
		t.Logf("load returned error: %s ", err)
		return
	}

	e, found := dblib.GetEntry(alias, tnsEntries, domain)
	require.True(t, found, "Alias not found")
	desc := common.RemoveSpace(e.Desc)
	t.Logf("Desc:%s", desc)
	dbContainer, err := prepareContainer()
	require.NoErrorf(t, err, "prepare Oracle Container failed")
	require.NotNil(t, dbContainer, "Prepare failed")
	defer common.DestroyDockerContainer(dbContainer)

	t.Run("Direct connect", func(t *testing.T) {
		var db *sql.DB
		t.Logf("connect to %s\n", target)
		db, err = sql.Open("oracle", target)
		assert.NoErrorf(t, err, "Open failed: %s", err)
		assert.IsType(t, &sql.DB{}, db, "Returned wrong type")
		err = db.Ping()
		assert.NoErrorf(t, err, "Connect failed: %s", err)
	})
	t.Run("CMD Check with dummy", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"check",
			"--filename", filename,
			"--service", alias,
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("service %s connected", alias)
		assert.Contains(t, out, expect, "Expected Message not found")
		assert.Contains(t, out, "Connect OK, but Login error", "Expected Login Error not found")
	})
	t.Run("CMD Check with real user", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"check",
			"--filename", filename,
			"--service", alias,
			"--user", DBUSER,
			"--password", DBPASSWORD,
			"--timeout", fmt.Sprintf("%d", TIMEOUT),
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("service %s connected", alias)
		assert.Contains(t, out, expect, "Expected Message not found")
	})
	t.Run("CMD false Check", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"check",
			"--filename", filename,
			"--service", "dummy",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.Errorf(t, err, "Check should fail")
		assert.Contains(t, out, "Error: alias dummy not found", "Expected Message not found")
	})
	t.Run("CMD DBHOST Query", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"check",
			"--filename", filename,
			"--service", alias,
			"--dbhost",
			"--user", DBUSER,
			"--password", DBPASSWORD,
			"--timeout", fmt.Sprintf("%d", TIMEOUT),
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("service %s connected", alias)
		assert.Contains(t, out, expect, "Expected connect Message not found")
		expect = "Query returned"
		assert.Contains(t, out, expect, "Expected Query Message not found")
	})
	t.Run("CMD XE Port Info", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"info",
			"ports",
			"--filename", filename,
			"--service", alias,
			"--info",
			"--nodns",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("Alias %s uses", alias)
		assert.Contains(t, out, expect, "Expected Message not found")
	})

	t.Run("CMD JDBC info", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"info",
			"jdbc",
			"--filename", filename,
			"--service", alias,
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := "jdbc:oracle:thin:@("
		assert.Contains(t, out, expect, "Expected Message not found")
	})
	t.Run("CMD TNS info", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"info",
			"tns",
			"--filename", filename,
			"--service", alias,
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := strings.ToUpper(alias)
		assert.Contains(t, strings.ToUpper(out), expect, "Expected Message '%s' not found", expect)
	})

	t.Run("CMD Portcheck", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"portcheck",
			"--filename", filename,
			"--service", alias,
			"--info",
			"--nodns",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := dbhost
		assert.Contains(t, out, expect, "Expected Message not found")
		assert.Contains(t, out, "OPEN", "Port should be open")
	})
}
