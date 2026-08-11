package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"
	"os"
	"time"

	"github.com/ory/dockertest/v4"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/tnscli/test"

	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
)

const (
	dbPort             = "21521"
	dbRepo             = "docker.io/gvenzl/oracle-free"
	dbRepoTag          = "23.26.2-slim"
	dbContainerTimeout = 600
	dbTimeout          = 5

	//nolint gosec
	dbPassword      = "XE-Manager23"
	dbSystemUser    = "system"
	dbSystemService = "FREE"
)

var (
	dbContainerName string
	dbHost          = common.GetEnv("DB_HOST", "127.0.0.1")
	target          string
)

// prepareContainer create an Oracle Docker Container
func prepareContainer() (container dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_ORACLE") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}
	dbContainerName = os.Getenv("DB_CONTAINER_NAME")
	if dbContainerName == "" {
		dbContainerName = "tnscli-oracledb"
	}
	var pool dockertest.ClosablePool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + dbRepo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, dbRepoTag)
	container, err = pool.Run(context.Background(), repoString,
		dockertest.WithTag(dbRepoTag),
		dockertest.WithHostname(dbContainerName),
		dockertest.WithName(dbContainerName),
		dockertest.WithEnv([]string{
			"ORACLE_PASSWORD=" + dbPassword,
		}),
		// need fixed mapping here
		dockertest.WithPortBindings(network.PortMap{
			network.MustParsePort("1521/tcp"): {
				{HostIP: netip.IPv4Unspecified(), HostPort: dbPort},
			},
		}),
		dockertest.WithMounts([]string{
			test.TestDir + "/docker/oracle-db:/container-entrypoint-initdb.d:ro",
		}),
		dockertest.WithHostConfig(func(config *dockercontainer.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = dockercontainer.RestartPolicy{Name: restartPolicyNo}
		}),
	)

	if err != nil {
		err = fmt.Errorf("error starting DB docker %s container: %v", dbContainerName, err)
		if container != nil {
			common.DestroyDockerContainer(container)
		}
		return
	}
	err = WaitForOracle(pool)
	if err != nil {
		printContainerLogs(container)
		common.DestroyDockerContainer(container)
		return
	}
	return
}

// printContainerLogs prints a container's stdout/stderr so a container that
// never becomes reachable (e.g. crashes right after start) is diagnosable
// from CI output instead of surfacing only as an opaque connection-refused
// or timeout error
func printContainerLogs(container dockertest.ClosableResource) {
	if container == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stdout, stderr, err := container.Logs(ctx)
	if err != nil {
		fmt.Printf("could not fetch container logs: %v\n", err)
		return
	}
	fmt.Printf("--- container stdout ---\n%s\n--- container stderr ---\n%s\n--- end container logs ---\n", stdout, stderr)
}

// WaitForOracle waits to successfully connect to Oracle
func WaitForOracle(pool dockertest.Pool) (err error) {
	if os.Getenv("SKIP_ORACLE") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}

	if pool == nil {
		pool, err = common.GetDockerPool()
		if err != nil {
			return
		}
	}

	target = fmt.Sprintf("oracle://%s:%s@%s:%s/%s", dbSystemUser, dbPassword, dbHost, dbPort, dbSystemService)
	fmt.Printf("Wait to successfully init db with %s (max %ds)...\n", target, dbContainerTimeout)
	start := time.Now()
	if err = pool.Retry(context.Background(), dbContainerTimeout*time.Second, func() error {
		var err error
		var db *sql.DB
		db, err = sql.Open("oracle", target)
		if err != nil {
			// cannot open connection
			return err
		}
		err = db.Ping()
		if err != nil {
			// db not answering
			return err
		}
		// check if init_done table exists, then we are ready
		checkSQL := "select count(*) from init_done"
		row := db.QueryRow(checkSQL)
		var count int64
		err = row.Scan(&count)
		if err != nil {
			// query failed, final init table not there
			return err
		}
		return nil
	}); err != nil {
		fmt.Printf("DB Container not ready: %s", err)
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("DB Ready is available after %s\n", elapsed.Round(time.Millisecond))
	// wait for init scripts finished
	err = nil
	return
}
