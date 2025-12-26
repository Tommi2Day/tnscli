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
	tnscliTestAddr      = racaddr
)

var (
	tnscliDNSContainerName  string
	tnscliDNSContainer      *dockertest.Resource
	tnscliDNSNetwork        *dockertest.Network
	tnscliDNSNetworkCreated = false
	tnscliDNSServer         = common.GetStringEnv("DNS_HOST", "")
	tnscliDNSPort           = 9055
)

// prepareDBlibDNSContainer create a Bind9 Docker Container
func prepareDNSContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_DNS") != "" {
		return nil, fmt.Errorf("skipping DB DNS Container in CI environment")
	}

	fmt.Printf("Try to prepare DB DNS Container on %s\n", tnscliDNSServer)
	tnscliDNSContainerName = getContainerName()
	// use versioned docker pool because of api error client version to old
	pool, err := common.GetVersionedDockerPool("")
	if err != nil {
		return nil, fmt.Errorf("pool: %s", err)
	}

	err = setupNetwork(pool)
	if err != nil {
		return nil, fmt.Errorf("docker network: %s", err)
	}

	container, err = buildAndRunContainer(pool)
	if err != nil {
		return container, fmt.Errorf("docker container: %s", err)
	}
	_ = container.Expire(120)
	time.Sleep(10 * time.Second)
	ip := validateContainerIP(container)
	if ip == "" {
		err = fmt.Errorf("could not get IP for Container")
		return
	}
	out, _, e := common.ExecDockerCmd(container, []string{"/usr/bin/ss", "-anl"})
	fmt.Printf("cmd out:%s\n", out)
	if e != nil {
		fmt.Printf("cmd errror: %s", e)
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
			Hostname: tnscliDNSContainerName,
			Name:     tnscliDNSContainerName,
			Networks: []*dockertest.Network{tnscliDNSNetwork},
			// need fixed mapping here
			PortBindings: map[docker.Port][]docker.PortBinding{
				"9055/tcp": {
					{HostIP: "0.0.0.0", HostPort: port},
				},
			},
			CapAdd: []string{"NET_ADMIN"},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = false
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})
}

func validateContainerIP(container *dockertest.Resource) string {
	ip := container.GetIPInNetwork(tnscliDNSNetwork)
	fmt.Printf("DB DNS Container IP: %s\n", ip)
	return ip
}

func waitForDNSServer(pool *dockertest.Pool) error {
	dh := common.GetDockerHost(pool)
	if dh != "" {
		fmt.Printf("Docker Host: %s\n", dh)
	}
	ns := os.Getenv("DB_HOST")
	if ns != "" {
		fmt.Printf("DB_HOST variable was set to %s\n", ns)
	} else if dh != "" {
		ns = dh
	}
	if ns == "" {
		ns = tnscliDNSServer
	}

	// use default resolver and port
	r := netlib.NewResolver("", 0, true)
	r.IPv4Only = true
	lips, err := r.LookupIP(ns)
	if err != nil || len(lips) == 0 {
		return fmt.Errorf("could not resolve DNS server IP for %s: %v", ns, err)
	}
	ip := lips[0]
	tnscliDNSServer = ns
	fmt.Printf("DNS Host %s  IP resolved as %s\n", tnscliDNSServer, ip)
	pool.MaxWait = tnscliDNSTimeout * time.Second
	start := time.Now()
	err = pool.Retry(func() error {
		c, e := net.Dial("tcp", net.JoinHostPort(tnscliDNSServer, fmt.Sprintf("%d", tnscliDNSPort)))
		if e != nil {
			fmt.Printf("Err:%s\n", e)
			return e
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
		_ = container.Close()
	}

	if tnscliDNSNetworkCreated && tnscliDNSNetwork != nil {
		_ = tnscliDNSNetwork.Close()
	}
}
