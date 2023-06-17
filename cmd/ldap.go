// Package cmd commands
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tommi2day/gomodules/dblib"
	"github.com/tommi2day/gomodules/ldaplib"
)

const (
	sOK   = "ok"
	sNew  = "new"
	sMod  = "mod"
	sDel  = "del"
	sSkip = "skip"
)

var (
	// check represents the list command
	ldapCmd = &cobra.Command{
		Use:   "ldap",
		Short: "LDAP TNS Entries",
		Long:  `handle TNS entries stored in LDAP Server`,
	}
)

var ldapReadCmd = &cobra.Command{
	Use:          "read",
	Aliases:      []string{"print"},
	Short:        "prints ldap tns entries to stdout",
	Long:         `read tns entries from LDAP Server`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug("ldapRead called")
		return ldapRead()
	},
}
var ldapWriteCmd = &cobra.Command{
	Use:          "write",
	Aliases:      []string{"save"},
	Short:        "update ldap tns entries",
	Long:         `update LDAP Entries from tnsnames.ora`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug("ldapWrite called")
		return ldapWrite()
	},
}

var ldapServer = ""
var ldapBindDN = ""
var ldapBindPassword = ""
var ldapBaseDN = ""
var ldapOracleContext = ""
var ldapPort = 0
var ldapInsecure = false
var ldapTLS = false
var ldapTimeout = 20
var tnsTarget = tnsAdmin + "/tnsnames.ora"

func init() {
	ldapCmd.PersistentFlags().StringVarP(&ldapServer, "ldap.host", "H", "", "Hostname of Ldap Server")
	ldapCmd.PersistentFlags().IntVarP(&ldapPort, "ldap.port", "p", ldapPort, "ldapport to connect, 0 means TLS flag will decide")
	ldapCmd.PersistentFlags().StringVarP(&ldapBaseDN, "ldap.base", "b", "", " Base DN to search from")
	ldapCmd.PersistentFlags().StringVarP(&ldapOracleContext, "ldap.oraclectx", "o", "", " Base DN of Oracle Context")
	ldapCmd.PersistentFlags().StringVarP(&ldapBindDN, "ldap.binddn", "D", "", "DN of user for LDAP bind, empty for anonymous access")
	ldapCmd.PersistentFlags().StringVarP(&ldapBindPassword, "ldap.bindpassword", "w", "", "password for LDAP Bind")
	ldapCmd.PersistentFlags().BoolVarP(&ldapTLS, "ldap.tls", "T", false, "use secure ldap (ldaps)")
	ldapCmd.PersistentFlags().BoolVarP(&ldapInsecure, "ldap.insecure", "I", false, "do not verify TLS")
	ldapCmd.PersistentFlags().IntVarP(&ldapTimeout, "ldap.timeout", "", ldapTimeout, "ldap timeout in sec")
	RootCmd.AddCommand(ldapCmd)

	ldapReadCmd.Flags().StringVarP(&tnsTarget, "ldap.tnstarget", "t", "", "filename to save ldap entries or stdout")
	ldapCmd.AddCommand(ldapReadCmd)

	ldapWriteCmd.Flags().StringVarP(&filename, "ldap.tnssource", "s", filename, "filename to read entries")
	ldapCmd.AddCommand(ldapWriteCmd)
}

func initLdapConfig() {
	if ldapServer == "" {
		ldapServer = viper.GetString("ldap.host")
	}
	if ldapPort == 0 {
		ldapPort = viper.GetInt("ldap.port")
	}
	if ldapBaseDN == "" {
		ldapBaseDN = viper.GetString("ldap.base")
	}
	if ldapOracleContext == "" {
		ldapOracleContext = viper.GetString("ldap.oraclectx")
	}
	if ldapBindDN == "" {
		ldapBindDN = viper.GetString("ldap.binddn")
	}
	if ldapBindPassword == "" {
		ldapBindPassword = viper.GetString("ldap.bindpassword")
	}
	if !ldapTLS {
		ldapTLS = viper.GetBool("ldap.tls")
	}
	if ldapInsecure {
		ldapInsecure = viper.GetBool("ldap.insecure")
	}
	if ldapTimeout == 0 {
		ldapTimeout = viper.GetInt("ldap.timeout")
	}
}

func ldapConnect() (lc *ldaplib.LdapConfigType, err error) {
	var servers []dblib.LdapServer

	if len(ldapOracleContext) == 0 {
		ldapOracleContext, servers = dblib.ReadLdapOra(tnsAdmin)
	}
	if len(ldapBaseDN) == 0 && len(ldapOracleContext) > 0 {
		ldapBaseDN = strings.ReplaceAll(ldapOracleContext, "cn=OracleContext,", "")
	}
	lc, err = doConnect(servers)
	if err != nil {
		log.Errorf("ldap connect failed:%s", err)
		return
	}
	// check
	base := ldapBaseDN
	if ldapOracleContext != "" {
		base = ldapOracleContext
	}
	ldapOracleContext, err = dblib.GetOracleContext(lc, base)

	// verify
	if ldapOracleContext == "" {
		err = fmt.Errorf("no Oracle Context found on base %s", ldapBaseDN)
	}
	log.Infof("Oracle Context selected: %s", ldapOracleContext)
	return
}

// separate function to make gocyclo happy
func doConnect(servers []dblib.LdapServer) (lc *ldaplib.LdapConfigType, err error) {
	switch {
	case ldapServer != "":
		log.Debugf("Try to connect to Ldap Server %s, Port %d, TLS %v, Insecure %v", ldapServer, ldapPort, ldapTLS, ldapInsecure)
		lc = ldaplib.NewConfig(ldapServer, ldapPort, ldapTLS, ldapInsecure, ldapBaseDN, ldapTimeout)
		err = lc.Connect(ldapBindDN, ldapBindPassword)
		if err == nil && lc.Conn != nil {
			log.Debugf("Ldap Connected")
		}
	case len(servers) > 0 && ldapServer == "":
		log.Debug("Try to use LDAP Server from ldap.ora")
		for _, s := range servers {
			ldapServer = s.Hostname
			ldapPort = s.Port
			sslport := s.SSLPort

			if sslport > 0 {
				ldapPort = sslport
				ldapTLS = true
			}
			log.Debugf("Try to connect to Ldap Server %s, Port %d", ldapServer, ldapPort)
			lc = ldaplib.NewConfig(ldapServer, ldapPort, ldapTLS, ldapInsecure, ldapBaseDN, ldapTimeout)
			err = lc.Connect(ldapBindDN, ldapBindPassword)
			if err == nil && lc.Conn != nil {
				log.Debugf("Connect to Ldap Server %s, Port %d", ldapServer, ldapPort)
				break
			}
		}
	default:
		err = fmt.Errorf("no Ldap Servers configured")
	}
	return
}

func ldapWrite() (err error) {
	var tnsEntries dblib.TNSEntries
	var domain string

	// print version
	version := GetVersion(false)
	log.Info(version)

	if filename == "" {
		err = fmt.Errorf("no input file to load given")
		return
	}
	tnsEntries, domain, err = dblib.GetTnsnames(filename, true)
	l := len(tnsEntries)
	if err != nil || l == 0 {
		if err == nil {
			err = fmt.Errorf("no Entries found")
		}
		log.Error(err)
		return
	}
	lc, err := ldapConnect()
	if err != nil {
		return
	}
	var status dblib.TWorkStatus
	status, err = dblib.WriteLdapTns(lc, tnsEntries, domain, ldapOracleContext)
	if err != nil {
		err = fmt.Errorf("write to ldap failed: %v", err)
		log.Error(err)
		return
	}
	o := status[sOK]
	n := status[sNew]
	m := status[sMod]
	d := status[sDel]
	s := status[sSkip]
	log.Infof("SUCCESS: '%s' written to LDAP - %d Entries Unchanged, New: %d, Mod: %d, Del: %d, Skip: %d\n", filename, o, n, m, d, s)
	fmt.Println("Finished successfully. For details run with --info or --debug")
	return
}

func ldapRead() (err error) {
	var fo *os.File
	lc, err := ldapConnect()
	if err != nil {
		return
	}
	// load available tns entries
	tnsEntries, err := dblib.ReadLdapTns(lc, ldapOracleContext)
	if err != nil {
		err = fmt.Errorf("read failed:%s", err)
		return
	}
	if tnsTarget == "" {
		fo = os.Stdout
		log.Debug("write to StdOut")
	} else {
		fo, err = os.Create(tnsTarget)
		if err != nil {
			err = fmt.Errorf("cannot create %s:%s ", tnsTarget, err)
			return
		}
		log.Debugf("write to %s", tnsTarget)
		// close fo on exit and check for its returned error
		defer func() {
			if err = fo.Close(); err != nil {
				log.Debugf("close failed")
			}
		}()
	}
	if len(tnsEntries) > 0 {
		err = outputTNS(tnsEntries, fo, true)
	}

	if err == nil {
		log.Infof("SUCCESS: %d LDAP entries found\n", len(tnsEntries))
	}
	return
}
