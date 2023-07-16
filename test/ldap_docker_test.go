package test

import (
	"fmt"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"

	ldap "github.com/go-ldap/ldap/v3"
	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const Ldaprepo = "docker.io/osixia/openldap"
const LdaprepoTag = "1.5.0"
const LdapcontainerTimeout = 120

var ldapcontainerName string
var ldapContainer *dockertest.Resource

// prepareContainer create an OpenLdap Docker Container
func prepareLdapContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_LDAP") != "" {
		err = fmt.Errorf("skipping LDAP Container in CI environment")
		return
	}
	ldapcontainerName = os.Getenv("LDAP_CONTAINER_NAME")
	if ldapcontainerName == "" {
		ldapcontainerName = "tnscli-ldap"
	}

	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + Ldaprepo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, LdaprepoTag)
	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        LdaprepoTag,
		Env: []string{
			"LDAP_ORGANISATION=" + ldapOrganisation,
			"LDAP_DOMAIN=" + LdapDomain,
			"LDAP_BASE_DN=" + LdapBaseDn,
			"LDAP_ADMIN_PASSWORD=" + LdapAdminPassword,
			"LDAP_CONFIG_PASSWORD=" + LdapConfigPassword,
			"LDAP_TLS_VERIFY_CLIENT=never",
			"LDAP_SEED_INTERNAL_LDIF_PATH=/bootstrap/ldif",
			"LDAP_SEED_INTERNAL_SCHEMA_PATH=/bootstrap/schema",
		},
		Mounts: []string{
			TestDir + "/oracle-ldap/ldif:/bootstrap/ldif:ro",
			TestDir + "/oracle-ldap/schema:/bootstrap/schema:ro",
		},
		Hostname: ldapcontainerName,
		Name:     ldapcontainerName,
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting ldap docker container: %v", err)
		return
	}

	pool.MaxWait = LdapcontainerTimeout * time.Second
	myhost, myport := common.GetContainerHostAndPort(container, "389/tcp")
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
	// wait 5s to init container
	time.Sleep(5 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("LDAP Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
