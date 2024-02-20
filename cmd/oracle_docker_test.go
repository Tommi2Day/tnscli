package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/tommi2day/tnscli/test"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/tommi2day/gomodules/common"
)

// DBPort is the port of the Oracle DB to access (default:21521)
const DBPort = "21521"
const repo = "docker.io/gvenzl/oracle-free"
const repoTag = "23.3-slim"
const containerTimeout = 600

// SYSTEMUSER is the name of the default DBA user
const SYSTEMUSER = "system"

// SYSTEMSERVICE is the name of the root service
const SYSTEMSERVICE = "FREE"

var containerName string

// prepareContainer create an Oracle Docker Container
func prepareContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_ORACLE") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}
	containerName = os.Getenv("CONTAINER_NAME")
	if containerName == "" {
		containerName = "tnscli-oracledb"
	}
	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + repo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, repoTag)
	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        repoTag,

		Hostname: containerName,
		Name:     containerName,
		Env: []string{
			"ORACLE_PASSWORD=" + DBPASSWORD,
		},
		ExposedPorts: []string{"1521"},
		// need fixed mapping here
		PortBindings: map[docker.Port][]docker.PortBinding{
			"1521": {
				{HostIP: "0.0.0.0", HostPort: DBPort},
			},
		},
		Mounts: []string{
			test.TestDir + "/docker/oracle-db:/container-entrypoint-initdb.d:ro",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting DB docker %s container: %v", containerName, err)
		_ = pool.Purge(container)
		return
	}

	start := time.Now()
	err = WaitForOracle(pool)
	if err != nil {
		_ = pool.Purge(container)
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("DB Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}

// WaitForOracle waits to successfully connect to Oracle
func WaitForOracle(pool *dockertest.Pool) (err error) {
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

	pool.MaxWait = containerTimeout * time.Second
	target = fmt.Sprintf("oracle://%s:%s@%s:%s/%s", SYSTEMUSER, DBPASSWORD, dbhost, DBPort, SYSTEMSERVICE)
	fmt.Printf("Wait to successfully init db with %s (max %ds)...\n", target, containerTimeout)
	start := time.Now()
	if err = pool.Retry(func() error {
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
