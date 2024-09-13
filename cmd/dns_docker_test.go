package cmd

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ory/dockertest/v3"

	"github.com/tommi2day/tnscli/test"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/netlib"

	"github.com/ory/dockertest/v3/docker"
)

const (
	tnscliDNSTimeout    = 10
	tnscliNetworkName   = "tnscli-dnsnetwork"
	tnscliNetworkPrefix = "172.25.2"
	tnscliRepoTag       = "9.20"
	tnscliDNSPort       = 9055
	tnscliTestAddr      = racaddr
)

var (
	tnscliDNSContainerName  string
	tnscliDNSContainer      *dockertest.Resource
	tnscliDNSNetwork        *dockertest.Network
	tnscliDNSNetworkCreated = false
	tnscliDNSServer         = common.GetStringEnv("DNS_HOST", "127.0.0.1")
)

// prepareDBlibDNSContainer create a Bind9 Docker Container
func prepareDNSContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_DB_DNS") != "" {
		return nil, fmt.Errorf("skipping DB DNS Container in CI environment")
	}

	tnscliDNSContainerName = getContainerName()
	pool, err := common.GetDockerPool()
	if err != nil {
		return nil, err
	}

	err = setupNetwork(pool)
	if err != nil {
		return nil, err
	}

	container, err = buildAndRunContainer(pool)
	if err != nil {
		return
	}

	time.Sleep(10 * time.Second)

	err = validateContainerIP(container)
	if err != nil {
		return
	}

	err = waitForDNSServer(pool)
	if err != nil {
		return
	}

	err = testDNSResolution()
	return
}

func getContainerName() string {
	name := os.Getenv("DBDNS_CONTAINER_NAME")
	if name == "" {
		name = "tnscli-bind9"
	}
	return name
}

func setupNetwork(pool *dockertest.Pool) error {
	networks, err := pool.NetworksByName(tnscliNetworkName)
	if err != nil || len(networks) == 0 {
		return createNetwork(pool)
	}
	tnscliDNSNetwork = &networks[0]
	return nil
}

func createNetwork(pool *dockertest.Pool) error {
	var err error
	tnscliDNSNetwork, err = pool.CreateNetwork(tnscliNetworkName, func(options *docker.CreateNetworkOptions) {
		options.Name = tnscliNetworkName
		options.CheckDuplicate = true
		options.IPAM = &docker.IPAMOptions{
			Driver: "default",
			Config: []docker.IPAMConfig{{
				Subnet:  tnscliNetworkPrefix + ".0/24",
				Gateway: tnscliNetworkPrefix + ".1",
			}},
		}
		options.EnableIPv6 = false
	})
	if err != nil {
		return fmt.Errorf("could not create Network: %s:%s", tnscliNetworkName, err)
	}
	tnscliDNSNetworkCreated = true
	return nil
}

func buildAndRunContainer(pool *dockertest.Pool) (*dockertest.Resource, error) {
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	fmt.Printf("Try to build and start docker container %s\n", tnscliDNSContainerName)
	buildArgs := []docker.BuildArg{
		{Name: "VENDOR_IMAGE_PREFIX", Value: vendorImagePrefix},
		{Name: "BIND9_VERSION", Value: tnscliRepoTag},
	}

	dockerContextDir := test.TestDir + "/docker/oracle-dns"

	port := fmt.Sprintf("%d", tnscliDNSPort)
	return pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: dockerContextDir,
			Dockerfile: "Dockerfile",
		},
		&dockertest.RunOptions{
			Hostname:     tnscliDNSContainerName,
			Name:         tnscliDNSContainerName,
			Networks:     []*dockertest.Network{tnscliDNSNetwork},
			ExposedPorts: []string{port},
			// need fixed mapping here
			PortBindings: map[docker.Port][]docker.PortBinding{
				"9055/tcp": {
					{HostIP: "0.0.0.0", HostPort: port},
				},
			},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = false
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})
}

func validateContainerIP(container *dockertest.Resource) error {
	ip := container.GetIPInNetwork(tnscliDNSNetwork)

	fmt.Printf("DB DNS Container IP: %s\n", ip)
	return nil
}

// func waitForDNSServer(pool *dockertest.Pool, container *dockertest.Resource) error {
func waitForDNSServer(pool *dockertest.Pool) error {
	pool.MaxWait = tnscliDNSTimeout * time.Second
	start := time.Now()
	err := pool.Retry(func() error {
		c, err := net.Dial("tcp", fmt.Sprintf("%s:%d", tnscliDNSServer, tnscliDNSPort))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
			return err
		}
		_ = c.Close()
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not connect to DB DNS Container: %v", err)
	}

	time.Sleep(10 * time.Second)
	elapsed := time.Since(start)
	fmt.Println("DB DNS Container is ready after ", elapsed.Round(time.Millisecond))
	return nil
}

func testDNSResolution() error {
	dns := netlib.NewResolver(tnscliDNSServer, tnscliDNSPort, true)
	dns.IPv4Only = true
	s := "/udp"
	if dns.TCP {
		s = "/tcp"
	}
	fmt.Printf("resolve on %s:%d%s\n", dns.Nameserver, dns.Port, s)
	ips, err := dns.LookupIP(tnscliTestAddr)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("could not resolve DNS for %s: %v", tnscliTestAddr, err)
	}
	fmt.Printf("Host %s resolved to %s\n", tnscliTestAddr, ips[0])
	return nil
}

func destroyDNSContainer(container *dockertest.Resource) {
	if container != nil {
		common.DestroyDockerContainer(container)
	}

	if tnscliDNSNetworkCreated && tnscliDNSNetwork != nil {
		_ = tnscliDNSNetwork.Close()
	}
}
