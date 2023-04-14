package test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/dblib"
)

const tnsnamesora = `
# Test ifile relative
ifile=ifile.ora
DB_T.local=
  (DESCRIPTION=
    (CONNECT_TIMEOUT=15)
    (TRANSPORT_CONNECT_TIMEOUT=3)
    (ADDRESS_LIST=
      (FAILOVER=on)
      (LOAD_BALANCE=on)
      (ADDRESS=
        (PROTOCOL=TCP)
        (HOST=tdb1.ora.local)
        (PORT=1562)
      )
      (ADDRESS=
        (PROTOCOL=TCP)
        (HOST=tdb2.ora.local)
        (PORT=1562)
      )
    )
    (CONNECT_DATA=
      (SERVER=dedicated)
      (SERVICE_NAME=DB_T.local)
    )
  )


DB_V.local =(DESCRIPTION =
	(CONNECT_TIMEOUT=15)
	(RETRY_COUNT=20)
	(RETRY_DELAY=3)
	(TRANSPORT_CONNECT_TIMEOUT=3)
	(ADDRESS_LIST =
		(LOAD_BALANCE=ON)
		(FAILOVER=ON)
		(ADDRESS=(PROTOCOL=TCP)(HOST=vdb1.ora.local)(PORT=1672))
		(ADDRESS=(PROTOCOL=TCP)(HOST=vdb2.ora.local)(PORT=1672))
	)
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = DB_V.local))
)
`

const ifileora = `
XE =(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE-ohne))
)
XE.local =(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)
XE1 =(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE1))
)`
const sqlnetora = `
NAMES.DEFAULT_DOMAIN=local
NAMES.DIRECTORY_PATH=(TNSNAMES,EZCONNECT)
`
const entryCount = 5

var tnsAdminDir = "testdata"

func TestParseTns(t *testing.T) {
	var err error
	Testinit(t)
	//nolint gosec
	err = os.Chdir(TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	//nolint gosec
	err = os.WriteFile(tnsAdminDir+"/sqlnet.ora", []byte(sqlnetora), 0644)
	//nolint gosec
	err = os.WriteFile(tnsAdminDir+"/sqlnet.ora", []byte(sqlnetora), 0644)
	require.NoErrorf(t, err, "Create test sqlnet.ora failed")
	//nolint gosec
	err = os.WriteFile(tnsAdminDir+"/tnsnames.ora", []byte(tnsnamesora), 0644)
	require.NoErrorf(t, err, "Create test tnsnames.ora failed")
	//nolint gosec
	err = os.WriteFile(tnsAdminDir+"/ifile.ora", []byte(ifileora), 0644)
	require.NoErrorf(t, err, "Create test ifile.ora failed")

	filename := tnsAdminDir + "/tnsnames.ora"
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
	t.Run("Count Entries", func(t *testing.T) {
		countEntries := len(tnsEntries)
		expected := entryCount
		actual := countEntries
		assert.Equal(t, expected, actual, "Count not expected")
	})
	t.Run("Check entry", func(t *testing.T) {
		type testTableType struct {
			name    string
			alias   string
			success bool
			service string
		}
		for _, test := range []testTableType{
			{
				name:    "XE-full",
				alias:   "XE.local",
				success: true,
				service: "XE",
			},
			{
				name:    "XE-short",
				alias:   "XE",
				success: true,
				service: "XE",
			},
			{
				name:    "XE1-short-invalid",
				alias:   "XE1",
				success: false,
				service: "",
			},
			{
				name:    "XE+full-invalid",
				alias:   "XE1.local",
				success: false,
				service: "",
			},
			{
				name:    "XE+invalid domain",
				alias:   "XE" + ".xx.xx",
				success: false,
				service: "",
			},
			{
				name:    "novalue",
				alias:   "",
				success: false,
				service: "",
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				e, ok := dblib.GetEntry(test.alias, tnsEntries, domain)
				if test.success {
					assert.True(t, ok, "Alias %s not found", test.alias)
					name := strings.ToUpper(e.Name)
					assert.True(t, strings.HasPrefix(name, strings.ToUpper(test.alias)), "entry not related to given alias %s", test.alias)
					assert.Equalf(t, test.service, e.Service, "entry returned wrong service ('%s' <>'%s)", e.Service, test.service)
				} else {
					assert.False(t, ok, "Alias %s found, but shouldnt be", test.alias)
				}
			})
		}
	})

	alias := "XE"
	t.Run("Check entry value", func(t *testing.T) {
		e, ok := dblib.GetEntry(alias, tnsEntries, domain)
		assert.True(t, ok, "Alias %s not found", alias)
		actualDesc := e.Desc
		expectedDesc := `(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)`
		assert.Equal(t, strings.TrimSpace(expectedDesc), strings.TrimSpace(actualDesc), "Description not expected")
	})
	t.Run("Check Server Entry", func(t *testing.T) {
		e, found := tnsEntries[alias]
		assert.True(t, found, "Alias not found")
		actual := len(e.Servers)
		expected := 1
		assert.Equal(t, expected, actual, "Server Count not expected")
		if actual > 0 {
			server := e.Servers[0]
			assert.NotEmpty(t, server.Host, "Host ist empty")
			assert.NotEmpty(t, server.Port, "Port ist empty")
		}
	})
	var out = ""
	t.Run("CMD List", func(t *testing.T) {
		args := []string{
			"list",
			"--filename", filename,
			"--search", "XE1",
			"--info",
		}
		out, err = cmdTest(args)
		assert.NoErrorf(t, err, "List command should not return an error:%s", err)
		assert.NotEmpty(t, out, "List should not empty")
		assert.Contains(t, out, "found 1 ", "Output should state one entry")
		t.Logf(out)
	})
}
