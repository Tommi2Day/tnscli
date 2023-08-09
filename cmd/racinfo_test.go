package cmd

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tommi2day/gomodules/common"

	"github.com/tommi2day/tnscli/test"

	"github.com/tommi2day/gomodules/dblib"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const racaddr = "myrac.rac.lan"
const racalias = "MYRAC"

var racora = fmt.Sprintf("MYRAC.local=(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%s)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=RACPDB1)))", racaddr, port)

const racinfoini = `
[MYRAC.RAC.LAN]
scan=myrac.rac.lan:1521
vip1=vip1.rac.lan:1521
vip2=vip2.rac.lan:1521
vip3=vip3.rac.lan:1521
`

func TestRACInfo(t *testing.T) {
	var err error
	// t.Skip()

	test.Testinit(t)
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	//nolint gosec
	err = os.WriteFile(tnsAdminDir+"/racinfo.ini", []byte(racinfoini), 0644)
	require.NoErrorf(t, err, "Create test racinfo.ini failed")
	racfilename := tnsAdminDir + "/rac.ora"
	//nolint gosec
	err = os.WriteFile(racfilename, []byte(racora), 0644)
	require.NoErrorf(t, err, "Create test rac.ora failed")
	connectfilename := tnsAdminDir + "/connect2.ora"
	//nolint gosec
	_ = os.WriteFile(connectfilename, []byte(connectora), 0644)
	if os.Getenv("SKIP_DNS") != "" {
		t.Skip("Skipping DNS testing in CI environment")
	}
	dnsContainer, err = prepareDNSContainer()
	require.NoErrorf(t, err, "DNS Server not available")
	require.NotNil(t, dnsContainer, "Prepare failed")
	defer destroyDNSContainer(dnsContainer)
	// use DNS from Docker
	dblib.Resolver = dblib.SetResolver(dnsserver, dnsport, true)
	dblib.NameserverTimeout = 8 * time.Second
	t.Run("Test RacInfo.ini resolution", func(t *testing.T) {
		dblib.IgnoreDNSLookup = false
		dblib.IPv4Only = true
		addr := dblib.GetRacAdresses(racaddr, tnsAdminDir+"/racinfo.ini")
		assert.Equal(t, 6, len(addr), "Count not expected")
		t.Logf("Addresses: %v", addr)
	})
	t.Run("Test Rac SRV resolution", func(t *testing.T) {
		dblib.IPv4Only = true
		dblib.IgnoreDNSLookup = false
		addr := dblib.GetRacAdresses(racaddr, "")
		assert.Equal(t, 6, len(addr), "Count not expected")
		t.Logf("Addresses: %v", addr)
	})
	t.Run("Test resolution with IP", func(t *testing.T) {
		dblib.IPv4Only = true
		dblib.IgnoreDNSLookup = false
		addr := dblib.GetRacAdresses("127.0.0.1", "")
		assert.Equal(t, 0, len(addr), "Count not expected")
		t.Logf("Addresses: %v", addr)
	})
	t.Run("CMD XE Port info IP Addr", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"info",
			"ports",
			"--filename", connectfilename,
			"--service", "XE.local",
			"--info",
			"--nodns",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("Alias XE.local uses %d addresses", 1)
		assert.Contains(t, out, expect, "Expected Message not found")
	})
	t.Run("CMD Port info", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"info",
			"ports",
			"--filename", racfilename,
			"--service", racalias,
			"--info",
			"--nameserver", fmt.Sprintf("%s:%d", dnsserver, dnsport),
			"--dnstcp",
			"--nodns=false",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Logf(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("Alias %s uses %d addresses", racalias, 6)
		assert.Contains(t, out, expect, "Expected Message not found")
	})
}
