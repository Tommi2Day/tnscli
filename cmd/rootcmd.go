// Package cmd commands
package cmd

import (
	"os"
	"path"
	"time"

	"github.com/tommi2day/gomodules/dblib"

	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tommi2day/gomodules/common"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const (
// allows you to override any config values using
// env APP_MY_VAR = "MY_VALUE"
// e.g. export APP_LDAP_USERNAME test
// maps to ldap.username
/*
	configEnvPrefix = "TNSCLI"
	configName      = "tnscli"
	configType      = "yaml"
*/
)

var (
	// RootCmd function to execute in tests
	RootCmd = &cobra.Command{
		Use:   "tnscli",
		Short: "tnscli – Small Oracle TNS Service and Connect Test Tool",
		Long:  ``,
	}

	oracleHome = common.GetEnv("ORACLE_HOME", "/opt/oracle")
	tnsAdmin   = common.GetEnv("TNS_ADMIN", path.Join(oracleHome, "network", "admin"))

	filename  = ""
	debugFlag = false
	infoFlag  = false
)

func init() {
	cobra.OnInitialize(initConfig)
	// parse commandline
	RootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "", false, "verbose debug output")
	RootCmd.PersistentFlags().BoolVarP(&infoFlag, "info", "", false, "reduced info output")
	RootCmd.PersistentFlags().StringVarP(&tnsAdmin, "tns_admin", "A", tnsAdmin, "TNS_ADMIN directory")
	RootCmd.PersistentFlags().StringVarP(&filename, "filename", "f", "", "path to alternate tnsnames.ora")
	RootCmd.MarkFlagsMutuallyExclusive("debug", "info")
	// don't have variables populated here
	if err := viper.BindPFlags(RootCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}

// Execute run application
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		// fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	// logger settings
	log.SetLevel(log.ErrorLevel)
	switch {
	case debugFlag:
		// report function name
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
	case infoFlag:
		log.SetLevel(log.InfoLevel)
	}
	ta, err := dblib.CheckTNSadmin(tnsAdmin)
	if err == nil {
		tnsAdmin = ta
	}
	if filename == "" {
		filename = path.Join(tnsAdmin, "tnsnames.ora")
	}
	logFormatter := &prefixed.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	}
	log.SetFormatter(logFormatter)
}
