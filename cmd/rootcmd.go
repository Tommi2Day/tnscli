// Package cmd commands
package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"

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

	configEnvPrefix = "TNSCLI"
	configName      = "tnscli"
	configType      = "yaml"
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
	cfgFile   = ""
)

func init() {
	cobra.OnInitialize(initConfig, initLdapConfig)
	// parse commandline
	RootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "", false, "verbose debug output")
	RootCmd.PersistentFlags().BoolVarP(&infoFlag, "info", "", false, "reduced info output")
	RootCmd.PersistentFlags().StringVarP(&tnsAdmin, "tns_admin", "A", tnsAdmin, "TNS_ADMIN directory")
	RootCmd.PersistentFlags().StringVarP(&filename, "filename", "f", "", "path to alternate tnsnames.ora")
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
	RootCmd.MarkFlagsMutuallyExclusive("debug", "info")

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
	viper.SetConfigType(configType)
	viper.SetConfigName(configName)
	if cfgFile == "" {
		// Use config file from the flag.
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home/etc and current directory).
		etc := path.Join(home, "etc")
		viper.AddConfigPath(etc)
		viper.AddConfigPath(".")
	} else {
		// set filename form cli
		viper.SetConfigFile(cfgFile)
	}

	// env var overrides
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix(configEnvPrefix)
	// env var `LDAP_USERNAME` will be mapped to `ldap.username`
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	haveConfig := false
	if err == nil {
		cfgFile = viper.ConfigFileUsed()
		haveConfig = true
		viper.Set("config", cfgFile)

		if RootCmd.Flags().Lookup("debug").Changed {
			viper.Set("debug", debugFlag)
		}
		if RootCmd.Flags().Lookup("info").Changed {
			viper.Set("info", infoFlag)
		}
		if RootCmd.Flags().Lookup("tns_admin").Changed {
			viper.Set("tns_admin", tnsAdmin)
		}
	}
	// logger settings
	log.SetLevel(log.ErrorLevel)
	switch {
	case viper.GetBool("debug"):
		// report function name
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
	case viper.GetBool("info"):
		log.SetLevel(log.InfoLevel)
	}
	logFormatter := &prefixed.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	}
	log.SetFormatter(logFormatter)

	// debug config file
	if haveConfig {
		log.Debugf("found configfile %s", cfgFile)
	} else {
		log.Debugf("Error using %s config : %s", configType, err)
	}

	// fix tnsadmin settings
	ta, err := dblib.CheckTNSadmin(viper.GetString("tns_admin"))
	if err == nil {
		tnsAdmin = ta
		viper.Set("tns_admin", tnsAdmin)
	}
	if filename == "" {
		filename = path.Join(tnsAdmin, "tnsnames.ora")
	}
}
