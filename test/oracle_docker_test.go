package test

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/tommi2day/gomodules/common"
)

const port = "21521"
const repo = "docker.io/gvenzl/oracle-xe"
const repoTag = "21.3.0-slim"
const containerTimeout = 600

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
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting ldap docker container: %v", err)
		return
	}

	pool.MaxWait = containerTimeout * time.Second
	target = fmt.Sprintf("oracle://%s:%s@%s:%s/xepdb1", "system", DBPASSWORD, dbhost, port)
	fmt.Printf("Wait to successfully connect to db with %s (max %ds)...\n", target, containerTimeout)
	start := time.Now()
	if err = pool.Retry(func() error {
		var err error
		var db *sql.DB
		db, err = sql.Open("oracle", target)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		fmt.Printf("Could not connect to DB Container: %s", err)
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("DB Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
