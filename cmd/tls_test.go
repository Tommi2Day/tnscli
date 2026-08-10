package cmd

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/dblib"
	"github.com/tommi2day/tnscli/test"
)

// TestWalletConfigWiring verifies tnscli picks up WALLET_LOCATION from
// sqlnet.ora the same way sqlplus resolves it via TNS_ADMIN, without needing
// a real TCPS listener; the actual wallet/TCPS connect behavior is covered by
// dblib's own docker test against a real orapki wallet and TCPS listener.
func TestWalletConfigWiring(t *testing.T) {
	test.InitTestDirs()
	defer func() { dblib.TNSSSLconfig = dblib.TNSSSL{} }()

	tlsDir := path.Join(test.TestData, "tlswiring")
	walletDir := path.Join(tlsDir, "wallet")
	require.NoErrorf(t, os.MkdirAll(walletDir, 0750), "create wallet dir failed")

	sqlnetContent := fmt.Sprintf("WALLET_LOCATION=(SOURCE=(METHOD=FILE)(METHOD_DATA=(DIRECTORY=\"%s\")))\nSSL_SERVER_DN_MATCH=Yes\n", walletDir)
	require.NoErrorf(t, common.WriteStringToFile(path.Join(tlsDir, "sqlnet.ora"), sqlnetContent), "write sqlnet.ora failed")

	alias := "TLSWIRING.local"
	desc := "(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCPS)(HOST=127.0.0.1)(PORT=2484)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=FREEPDB1)))"
	tnsFilename := path.Join(tlsDir, "connect.ora")
	require.NoErrorf(t, common.WriteStringToFile(tnsFilename, alias+"="+desc), "write tnsnames failed")

	t.Run("CMD JDBC info includes wallet location from sqlnet.ora", func(t *testing.T) {
		args := []string{
			cmdService,
			cmdInfo,
			cmdJdbc,
			flagFilename, tnsFilename,
			flagService, alias,
			flagInfo,
			flagUnitTest,
		}
		out, err := common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, err, "jdbc info should succeed")
		assert.Contains(t, out, "WALLET_LOCATION="+walletDir, "expected wallet location not found in jdbc url")
	})
}
