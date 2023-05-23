// Package cmd commands
package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	goora "github.com/sijms/go-ora/v2"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/dblib"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var (
	// check represents the list command
	checkCmd = &cobra.Command{
		Use:          "check",
		Short:        "Check TNS Entries",
		Long:         `Check all TNS Entries or one with real connect to database`,
		RunE:         checkTns,
		SilenceUsage: true,
	}
)

const defaultUser = "C##TCHECK"
const defaultPassword = "<MyCheckPassword>"

var dbUser = ""
var dbPass = ""
var tnsKey = ""
var timeout = 15
var a = false
var dbhost = false

func init() {
	checkCmd.PersistentFlags().StringVarP(&dbUser, "user", "u", dbUser, "User for real connect or set TNSCLI_USER")
	checkCmd.PersistentFlags().StringVarP(&dbPass, "password", "p", dbPass, "Password for real connect or set TNSCLI_PASSWORD")
	checkCmd.Flags().StringVarP(&tnsKey, "service", "s", "", "service name to check")
	checkCmd.PersistentFlags().BoolVarP(&a, "all", "a", false, "check all entries")
	checkCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", timeout, "timeout in sec")
	checkCmd.MarkFlagsMutuallyExclusive("all", "service")
	checkCmd.Flags().BoolVarP(&dbhost, "dbhost", "H", false, "print current host:cdb:pdb")
	RootCmd.AddCommand(checkCmd)
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
		file := entry.File
		tnsAlias := entry.Name
		con := ""
		log.Debugf("connect service %s from %s, timeout: %d s", tnsAlias, file, timeout)
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
		sql := "select to_char(sysdate,'YYYY-MM-DD HH24:MI:SS') from dual"
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
