package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ory/dockertest/v4"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/dblib"
	"github.com/tommi2day/tnscli/test"

	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TCPS testing needs a real Oracle wallet (orapki) which requires the "-full"
// image; the "-slim" image used by the other Oracle tests has no Java and
// therefore no working orapki/mkstore. The "-faststart" variant restores from
// a pre-built snapshot instead of running dbca from scratch, so it starts
// fast enough to fit go test's default 10-minute binary-wide deadline.
const (
	tcpsRepoTag = "23.26.2-full-faststart"
	tcpsDBPort  = "21525"
	tcpsPort    = "21526"
	// keep this safely under go test's default 10-minute (600s) overall
	// deadline (not equal to it - that races the test binary's own alarm) so
	// a genuinely stuck container fails with a clear message here instead of
	// an opaque timeout panic that doesn't print the last error
	tcpsContainerTimeout = 480
)

var tcpsContainerName string

// prepareTCPSContainer starts an Oracle "-full" container whose
// container-entrypoint-initdb.d scripts (test/docker/oracle-db-tcps) create a
// real orapki auto-login wallet and configure a real TCPS listener with it,
// then copy that wallet into walletDir (bind-mounted at /client-wallet) so
// tnscli can use it as WALLET_LOCATION exactly like a real client would.
func prepareTCPSContainer(walletDir string) (container dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_ORACLE") != "" || os.Getenv("SKIP_ORACLE_TCPS") != "" {
		err = fmt.Errorf("skipping ORACLE TCPS Container in CI environment")
		return
	}
	tcpsContainerName = os.Getenv("TCPS_CONTAINER_NAME")
	if tcpsContainerName == "" {
		tcpsContainerName = "tnscli-oracledb-tcps"
	}
	var pool dockertest.ClosablePool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + dbRepo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, tcpsRepoTag)
	container, err = pool.Run(context.Background(), repoString,
		dockertest.WithTag(tcpsRepoTag),
		dockertest.WithHostname(tcpsContainerName),
		dockertest.WithName(tcpsContainerName),
		dockertest.WithEnv([]string{
			"ORACLE_PASSWORD=" + dbPassword,
		}),
		dockertest.WithPortBindings(network.PortMap{
			network.MustParsePort("1521/tcp"): {
				{HostIP: netip.IPv4Unspecified(), HostPort: tcpsDBPort},
			},
			network.MustParsePort("2484/tcp"): {
				{HostIP: netip.IPv4Unspecified(), HostPort: tcpsPort},
			},
		}),
		dockertest.WithMounts([]string{
			test.TestDir + "/docker/oracle-db-tcps:/container-entrypoint-initdb.d:ro",
			walletDir + ":/client-wallet",
		}),
		dockertest.WithHostConfig(func(config *dockercontainer.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = dockercontainer.RestartPolicy{Name: restartPolicyNo}
		}),
	)
	if err != nil {
		err = fmt.Errorf("error starting TCPS DB docker %s container: %v", tcpsContainerName, err)
		if container != nil {
			common.DestroyDockerContainer(container)
		}
		return
	}

	tcpsTarget := fmt.Sprintf("oracle://%s:%s@%s:%s/%s", dbSystemUser, dbPassword, dbHost, tcpsDBPort, dbSystemService)
	fmt.Printf("Wait to successfully init TCPS db with %s (max %ds)...\n", tcpsTarget, tcpsContainerTimeout)
	start := time.Now()
	lastLog := start
	if err = pool.Retry(context.Background(), tcpsContainerTimeout*time.Second, func() error {
		attemptErr := tcpsDBReady(tcpsTarget)
		// this can silently loop for minutes; log the last error periodically
		// so a stuck run is diagnosable instead of a bare test-timeout panic
		if attemptErr != nil && time.Since(lastLog) > 30*time.Second {
			fmt.Printf("TCPS DB still not ready after %s: %v\n", time.Since(start).Round(time.Second), attemptErr)
			lastLog = time.Now()
		}
		return attemptErr
	}); err != nil {
		fmt.Printf("TCPS DB Container not ready: %s\n", err)
		diagnosePortBinding(container, "1521")
		printContainerLogs(container)
		common.DestroyDockerContainer(container)
		return
	}
	fmt.Printf("TCPS DB Ready is available after %s\n", time.Since(start).Round(time.Millisecond))
	return
}

// tcpsDBReady checks the DB is reachable and every init script, including the
// TCPS wallet/listener setup, has finished (marked by the init_done table)
func tcpsDBReady(target string) error {
	db, err := sql.Open("oracle", target)
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
	}
	row := db.QueryRow("select count(*) from init_done")
	var count int64
	return row.Scan(&count)
}

// TestOracleTCPSConnect verifies tnscli connects through a real TCPS listener
// using a real orapki auto-login wallet, the same way sqlplus would: a TNS
// entry with PROTOCOL=TCPS plus WALLET_LOCATION in sqlnet.ora, no password
// needed since the wallet is auto-login (matches production practice).
func TestOracleTCPSConnect(t *testing.T) {
	if os.Getenv("SKIP_ORACLE") != "" || os.Getenv("SKIP_ORACLE_TCPS") != "" {
		t.Skip("Skipping ORACLE TCPS testing in CI environment")
	}
	test.InitTestDirs()

	// dbUser/dbPass/walletPassword and dblib.TNSSSLconfig are shared package
	// state bound to persistent flags / a package var; restore them so tests
	// that run after this one in the same process aren't affected.
	defer func() {
		dbUser = ""
		dbPass = ""
		walletPassword = ""
		dblib.TNSSSLconfig = dblib.TNSSSL{}
	}()

	tcpsDir := path.Join(test.TestData, "tcps")
	walletDir := path.Join(tcpsDir, "wallet")
	// clean up any stale wallet from a previous failed run before recreating
	_ = os.RemoveAll(walletDir)
	require.NoErrorf(t, os.MkdirAll(walletDir, 0777), "create wallet dir failed")
	// MkdirAll's mode is masked by umask; chmod explicitly so the
	// container's oracle user (a different uid) can write the wallet here
	require.NoErrorf(t, os.Chmod(walletDir, 0777), "chmod wallet dir failed")

	tcpsAlias := "TCPSTEST.local"
	tcpsDesc := fmt.Sprintf("(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCPS)(HOST=%s)(PORT=%s)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=FREEPDB1)))", dbHost, tcpsPort)

	// no sqlnet.ora here: used to prove the check fails without a wallet
	noWalletDir := path.Join(tcpsDir, "nowallet")
	require.NoErrorf(t, os.MkdirAll(noWalletDir, 0750), "create nowallet dir failed")
	noWalletFilename := path.Join(noWalletDir, "connect.ora")
	require.NoErrorf(t, common.WriteStringToFile(noWalletFilename, tcpsAlias+"="+tcpsDesc), "write tnsnames failed")

	// sqlnet.ora with WALLET_LOCATION pointing at the container's wallet
	sqlnetContent := fmt.Sprintf("WALLET_LOCATION=(SOURCE=(METHOD=FILE)(METHOD_DATA=(DIRECTORY=\"%s\")))\n", walletDir)
	require.NoErrorf(t, common.WriteStringToFile(path.Join(tcpsDir, "sqlnet.ora"), sqlnetContent), "write sqlnet.ora failed")
	tnsFilename := path.Join(tcpsDir, "tcps_connect.ora")
	require.NoErrorf(t, common.WriteStringToFile(tnsFilename, tcpsAlias+"="+tcpsDesc), "write tnsnames failed")

	oracleContainer, err := prepareTCPSContainer(walletDir)
	require.NoErrorf(t, err, "prepare TCPS Oracle Container failed")
	require.NotNil(t, oracleContainer, "Prepare failed")
	defer common.DestroyDockerContainer(oracleContainer)

	t.Run("CMD Check TCPS without wallet fails", func(t *testing.T) {
		args := []string{
			cmdService,
			cmdCheck,
			flagFilename, noWalletFilename,
			flagService, tcpsAlias,
			flagUser, dbSystemUser,
			flagPassword, dbPassword,
			flagInfo,
			flagUnitTest,
		}
		out, cmdErr := common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.Errorf(t, cmdErr, "Check should fail without a wallet")
	})

	t.Run("CMD Check TCPS with wallet succeeds", func(t *testing.T) {
		args := []string{
			cmdService,
			cmdCheck,
			flagFilename, tnsFilename,
			flagService, tcpsAlias,
			flagUser, dbSystemUser,
			flagPassword, dbPassword,
			flagInfo,
			flagUnitTest,
		}
		out, cmdErr := common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, cmdErr, "TCPS check with wallet should succeed")
		expect := fmt.Sprintf("service %s connected", tcpsAlias)
		assert.Contains(t, out, expect, "Expected Message not found")
	})

	t.Run("CMD JDBC info includes wallet location", func(t *testing.T) {
		args := []string{
			cmdService,
			cmdInfo,
			cmdJdbc,
			flagFilename, tnsFilename,
			flagService, tcpsAlias,
			flagInfo,
			flagUnitTest,
		}
		out, cmdErr := common.CmdRun(RootCmd, args)
		t.Log(out)
		assert.NoErrorf(t, cmdErr, "jdbc info should succeed")
		assert.Contains(t, out, "WALLET_LOCATION="+walletDir, "expected wallet location not found in jdbc url")
	})
}
