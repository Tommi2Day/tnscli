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

var xetest = fmt.Sprintf("(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%s)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=FREEPDB1)))", dbHost, dbPort)

const xealias = "FREE.local"

func TestOracleConnect(t *testing.T) {
	if os.Getenv("SKIP_ORACLE") != "" {
		t.Skip("Skipping ORACLE testing in CI environment")
	}

	const toalias = "TOTEST.local"
	const totest = "(DESCRIPTION=((TRANSPORT_CONNECT_TIMEOUT=3)(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=totest)(PORT=1521)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=totest.local)))"
	tnsFilename := tnsAdminDir + "/connect.ora"
	_ = common.WriteStringToFile(tnsFilename, xealias+"="+xetest+"\n\n"+toalias+"="+totest)
	t.Logf("load from %s", tnsFilename)
	tnsEntries, domain, err := dblib.GetTnsnames(tnsFilename, true)
	t.Logf("Default Domain: '%s'", domain)

	t.Run("Parse TNSNames.ora", func(t *testing.T) {
		require.NoErrorf(t, err, "Parsing %s failed: %s", tnsFilename, err)
	})
	if err != nil {
		t.Logf("load returned error: %s ", err)
		return
	}

	e, found := dblib.GetEntry(xealias, tnsEntries, domain)
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
			"--filename", tnsFilename,
			"--service", xealias,
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("service %s connected", xealias)
		assert.Contains(t, out, expect, "Expected Message not found")
		assert.Contains(t, out, "Connect OK, but Login error", "Expected Login Error not found")
	})
	t.Run("CMD all Check with dummy", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"check",
			"--filename", tnsFilename,
			"--all",
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.Errorf(t, err, "Check should fail")
		expect := "2 entries checked, 1 ok, 1 failed"
		assert.Contains(t, out, expect, "Expected Message not found")
		all = false
	})
	t.Run("CMD Check with real user", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"check",
			"--filename", tnsFilename,
			"--service", xealias,
			"--user", dbSystemUser,
			"--password", dbPassword,
			"--timeout", fmt.Sprintf("%d", dbTimeout),
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("service %s connected", xealias)
		assert.Contains(t, out, expect, "Expected Message not found")
	})
	t.Run("CMD false Check", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"check",
			"--filename", tnsFilename,
			"--service", "dummy",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.Errorf(t, err, "Check should fail")
		assert.Contains(t, out, "Error: alias dummy not found", "Expected Message not found")
	})
	t.Run("CMD DBHOST Query", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"check",
			"--filename", tnsFilename,
			"--service", xealias,
			"--dbhost",
			"--user", dbSystemUser,
			"--password", dbPassword,
			"--timeout", fmt.Sprintf("%d", dbTimeout),
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("service %s connected", xealias)
		assert.Contains(t, out, expect, "Expected connect Message not found")
		expect = "Query returned"
		assert.Contains(t, out, expect, "Expected Query Message not found")
	})
	t.Run("CMD FREE Port Info", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"info",
			"ports",
			"--filename", tnsFilename,
			"--service", xealias,
			"--info",
			"--nodns",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("Alias %s uses", xealias)
		assert.Contains(t, out, expect, "Expected Message not found")
	})
	t.Run("CMD JDBC info", func(t *testing.T) {
		const jdbcprefix = "jdbc:oracle:thin:@"
		t.Run("CMD JDBC info normal", func(t *testing.T) {
			out := ""
			args := []string{
				"service",
				"info",
				"jdbc",
				"--filename", tnsFilename,
				"--service", xealias,
				"--info",
				"--unit-test",
			}
			out, err = common.CmdRun(RootCmd, args)
			t.Log(out)
			assert.NoErrorf(t, err, "Check should succeed")
			expect := jdbcprefix + xetest
			assert.Contains(t, out, expect, "Expected Message not found")
		})

		t.Run("CMD JDBC Timeout replaced", func(t *testing.T) {
			out := ""
			args := []string{
				"service",
				"info",
				"jdbc",
				"--filename", tnsFilename,
				"--service", toalias,
				"--info",
				"--unit-test",
			}
			out, err = common.CmdRun(RootCmd, args)
			t.Log(out)
			assert.NoErrorf(t, err, "Check should succeed")
			expect := jdbcprefix + totest
			expect = strings.ReplaceAll(expect, "TIMEOUT=3)", "TIMEOUT=3000)")
			assert.Contains(t, out, expect, "Expected Message not found")
		})
		t.Run("CMD JDBC Timeout not replaced", func(t *testing.T) {
			out := ""
			args := []string{
				"service",
				"info",
				"jdbc",
				"--filename", tnsFilename,
				"--service", toalias,
				"--noModifyTransportConnectTimeout",
				"--info",
				"--unit-test",
			}
			out, err = common.CmdRun(RootCmd, args)
			t.Log(out)
			assert.NoErrorf(t, err, "Check should succeed")
			expect := jdbcprefix + totest
			assert.Contains(t, out, expect, "Expected Message not found")
		})
	})
	t.Run("CMD TNS info", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"info",
			"tns",
			"--filename", tnsFilename,
			"--service", xealias,
			"--info",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := strings.ToUpper(xealias)
		assert.Contains(t, strings.ToUpper(out), expect, "Expected Message '%s' not found", expect)
	})

	t.Run("CMD Portcheck", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"portcheck",
			"--filename", tnsFilename,
			"--service", xealias,
			"--info",
			"--nodns",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := dbHost
		assert.Contains(t, out, expect, "Expected Message not found")
		assert.Contains(t, out, "OPEN", "Port should be open")
	})
	t.Run("CMD Portcheck Error", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"portcheck",
			"--filename", tnsFilename,
			"--service", toalias,
			"--info",
			"--nodns",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		assert.Contains(t, out, "PROBLEM", "Port result should be PROBLEM")
	})
}
