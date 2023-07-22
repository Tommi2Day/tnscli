// Package cmd commands
package cmd

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/viper"

	goora "github.com/sijms/go-ora/v2"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/dblib"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var (
	serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Service sub command",
		Long:  ``,
	}

	checkCmd = &cobra.Command{
		Use:          "check",
		Short:        "Check TNS Entries",
		Long:         `Check all TNS Entries or one with real connect to database`,
		RunE:         checkTns,
		SilenceUsage: true,
	}
	portcheckCmd = &cobra.Command{
		Use:          "portcheck",
		Short:        "try to connect each service and report if it is open or not",
		Long:         `list defined host:port and checks if requested. If racinfo.ini or SRV info given,  addresses will be checked as well`,
		RunE:         doPortcheck,
		SilenceUsage: true,
	}
	infoCmd = &cobra.Command{
		Use:   "info",
		Short: "give details for the given service",
		Long:  `printout more details for the service`,
	}
	portInfoCmd = &cobra.Command{
		Use:          "ports",
		Short:        "list service addresses and ports",
		Long:         `list defined host:port and checks if requested. If racinfo.ini given, it will be listed as well`,
		RunE:         portInfo,
		SilenceUsage: true,
	}
	jdbcInfoCmd = &cobra.Command{
		Use:          "jdbc",
		Short:        "print tns entry as jdbc string",
		Long:         `printout jdbc string for the service`,
		RunE:         getJdbcInfo,
		SilenceUsage: true,
	}
	tnsInfoCmd = &cobra.Command{
		Use:          "tns",
		Short:        "print tns entry for the given service",
		Long:         `printout tns entry for the service`,
		RunE:         getTnsInfo,
		SilenceUsage: true,
	}
)

const defaultUser = "C##TCHECK"
const defaultPassword = "<MyCheckPassword>"

var dbUser = ""
var dbPass = ""
var tnsKey = ""
var timeout = 15
var dbhost = false
var a = false
var racinfo = ""
var nodns = false
var ipv4 = false
var nameserver = ""
var pingTimeout = 5
var tcpcheck = false
var dnstcp = false

func init() {
	serviceCmd.PersistentFlags().StringVarP(&tnsKey, "service", "s", "", "service name to check")
	RootCmd.AddCommand(serviceCmd)

	checkCmd.PersistentFlags().StringVarP(&dbUser, "user", "u", dbUser, "User for real connect or set TNSCLI_USER")
	checkCmd.PersistentFlags().StringVarP(&dbPass, "password", "p", dbPass, "Password for real connect or set TNSCLI_PASSWORD")
	checkCmd.PersistentFlags().BoolVarP(&a, "all", "a", false, "check all entries")
	checkCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", timeout, "timeout in sec")
	checkCmd.Flags().BoolVarP(&dbhost, "dbhost", "H", false, "print actual connected host:cdb:pdb")

	portInfoCmd.Flags().StringVarP(&racinfo, "racinfo", "r", "", "path to racinfo.ini to resolve all RAC TCP Adresses, default $TNS_ADMIN/racinfo.ini")
	portInfoCmd.Flags().StringVarP(&nameserver, "nameserver", "n", "", "alternative nameserver to use for DNS lookup (IP:PORT)")
	portInfoCmd.Flags().BoolVar(&nodns, "nodns", false, "do not use DNS to resolve hostnames")
	portInfoCmd.Flags().BoolVar(&ipv4, "ipv4", false, "resolve only IPv4 addresses")
	portInfoCmd.Flags().BoolVar(&dnstcp, "dnstcp", false, "Use TCP to resolve DNS names")

	portcheckCmd.Flags().StringVarP(&racinfo, "racinfo", "r", "", "path to racinfo.ini to resolve all RAC TCP Adresses, default $TNS_ADMIN/racinfo.ini")
	portcheckCmd.Flags().StringVarP(&nameserver, "nameserver", "n", "", "alternative nameserver to use for DNS lookup (IP:PORT)")
	portcheckCmd.Flags().BoolVar(&nodns, "nodns", false, "do not use DNS to resolve hostnames")
	portcheckCmd.Flags().BoolVar(&ipv4, "ipv4", false, "resolve only IPv4 addresses")
	portcheckCmd.Flags().IntVarP(&pingTimeout, "timeout", "t", pingTimeout, "timeout for tcp ping")
	portcheckCmd.Flags().BoolVar(&dnstcp, "dnstcp", false, "Use TCP to resolve DNS names")

	infoCmd.AddCommand(portInfoCmd)
	infoCmd.AddCommand(jdbcInfoCmd)
	infoCmd.AddCommand(tnsInfoCmd)

	serviceCmd.AddCommand(checkCmd)
	serviceCmd.AddCommand(infoCmd)
	serviceCmd.AddCommand(portcheckCmd)
}
func getEntry(tnsKey string) (entry dblib.TNSEntry, err error) {
	// load available tns entries
	tnsEntries, domain, err := dblib.GetTnsnames(filename, true)
	l := len(tnsEntries)
	if err != nil {
		return
	}
	if l == 0 {
		err = fmt.Errorf("cannot proceed without tns entries")
		log.Error(err)
		return
	}
	if tnsKey == "" {
		err = fmt.Errorf("dont have a service to check, use --service to provide")
		return
	}
	log.Debugf("get info for service %s ", tnsKey)

	entry, found := dblib.GetEntry(tnsKey, tnsEntries, domain)
	if !found {
		err = fmt.Errorf("alias %s not found", tnsKey)
		return
	}
	return
}

func portInfo(_ *cobra.Command, args []string) (err error) {
	var allservices dblib.ServiceEntries
	if len(args) > 0 {
		tnsKey = args[0]
	}
	if tnsKey == "" {
		err = fmt.Errorf("dont have a service to check, use --service to provide")
		return
	}
	if racinfo == "" {
		racinfo = viper.GetString("tns_admin") + "/racinfo.ini"
	}
	entry, err := getEntry(tnsKey)
	if err == nil {
		servers := entry.Servers
		l := len(servers)
		if l == 0 {
			err = fmt.Errorf("alias %s: No hosts found", tnsKey)
			return
		}
		log.Infof("Alias %s uses %d hosts", tnsKey, l)
		dblib.IgnoreDNSLookup = nodns
		dblib.IPv4Only = ipv4
		ns, p, e := common.GetHostPort(nameserver)
		if e != nil {
			ns = nameserver
			p = 0
		}
		dblib.Resolver = dblib.SetResolver(ns, p, dnstcp)
		for _, s := range servers {
			host := s.Host
			port := s.Port
			services := dblib.GetRacAdresses(host, racinfo)
			if len(services) == 0 {
				log.Debugf("no racinfo found, will use original entry %s:%s", host, port)
				services = append(services, dblib.ServiceEntryType{Host: host, Port: port, Address: host + ":" + port})
			}
			allservices = append(allservices, services...)
		}
		log.Infof("Alias %s uses %d addresses", tnsKey, len(allservices))
		for _, s := range allservices {
			if tcpcheck {
				doTCPPing(s.Host, s.Address)
			} else {
				fmt.Printf("%s (%s)\n", s.Host, s.Address)
			}
		}
		return
	}
	return
}

func doPortcheck(c *cobra.Command, args []string) (err error) {
	tcpcheck = true
	err = portInfo(c, args)
	return
}

func doTCPPing(host string, address string) {
	d := net.Dialer{Timeout: time.Duration(pingTimeout) * time.Second}
	_, err := d.Dial("tcp", address)
	if err != nil {
		match, _ := regexp.MatchString("refused", err.Error())
		if match {
			// Closed
			log.Infof("%s, %s is CLOSED/REFUSED (no service)", host, address)
			fmt.Printf("%s (%s) is CLOSED/REFUSED (no service)\n", host, address)
			return
		}
		match, _ = regexp.MatchString("timeout", err.Error())
		if match {
			// Timeout
			log.Infof("%s (%s) TIMEOUT (blocked)", host, address)
			fmt.Printf("%s (%s) TIMEOUT (blocked)\n", host, address)
			return
		}
	} else {
		// Open
		log.Infof("%s(%s) is OPEN", host, address)
		fmt.Printf("%s (%s) is OPEN\n", host, address)
		return
	}
	// Default
	log.Infof("%s(%s) UNKNOWN ", host, address)
	fmt.Printf("%s (%s) UNKNOWN\n", host, address)
}

func getTnsInfo(_ *cobra.Command, args []string) (err error) {
	if tnsKey == "" {
		tnsKey = args[0]
	}
	entry, err := getEntry(tnsKey)
	if err == nil {
		desc := entry.Desc
		tnsAlias := entry.Name
		loc := entry.Location
		desc = strings.ReplaceAll(desc, "\r", " ")
		desc = strings.ReplaceAll(desc, "\n", "\n  ")
		desc = strings.ReplaceAll(desc, "(ADDRESS_LIST", "  (ADDRESS_LIST")
		desc = strings.ReplaceAll(desc, "(CONNECT_DATA", "  (CONNECT_DATA")
		out := fmt.Sprintf("# Location: %s \n%s=  %s", loc, tnsAlias, desc)
		log.Infof(out)
		fmt.Println(out)
	}
	return
}

func getJdbcInfo(_ *cobra.Command, args []string) (err error) {
	if tnsKey == "" {
		tnsKey = args[0]
	}
	entry, err := getEntry(tnsKey)
	if err == nil {
		desc := entry.Desc
		repl := strings.NewReplacer("\r", "", "\n", "", "\t", "", " ", "")
		desc = repl.Replace(desc)
		out := fmt.Sprintf("jdbc:Oracle:thin@%s", desc)
		log.Infof(out)
		fmt.Println(out)
	}
	return
}

func checkTns(_ *cobra.Command, args []string) (err error) {
	// load available tns entries
	tnsEntries, domain, err := dblib.GetTnsnames(filename, true)
	l := len(tnsEntries)
	if err != nil {
		return
	}
	if l == 0 {
		err = fmt.Errorf("cannot proceed without tns entries")
		log.Error(err)
		return
	}
	log.Debugf("have %d tns entries", l)
	if dbUser == "" {
		dbUser = common.GetEnv("TNSCLI_USER", "")
		log.Debugf("use default db user %s", dbUser)
	}
	if dbPass == "" {
		dbPass = common.GetEnv("TNSCLI_PASSWORD", "")
		log.Debug("use default db pass")
	}

	// do checks depending on mode
	if a {
		// all flag given, check every entry
		return allCheck(tnsEntries)
	}
	// check specific entries from arg
	return singleCheck(args, tnsEntries, domain)
}

func allCheck(tnsEntries dblib.TNSEntries) (err error) {
	var failed []string
	l := len(tnsEntries)
	log.Debugf("check all %d entries", l)
	keys := make([]string, 0, l)
	for k := range tnsEntries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	o := 0
	e := 0
	i := 0
	for _, k := range keys {
		entry := tnsEntries[k]
		desc := entry.Desc
		tnsAlias := entry.Name
		fmt.Printf("%s: ", tnsAlias)
		ok, elapsed, hostval, errmsg := CheckWithOracle(dbUser, dbPass, desc, timeout)
		if ok {
			o++
			if dbhost {
				fmt.Printf(" OK-> %s, %s\n", hostval, elapsed.Round(time.Millisecond))
			} else {
				fmt.Printf(" OK-> %s\n", elapsed.Round(time.Millisecond))
			}
		} else {
			e++
			fmt.Printf(" ERROR: %s\n", errmsg)
			failed = append(failed, fmt.Sprintf("%s: %v", tnsAlias, errmsg))
		}
		i++
	}
	log.Info("Checks finished ...")
	log.Infof(" %d entries checked, %d ok, %d failed\n", i, o, e)
	if len(failed) > 0 {
		for _, s := range failed {
			fmt.Println(s)
		}
		err = fmt.Errorf("some checks failed")
	}
	return
}

func singleCheck(args []string, tnsEntries dblib.TNSEntries, domain string) (err error) {
	// not all modus, we have to  check one single entry
	// use first argument as service if is nothing given
	la := len(args)
	log.Debugf("service %s, domain '%s', args: %d -> %v", tnsKey, domain, la, args)
	if tnsKey == "" && la > 0 {
		tnsKey = args[0]
	}
	if tnsKey == "" {
		err = fmt.Errorf("dont have a service to check, use --service to provide")
		return
	}
	log.Debugf("enter check for service %s ", tnsKey)

	if entry, found := dblib.GetEntry(tnsKey, tnsEntries, domain); found {
		desc := entry.Desc
		location := entry.Location
		tnsAlias := entry.Name
		con := ""
		log.Debugf("connect service %s from %s, timeout: %d s", tnsAlias, location, timeout)
		log.Infof("use entry \n%s=%s\n", tnsAlias, strings.ReplaceAll(desc, "\r", " "))
		ok, elapsed, hostval, errmsg := CheckWithOracle(dbUser, dbPass, desc, timeout)
		if len(dbUser) > 0 {
			con = fmt.Sprintf("using user '%s'", dbUser)
		}
		hv := ""
		if ok {
			if dbhost && hostval == "" {
				err = fmt.Errorf("database is available, but couldnt extract host info from database for alias %s, maybe login failed", tnsKey)
				return
			}
			if hostval != "" {
				hv = "(" + hostval + ") "
			}
			log.Infof("service %s connected %s%s in %s\n", tnsKey, hv, con, elapsed.Round(time.Millisecond))
			if dbhost {
				fmt.Printf("%s -> %s\n", tnsKey, hostval)
			} else {
				fmt.Printf("OK, service %s reachable\n", tnsAlias)
			}
		} else {
			err = fmt.Errorf("service %s %s NOT reached:%s", tnsAlias, con, errmsg)
		}
		return
	}
	err = fmt.Errorf("alias %s not found", tnsKey)
	return
}

// CheckWithOracle try connecting to oracle with dummy creds to get an ORA error.
// If this happens, the connection is working
func CheckWithOracle(dbuser string, dbpass string, tnsDesc string, timeout int) (ok bool, elapsed time.Duration, hostval string, err error) {
	urlOptions := map[string]string{
		// "CONNECTION TIMEOUT": "3",
	}
	ok = false
	if dbuser == "" {
		dbuser = defaultUser
	}
	if dbpass == "" {
		dbpass = defaultPassword
	}
	// jdbc url needs spaces stripped
	tnsDesc = strings.Join(strings.Fields(tnsDesc), "")
	url := goora.BuildJDBC(dbuser, dbpass, tnsDesc, urlOptions)
	log.Debugf("Try to connect %s@%s", dbuser, tnsDesc)
	start := time.Now()
	db, err := dblib.DBConnect("oracle", url, timeout)
	elapsed = time.Since(start)

	// check results
	if err != nil {
		// check error code, we expect 1017
		isOerr, code, _ := dblib.HaveOerr(err)
		if isOerr && code == 1017 {
			ok = true
			log.Warnf("Connect OK, but Login error, maybe expected")
		}
	} else {
		log.Debugf("Connection OK, test if db is open using select")
		sql := "select 'DB is open, sysdate:'||to_char(sysdate,'YYYY-MM-DD HH24:MI:SS') from dual"
		if dbhost {
			// extract host,cdb and pdb from database
			sql = "select sys_context('USERENV','SERVER_HOST')||':'||sys_context('USERENV','INSTANCE_NAME')||':'||nvl(sys_context('USERENV','CON_NAME'),'') as dbhost from dual"
		}
		hostval, err = dblib.SelectOneStringValue(db, sql)
		log.Infof("Query returned:  %s", hostval)
		if err == nil {
			ok = true
		}
	}
	return
}
