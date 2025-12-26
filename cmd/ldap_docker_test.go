package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/ldaplib"
	"github.com/tommi2day/tnscli/test"

	"github.com/go-ldap/ldap/v3"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const Ldaprepo = "docker.io/cleanstart/openldap"
const LdaprepoTag = "2.6.10"
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
		Mounts: []string{
			test.TestDir + "/docker/oracle-ldap/certs:/certs:ro",
			// test.TestDir + "/docker/oracle-ldap/schema:/schema:ro",
			test.TestDir + "/docker/oracle-ldap/ldif:/ldif:ro",
			test.TestDir + "/docker/oracle-ldap/etc/slapd.conf:/etc/openldap/slapd.conf:ro",
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
	// wait 15s to init container
	time.Sleep(15 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("LDAP Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = applyLdapConfigs(myhost, myport, test.TestDir+"/docker/oracle-ldap/ldif")
	if err != nil {
		return
	}
	err = nil
	return
}
func applyLdapConfigs(server string, port int, ldifDir string) (err error) {
	lc := ldaplib.NewConfig(server, port, false, false, "cn=config", ldapTimeout)
	err = lc.Connect(LdapConfigUser, LdapConfigPassword)
	if err != nil || lc.Conn == nil {
		err = fmt.Errorf("LDAP Config Connect failed: %v", err)
		return
	}

	pattern := "*.schema"
	// Apply all files matching *.config
	err = lc.ApplyLDIFDir(ldifDir, pattern, false)
	if err != nil {
		return
	}

	// Verify by searching for one of the applied schemas/configs if needed
	// For example, checking if a specific schema DN exists
	schemaBase := "cn=schema,cn=config"
	entries, e := lc.Search(schemaBase, "(cn=*oidbase)", []string{"dn"}, ldap.ScopeWholeSubtree, ldap.DerefInSearching)
	if e != nil || len(entries) == 0 {
		err = fmt.Errorf("Search for schema oidbase failed: %v", e)
		return
	}
	fmt.Printf("Schema Verified: %s exists\n", entries[0].DN)
	pattern = "*.config"
	err = lc.ApplyLDIFDir(ldifDir, pattern, false)
	if err != nil {
		return
	}
	fmt.Println("LDAP Configs applied")

	// apply entries
	la := ldaplib.NewConfig(server, port, false, false, LdapBaseDn, ldapTimeout)
	err = la.Connect(LdapAdminUser, LdapAdminPassword)
	if err != nil || la.Conn == nil {
		err = fmt.Errorf("LDAP Admin Connect failed: %v", err)
		return
	}
	pattern = "*.ldif"
	err = la.ApplyLDIFDir(ldifDir, pattern, false)
	if err != nil {
		return
	}
	fmt.Println("LDAP Entries applied")
	return
}
