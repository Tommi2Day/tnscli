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
		/*
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) < 1 {
					return errors.New("requires a service or all flag")
				}
				return nil
			},
		*/
	}
)

const defaultUser = "C##TCHECK"
const defaultPassword = "C0nnectMe!now"

var dbUser = ""
var dbPass = ""
var tnsKey = ""
var timeout = 15
var a = false

func init() {
	// don't have variables populated here
	checkCmd.PersistentFlags().StringVarP(&dbUser, "user", "u", dbUser, "User for real connect or set TNSCLI_USER")
	checkCmd.PersistentFlags().StringVarP(&dbPass, "password", "p", dbPass, "Password for real connect or set TNSCLI_PASSWORD")
	checkCmd.Flags().StringVarP(&tnsKey, "service", "s", "", "service name to check")
	checkCmd.PersistentFlags().BoolVarP(&a, "all", "a", false, "check all entries")
	checkCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", timeout, "timeout in sec")
	checkCmd.MarkFlagsMutuallyExclusive("all", "service")
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

	// all flag given, check every entry
	if a {
		log.Debugf("check all %d entries", l)
		var failed []string
		keys := make([]string, 0, len(tnsEntries))
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
			ok, elapsed, errmsg := CheckWithOracle(dbUser, dbPass, desc, timeout)
			if ok {
				o++
				fmt.Printf(" OK: %s\n", elapsed.Round(time.Millisecond))
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

	// not all modus, we have to  check one single entry
	// use first argument as service if is nothing given
	la := len(args)
	log.Debugf("service %s, args: %d -> %v", tnsKey, la, args)
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
		ok, elapsed, errmsg := CheckWithOracle(dbUser, dbPass, desc, timeout)
		if len(dbUser) > 0 {
			con = fmt.Sprintf("using user '%s'", dbUser)
		}
		if ok {
			log.Infof("service %s connected %s in %s\n", tnsKey, con, elapsed.Round(time.Millisecond))
			fmt.Printf("OK, service %s reachable\n", tnsAlias)
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
func CheckWithOracle(dbuser string, dbpass string, tnsDesc string, timeout int) (ok bool, elapsed time.Duration, err error) {
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
		val, err := dblib.SelectOneStringValue(db, sql)
		log.Debugf("Query returned sysdate = %s", val)
		if err == nil {
			ok = true
		}
	}
	return
}
