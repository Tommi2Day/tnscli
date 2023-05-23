// Package cmd commands
package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/tommi2day/gomodules/dblib"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var (
	// check represents the list command
	listCmd = &cobra.Command{
		Use:          "list",
		Short:        "list TNS Entries",
		Long:         `list all TNS Entries`,
		RunE:         listTns,
		SilenceUsage: true,
	}
)
var search = ""
var complete = false

func init() {
	// don't have variables populated here
	listCmd.Flags().StringVarP(&search, "search", "s", "", "search for tns name")
	listCmd.Flags().BoolVarP(&complete, "complete", "C", false, "print complete entry")
	RootCmd.AddCommand(listCmd)
}

// search or list tns entries
func listTns(_ *cobra.Command, _ []string) error {
	// load available tns entries
	tnsEntries, _, err := dblib.GetTnsnames(filename, true)
	l := len(tnsEntries)
	if err != nil || l == 0 {
		log.Info("No Entries found")
		return err
	}
	log.Infof("list %d entries", l)
	err = outputTNS(tnsEntries, nil, complete)
	return err
}

func outputTNS(tnsEntries dblib.TNSEntries, fo *os.File, full bool) (err error) {
	l := len(tnsEntries)
	if search == "" {
		log.Debugf("list %d entries", l)
	} else {
		log.Debugf("search for %s in %d entries", search, l)
	}
	keys := make([]string, 0, len(tnsEntries))
	for k := range tnsEntries {
		keys = append(keys, k)
	}
	if fo == nil {
		fo = os.Stdout
	}
	sort.Strings(keys)
	f := 0
	var re *regexp.Regexp
	if search != "" {
		re = regexp.MustCompile("(?i)" + search)
	}
	for _, k := range keys {
		out := formatEntry(tnsEntries, k, full)
		if search == "" {
			_, err = fmt.Fprintln(fo, out)
		} else if re.Match([]byte(k)) {
			log.Debugf("alias %s matches search string %s", k, search)
			f++
			_, err = fmt.Fprintln(fo, out)
		}
		if err != nil {
			log.Errorf("Error while writing:%v", err)
			return
		}
	}
	if search != "" {
		log.Infof("found %d entries\n", f)
		if f == 0 {
			err = fmt.Errorf("no alias with '%s' found", search)
		}
	}
	return
}

// format one tns entry
func formatEntry(entries dblib.TNSEntries, key string, full bool) (out string) {
	if full {
		entry := entries[key]
		desc := entry.Desc
		tnsAlias := entry.Name
		desc = strings.ReplaceAll(desc, "\r", " ")
		desc = strings.ReplaceAll(desc, "\n", "\n  ")
		desc = strings.ReplaceAll(desc, "(ADDRESS_LIST", "  (ADDRESS_LIST")
		desc = strings.ReplaceAll(desc, "(CONNECT_DATA", "  (CONNECT_DATA")
		out = fmt.Sprintf("%s=  %s", tnsAlias, desc)
	} else {
		out = fmt.Sprintf("%s\n", key)
	}
	return
}
