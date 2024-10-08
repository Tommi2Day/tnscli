package cmd

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/tommi2day/gomodules/netlib"

	"github.com/tommi2day/gomodules/common"

	"github.com/tommi2day/tnscli/test"

	"github.com/tommi2day/gomodules/dblib"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const racaddr = "myrac.rac.lan"
const racalias = "MYRAC"

var racora = fmt.Sprintf("MYRAC.local=(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%s)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=RACPDB1)))", racaddr, dbPort)

const racinfoini = `
[MYRAC.RAC.LAN]
scan=myrac.rac.lan:1521
vip1=vip1.rac.lan:1521
vip2=vip2.rac.lan:1521
vip3=vip3.rac.lan:1521
`

func TestRACInfo(t *testing.T) {
	var err error
	test.InitTestDirs()
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	err = common.WriteStringToFile(tnsAdminDir+"/racinfo.ini", racinfoini)
	require.NoErrorf(t, err, "Create test racinfo.ini failed")
	racfilename := tnsAdminDir + "/rac.ora"

	err = common.WriteStringToFile(racfilename, racora)
	require.NoErrorf(t, err, "Create test rac.ora failed")
	connectfilename := tnsAdminDir + "/connect2.ora"
	_ = common.WriteStringToFile(connectfilename, xealias+"="+xetest)
	if os.Getenv("SKIP_DNS") != "" {
		t.Skip("Skipping DNS testing in CI environment")
	}
	tnscliDNSContainer, err = prepareDNSContainer()
	require.NoErrorf(t, err, "DNS Server not available")
	require.NotNil(t, tnscliDNSContainer, "Prepare failed")
	defer destroyDNSContainer(tnscliDNSContainer)
	// use DNS from Docker
	dns := netlib.NewResolver(tnscliDNSServer, tnscliDNSPort, true)
	dns.Timeout = 8 * time.Second
	t.Run("Test RacInfo.ini resolution", func(t *testing.T) {
		dns.IPv4Only = true
		dblib.IgnoreDNSLookup = false
		dblib.IPv4Only = true
		dblib.DNSConfig = dns
		addr := dblib.GetRacAdresses(racaddr, path.Join(tnsAdminDir, racinfoFile))
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
	t.Run("CMD FREE Port info IP Addr", func(t *testing.T) {
		out := ""
		args := []string{
			"service",
			"info",
			"ports",
			"--filename", connectfilename,
			"--service", "FREE.local",
			"--info",
			"--nodns",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("Alias FREE.local uses %d addresses", 1)
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
			"--nameserver", fmt.Sprintf("%s:%d", tnscliDNSServer, tnscliDNSPort),
			"--dnstcp",
			"--nodns=false",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "Check should succeed")
		expect := fmt.Sprintf("Alias %s uses %d addresses", racalias, 6)
		assert.Contains(t, out, expect, "Expected Message not found")
	})
}
