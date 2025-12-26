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

const (
	dbPort             = "21521"
	dbRepo             = "docker.io/gvenzl/oracle-free"
	dbRepoTag          = "23.26.0-slim"
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
func prepareContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_ORACLE") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}
	dbContainerName = os.Getenv("DB_CONTAINER_NAME")
	if dbContainerName == "" {
		dbContainerName = "tnscli-oracledb"
	}
	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + dbRepo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, dbRepoTag)
	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        dbRepoTag,

		Hostname: dbContainerName,
		Name:     dbContainerName,
		Env: []string{
			"ORACLE_PASSWORD=" + dbPassword,
		},
		ExposedPorts: []string{"1521"},
		// need fixed mapping here
		PortBindings: map[docker.Port][]docker.PortBinding{
			"1521": {
				{HostIP: "0.0.0.0", HostPort: dbPort},
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
		err = fmt.Errorf("error starting DB docker %s container: %v", dbContainerName, err)
		if container != nil {
			_ = pool.Purge(container)
		}
		return
	}
	err = WaitForOracle(pool)
	if err != nil {
		_ = pool.Purge(container)
		return
	}
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

	pool.MaxWait = dbContainerTimeout * time.Second
	target = fmt.Sprintf("oracle://%s:%s@%s:%s/%s", dbSystemUser, dbPassword, dbHost, dbPort, dbSystemService)
	fmt.Printf("Wait to successfully init db with %s (max %ds)...\n", target, dbContainerTimeout)
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
