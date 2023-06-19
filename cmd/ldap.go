// Package cmd commands
package cmd

import (
	"fmt"
	"os"
	"sort"
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

// TWorkStatus structure to handover statistics
type TWorkStatus map[string]int

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

var ldapClearCmd = &cobra.Command{
	Use:          "clear",
	Aliases:      []string{},
	Short:        "clear ldap tns entries",
	Long:         `clear oracle TNS LDAP Entries below BaseDN`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug("ldapClear called")
		return ldapClear()
	},
}
var ldapServer = ""
var ldapBindDN = ""
var ldapBindPassword = ""
var ldapBaseDN = ""
var contextDN = ""
var ldapPort = 0
var ldapInsecure = false
var ldapTLS = false
var ldapTimeout = 20
var tnsTarget = tnsAdmin + "/tnsnames.ora"

func init() {
	ldapCmd.PersistentFlags().StringVarP(&ldapServer, "ldap.host", "H", "", "Hostname of Ldap Server")
	ldapCmd.PersistentFlags().IntVarP(&ldapPort, "ldap.port", "p", ldapPort, "ldapport to connect, 0 means TLS flag will decide")
	ldapCmd.PersistentFlags().StringVarP(&ldapBaseDN, "ldap.base", "b", "", " Base DN to search from")
	ldapCmd.PersistentFlags().StringVarP(&contextDN, "ldap.oraclectx", "o", "", " Base DN of Oracle Context")
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

	ldapCmd.AddCommand(ldapClearCmd)
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
	if contextDN == "" {
		contextDN = viper.GetString("ldap.oraclectx")
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

	if len(contextDN) == 0 {
		contextDN, servers = dblib.ReadLdapOra(tnsAdmin)
	}
	if len(ldapBaseDN) == 0 && len(contextDN) > 0 {
		ldapBaseDN = strings.ReplaceAll(contextDN, "cn=OracleContext,", "")
	}
	lc, err = doConnect(servers)
	if err != nil {
		log.Errorf("ldap connect failed:%s", err)
		return
	}
	// check
	base := ldapBaseDN
	if contextDN != "" {
		base = contextDN
	}
	contextDN, err = dblib.GetOracleContext(lc, base)

	// verify
	if contextDN == "" {
		err = fmt.Errorf("no Oracle Context found on base %s", ldapBaseDN)
	}
	log.Infof("Oracle Context selected: %s", contextDN)
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
	var status TWorkStatus
	status, err = WriteLdapTns(lc, tnsEntries, domain, contextDN)
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
	tnsEntries, err := dblib.ReadLdapTns(lc, contextDN)
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

func ldapClear() (err error) {
	// print version
	version := GetVersion(false)
	log.Info(version)

	lc, err := ldapConnect()
	if err != nil {
		return
	}
	o, f := ClearLdapTns(lc, contextDN)
	log.Infof("SUCCESS: '%d' Entries deleted, %d  failed\n", o, f)
	if f == 0 {
		fmt.Printf("Clear LDAP finished successfully.")
	} else {
		err = fmt.Errorf("clearing LDAP TNS entries finished with %d errors", f)
	}
	return
}

// buildstatus creates ops task map to handle
func buildStatusMap(lc *ldaplib.LdapConfigType, tnsEntries dblib.TNSEntries, contextDN string) (dblib.TNSEntries, map[string]string, error) {
	var alias string
	ldapstatus := map[string]string{}

	ldapTNS, err := dblib.ReadLdapTns(lc, contextDN)
	if err != nil {
		return nil, ldapstatus, err
	}
	for _, a := range ldapTNS {
		alias = a.Name
		ldapstatus[alias] = ""
		log.Debugf("prepare status for LDAP Alias  %s", alias)
	}

	for _, t := range tnsEntries {
		alias = t.Name
		l, valid := ldapTNS[alias]
		if valid {
			comp := strings.Compare(l.Desc, t.Desc)
			if comp == 0 {
				ldapstatus[alias] = sOK
				log.Debugf("TNS Alias %s exists in LDAP and is equal ->OK", alias)
				continue
			}
			ldapstatus[alias] = sMod
			log.Debugf("TNS Alias %s exists in LDAP, but description changed ->MOD", alias)
		} else {
			ldapstatus[alias] = sNew
			log.Debugf("TNS Alias %s missed in LDAP ->NEW", alias)
		}
	}
	return ldapTNS, ldapstatus, err
}

// ClearLdapTns deletes all oraclenet entries below given context
func ClearLdapTns(lc *ldaplib.LdapConfigType, contextDN string) (ok int, fail int) {
	var err error
	// counter
	ok = 0
	fail = 0
	// verify OracleContext
	if contextDN == "" {
		log.Errorf("clearLdap:no OracleContext given")
		fail = 1
		return
	}
	log.Debugf("Use OracleContext DN %s", contextDN)
	// load available ldap entries
	ldapEntries, err := dblib.ReadLdapTns(lc, contextDN)
	if err != nil {
		log.Errorf("read failed:%s", err)
		fail = 1
		return
	}
	if len(ldapEntries) == 0 {
		log.Warnf("no entries found in Context %s", contextDN)
		return
	}

	// loop
	for _, e := range ldapEntries {
		dn := e.File
		alias := e.Name
		// check dn
		if dn != "" && strings.HasPrefix(dn, "cn=") {
			err = dblib.DeleteLdapTNSEntry(lc, dn, alias)
			if err != nil {
				log.Warnf("Cannot delete alias %s: %s", alias, err)
				fail++
			} else {
				ok++
				log.Infof("Ldap Alias %s deleted", alias)
			}
		} else {
			log.Warnf("Cannot delete alias %s with invalid dn '%s'", alias, dn)
			fail++
		}
	}
	return
}

// WriteLdapTns writes a set of TNS entries to Ldap
func WriteLdapTns(lc *ldaplib.LdapConfigType, tnsEntries dblib.TNSEntries, domain string, contextDN string) (TWorkStatus, error) {
	var ldapstatus map[string]string
	var ldapTNS dblib.TNSEntries
	var alias string
	var err error
	workStatus := make(TWorkStatus)
	workStatus[sOK] = 0
	workStatus[sMod] = 0
	workStatus[sNew] = 0
	workStatus[sDel] = 0
	workStatus[sSkip] = 0

	log.Infof("Update LDAP Context %s with %d tnsnames.ora entries using domain %s", contextDN, len(tnsEntries), domain)
	ldapTNS, ldapstatus, err = buildStatusMap(lc, tnsEntries, contextDN)
	status := ""

	// sort candidates
	sortedAlias := make([]string, 0, len(ldapstatus))
	for k := range ldapstatus {
		sortedAlias = append(sortedAlias, k)
	}
	sort.Strings(sortedAlias)
	for _, alias = range sortedAlias {
		status = ldapstatus[alias]
		switch status {
		case sOK:
			log.Debugf("Alias %s unchanged", alias)
			workStatus[sOK]++
		case sNew:
			tnsEntry, valid := tnsEntries[alias]
			if !valid {
				log.Warnf("Skip add invalid tns alias %s", alias)
				workStatus[sSkip]++
				continue
			}
			err = dblib.AddLdapTNSEntry(lc, contextDN, tnsEntry.Name, tnsEntry.Desc)
			if err != nil {
				log.Warnf("Add %s failed: %v", tnsEntry.Name, err)
				workStatus[sSkip]++
				continue
			}
			workStatus[sNew]++
			log.Debugf("Alias %s added", tnsEntry.Name)
		case sMod:
			// delete and add
			ldapEntry, valid := ldapTNS[alias]
			if !valid {
				log.Warnf("Skip modify invalid ldap alias %s", alias)
				workStatus[sSkip]++
				continue
			}
			dn := ldapEntry.File
			tnsEntry, valid := tnsEntries[alias]
			if !valid {
				log.Warnf("Skip modify invalid tns alias %s", alias)
				workStatus[sSkip]++
				continue
			}
			err = dblib.ModifyLdapTNSEntry(lc, dn, tnsEntry.Name, tnsEntry.Desc)
			if err != nil {
				log.Warnf("Modify %s failed: %v", tnsEntry.Name, err)
				workStatus[sSkip]++
			} else {
				log.Debugf("Alias %s replaced", tnsEntry.Name)
				workStatus[sMod]++
			}
		case "":
			ldapEntry, valid := ldapTNS[alias]
			if !valid {
				log.Warnf("Skip delete invalid ldap alias %s", alias)
				workStatus[sSkip]++
				continue
			}
			dn := ldapEntry.File
			err = dblib.DeleteLdapTNSEntry(lc, dn, alias)
			if err != nil {
				log.Warnf("Delete %s failed: %v", alias, err)
				workStatus[sSkip]++
			} else {
				log.Debugf("Alias %s deleted", alias)
				workStatus[sDel]++
			}
		}
	}
	log.Infof("%d TNS entries unchanged,%d new written, %d modified, %d deleted and %d skipped because of errors",
		workStatus[sOK], workStatus[sNew], workStatus[sMod], workStatus[sDel], workStatus[sSkip])
	return workStatus, err
}
