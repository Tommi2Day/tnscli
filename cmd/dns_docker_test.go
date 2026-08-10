package cmd

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/ory/dockertest/v4"
	"github.com/tommi2day/tnscli/test"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/netlib"

	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
)

const (
	tnscliDNSTimeout    = 10
	tnscliNetworkName   = "tnscli-dnsnetwork"
	tnscliNetworkPrefix = "172.25.2"
	tnscliRepoTag       = "9.21"
	tnscliTestAddr      = racaddr
)

var (
	tnscliDNSContainerName string
	tnscliDNSContainer     dockertest.ClosableResource
	tnscliDNSNetwork       dockertest.ClosableNetwork
	tnscliDNSServer        = common.GetStringEnv("DNS_HOST", "127.0.0.1")
	tnscliDNSPort          = 9055
)

// prepareDBlibDNSContainer create a Bind9 Docker Container
func prepareDNSContainer() (container dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_DNS") != "" {
		return nil, fmt.Errorf("skipping DB DNS Container in CI environment")
	}

	fmt.Printf("Try to prepare DB DNS Container on %s\n", tnscliDNSServer)
	tnscliDNSContainerName = getContainerName()
	pool, err := common.GetDockerPool()
	if err != nil {
		return nil, fmt.Errorf("pool: %s", err)
	}

	tnscliDNSNetwork, err = common.CreateNetworkWithSubnet(pool, tnscliNetworkName, tnscliNetworkPrefix+".0/24", tnscliNetworkPrefix+".1")
	if err != nil {
		return nil, fmt.Errorf("docker network: %s", err)
	}

	container, err = buildAndRunContainer(pool)
	if err != nil {
		return container, fmt.Errorf("docker container: %s", err)
	}
	err = container.ConnectToNetwork(context.Background(), tnscliDNSNetwork)
	if err != nil {
		return container, fmt.Errorf("connect to network: %s", err)
	}
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

func buildAndRunContainer(pool dockertest.Pool) (dockertest.ClosableResource, error) {
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	fmt.Printf("Try to build and start docker container %s\n", tnscliDNSContainerName)

	dockerContextDir := test.TestDir + "/docker/oracle-dns"
	imageName := fmt.Sprintf("tnscli-bind9:%s", tnscliRepoTag)
	port := fmt.Sprintf("%d", tnscliDNSPort)
	bindVersion := tnscliRepoTag

	return pool.BuildAndRun(context.Background(), imageName,
		&dockertest.BuildOptions{
			BuildArgs: map[string]*string{
				"VENDOR_IMAGE_PREFIX": &vendorImagePrefix,
				"BIND9_VERSION":       &bindVersion,
			},
			ContextDir: dockerContextDir,
			Dockerfile: "Dockerfile",
		},
		dockertest.WithHostname(tnscliDNSContainerName),
		dockertest.WithName(tnscliDNSContainerName),
		// need fixed mapping here
		dockertest.WithPortBindings(network.PortMap{
			network.MustParsePort("9055/tcp"): {
				{HostIP: netip.IPv4Unspecified(), HostPort: port},
			},
		}),
		dockertest.WithHostConfig(func(config *dockercontainer.HostConfig) {
			config.CapAdd = []string{"NET_ADMIN"}
			config.AutoRemove = false
			config.RestartPolicy = dockercontainer.RestartPolicy{Name: restartPolicyNo}
		}),
	)
}

func validateContainerIP(container dockertest.ClosableResource) string {
	ip := container.GetIPInNetwork(tnscliDNSNetwork)
	fmt.Printf("DB DNS Container IP: %s\n", ip)
	return ip
}

func waitForDNSServer(pool dockertest.Pool) error {
	dh := common.GetDockerHost(pool.Client().DaemonHost())
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
	start := time.Now()
	err = pool.Retry(context.Background(), tnscliDNSTimeout*time.Second, func() error {
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

func destroyDNSContainer(container dockertest.ClosableResource) {
	if container != nil {
		common.DestroyDockerContainer(container)
	}

	if tnscliDNSNetwork != nil {
		_ = tnscliDNSNetwork.Close(context.Background())
	}
}
