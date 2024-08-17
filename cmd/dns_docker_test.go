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

const DNScontainerTimeout = 10
const networkName = "dblib-dnsnetwork"

var dnscontainerName string
var dnsContainer *dockertest.Resource
var dnsnetwork *dockertest.Network
var networkCreated = false
var dnsserver = ""
var dnsport = 0

// prepareDNSContainer create a Bind9 Docker Container
func prepareDNSContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_DNS") != "" {
		err = fmt.Errorf("skipping DNS Container in CI environment")
		return
	}
	dnscontainerName = os.Getenv("DNS_CONTAINER_NAME")
	if dnscontainerName == "" {
		dnscontainerName = "tnscli-bind9"
	}
	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}

	networks, err := pool.NetworksByName(networkName)
	if err != nil || len(networks) == 0 {
		dnsnetwork, err = pool.CreateNetwork(networkName, func(options *docker.CreateNetworkOptions) {
			options.Name = networkName
			options.CheckDuplicate = true
			options.IPAM = &docker.IPAMOptions{
				Driver: "default",
				Config: []docker.IPAMConfig{{
					Subnet:  "172.25.1.0/24",
					Gateway: "172.25.1.1",
				}},
			}
			options.EnableIPv6 = false
			// options.Internal = true
		})
		if err != nil {
			err = fmt.Errorf("could not create Network: %s:%s", networkName, err)
			return
		}
		networkCreated = true
	} else {
		dnsnetwork = &networks[0]
	}

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")

	fmt.Printf("Try to build and start docker container  %s\n", dnscontainerName)
	buildArgs := []docker.BuildArg{
		{
			Name:  "VENDOR_IMAGE_PREFIX",
			Value: vendorImagePrefix,
		},
		{
			Name:  "BIND9_VERSION",
			Value: "9.18",
		},
	}
	container, err = pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: test.TestDir + "/docker/dns",
			Dockerfile: "Dockerfile",
		},
		&dockertest.RunOptions{
			Hostname:     dnscontainerName,
			Name:         dnscontainerName,
			Networks:     []*dockertest.Network{dnsnetwork},
			ExposedPorts: []string{"53/tcp", "53/udp", "953/tcp"},
		}, func(config *docker.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})

	if err != nil {
		err = fmt.Errorf("error starting dns docker container: %v", err)
		return
	}
	// ip := container.Container.NetworkSettings.Networks[networkName].IPAddress
	ip := container.GetIPInNetwork(dnsnetwork)
	if ip != "172.25.1.2" {
		err = fmt.Errorf("internal ip not as expected: %s", ip)
		return
	}
	pool.MaxWait = DNScontainerTimeout * time.Second
	dnsserver, dnsport = common.GetContainerHostAndPort(container, "53/tcp")
	fmt.Printf("Wait to successfully connect to DNS to %s:%d (max %ds)...\n", dnsserver, dnsport, DNScontainerTimeout)
	start := time.Now()
	var c net.Conn
	if err = pool.Retry(func() error {
		c, err = net.Dial("tcp", fmt.Sprintf("%s:%d", dnsserver, dnsport))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
		}
		return err
	}); err != nil {
		fmt.Printf("Could not connect to DNS Container: %d", err)
		return
	}
	_ = c.Close()

	// wait 5s to init container
	time.Sleep(5 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("DNS Container is available after %s\n", elapsed.Round(time.Millisecond))
	// test dns
	dns := netlib.NewResolver(dnsserver, dnsport, true)
	ips, e := dns.LookupIP(racaddr)
	if e != nil || len(ips) == 0 {
		fmt.Printf("Could not resolve DNS with %s: %v", racaddr, e)
		return
	}
	fmt.Println("DNS Container is ready, host", racaddr, "resolved to", ips[0])
	err = nil
	return
}

func destroyDNSContainer(container *dockertest.Resource) {
	common.DestroyDockerContainer(container)
	if networkCreated {
		_ = dnsnetwork.Close()
	}
}
