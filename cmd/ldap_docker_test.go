package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/tnscli/test"

	"github.com/go-ldap/ldap/v3"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const Ldaprepo = "docker.io/bitnami/openldap"
const LdaprepoTag = "2.6.8"
const LdapcontainerTimeout = 120

var TnsLdapcontainerName string
var TnsLdapContainer *dockertest.Resource

// prepareContainer create an OpenLdap Docker Container
func prepareTnsLdapContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_LDAP") != "" {
		err = fmt.Errorf("skipping LDAP Container in CI environment")
		return
	}
	TnsLdapcontainerName = os.Getenv("LDAP_CONTAINER_NAME")
	if TnsLdapcontainerName == "" {
		TnsLdapcontainerName = "tnscli-ldap"
	}
	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil || pool == nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + Ldaprepo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, LdaprepoTag)
	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        LdaprepoTag,
		Env: []string{
			"LDAP_PORT_NUMBER=1389",
			"BITNAMI_DEBUG=true",
			"LDAP_ROOT=" + LdapBaseDn,
			"LDAP_ADMIN_USERNAME=admin",
			"LDAP_ADMIN_PASSWORD=" + LdapAdminPassword,
			"LDAP_CONFIG_ADMIN_ENABLED=yes",
			"LDAP_CONFIG_ADMIN_USERNAME=config",
			"LDAP_CONFIG_ADMIN_PASSWORD=" + LdapConfigPassword,
			"LDAP_SKIP_DEFAULT_TREE=yes",
			"LDAP_CUSTOM_LDIF_DIR=/bootstrap/ldif",
			"LDAP_CUSTOM_SCHEMA_DIR=/bootstrap/schema",
			"LDAP_ADD_SCHEMAS=yes",
			"LDAP_EXTRA_SCHEMAS=cosine,inetorgperson,nis",
			"LDAP_ALLOW_ANON_BINDING=yes",
		},
		Mounts: []string{
			test.TestDir + "/docker/oracle-ldap/schema:/bootstrap/schema:ro",
			test.TestDir + "/docker/oracle-ldap/entrypoint:/docker-entrypoint-initdb.d",
			test.TestDir + "/docker/oracle-ldap/ldif:/bootstrap/ldif:ro",
		},
		Hostname: TnsLdapcontainerName,
		Name:     TnsLdapcontainerName,
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil || container == nil {
		err = fmt.Errorf("error starting ldap docker container: %v", err)
		return
	}

	pool.MaxWait = LdapcontainerTimeout * time.Second
	myhost, myport := common.GetContainerHostAndPort(container, "1389/tcp")
	dialURL := fmt.Sprintf("ldap://%s:%d", myhost, myport)
	fmt.Printf("Wait to successfully connect to Ldap with %s (max %ds)...\n", dialURL, LdapcontainerTimeout)
	start := time.Now()
	var l *ldap.Conn
	if err = pool.Retry(func() error {
		l, err = ldap.DialURL(dialURL)
		return err
	}); err != nil {
		fmt.Printf("Could not connect to LDAP Container: %s", err)
		return
	}
	_ = l.Close()
	// wait 15s to init container
	time.Sleep(15 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("LDAP Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
