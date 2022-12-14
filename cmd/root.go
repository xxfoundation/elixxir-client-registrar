////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger

package cmd

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client-registrar/storage"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/utils"
	"net"
	"os"
	"os/signal"
	"path"
	"sync/atomic"
	"syscall"
	"time"
)

type Params struct {
	Address           string
	CertPath          string
	KeyPath           string
	SignedCertPath    string
	SignedKeyPath     string
	userRegCapacity   uint32
	userRegLeakPeriod time.Duration
	publicAddress     string
}

var (
	cfgFile        string
	logLevel       uint // 0 = info, 1 = debug, >1 = trace
	noTLS          bool
	ClientRegCodes []string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client_registrar",
	Short: "Runs a registration server for cMix",
	Long:  `This server provides client registration functions on cMix`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse config file options
		certPath := viper.GetString("certPath")
		keyPath := viper.GetString("keyPath")
		signedCertPath := viper.GetString("signedCertPath")
		signedKeyPath := viper.GetString("signedKeyPath")

		localAddress := fmt.Sprintf("0.0.0.0:%d", viper.GetInt("port"))
		ipAddr := viper.GetString("publicAddress")

		publicAddress := fmt.Sprintf("%s:%d", ipAddr, viper.GetInt("port"))

		// Set up database connection
		rawAddr := viper.GetString("dbAddress")

		var addr, port string
		var err error
		if rawAddr != "" {
			addr, port, err = net.SplitHostPort(rawAddr)
			if err != nil {
				jww.FATAL.Panicf("Unable to get database port: %+v", err)
			}
		}

		db, _, err := storage.NewDatabase(
			viper.GetString("dbUsername"),
			viper.GetString("dbPassword"),
			viper.GetString("dbName"),
			addr,
			port,
		)
		if err != nil {
			jww.FATAL.Panicf("Unable to initialize storage: %+v", err)
		}

		ClientRegCodes = viper.GetStringSlice("clientRegCodes")
		err = db.PopulateClientRegistrationCodes(ClientRegCodes, 1000)
		if err != nil {
			jww.FATAL.Panicf("Failed to insert client registration codes: %+v", err)
		}

		userRegLeakPeriodString := viper.GetString("userRegLeakPeriod")
		var userRegLeakPeriod time.Duration
		if userRegLeakPeriodString != "" {
			// specified, so try to parse
			userRegLeakPeriod, err = time.ParseDuration(userRegLeakPeriodString)
			if err != nil {
				jww.FATAL.Panicf("Could not parse duration: %+v", err)
			}
		} else {
			// use default
			userRegLeakPeriod = time.Hour * 24
		}
		userRegCapacity := viper.GetUint32("userRegCapacity")
		if userRegCapacity == 0 {
			// use default
			userRegCapacity = 1000
		}

		viper.SetDefault("addressSpace", 5)

		// Populate params
		params := Params{
			Address:           localAddress,
			CertPath:          certPath,
			KeyPath:           keyPath,
			SignedCertPath:    signedCertPath,
			SignedKeyPath:     signedKeyPath,
			publicAddress:     publicAddress,
			userRegCapacity:   userRegCapacity,
			userRegLeakPeriod: userRegLeakPeriod,
		}

		jww.INFO.Println("Starting client registration Server...")

		// Start registration server
		impl, err := StartRegistrar(params, &db)
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}

		// Block forever on Signal Handler for safe program exit
		stopCh := ReceiveExitSignal()

		// Block forever to prevent the program ending
		// Block until a signal is received, then call the function
		// provided
		select {
		case <-stopCh:
			jww.INFO.Printf(
				"Received Exit (SIGTERM or SIGINT) signal...\n")
			if atomic.LoadUint32(impl.Stopped) != 1 {
				os.Exit(-1)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// NOTE: The point of init() is to be declarative.
	// There is one init in each sub command. Do not put variable declarations
	// here, and ensure all the Flags are of the *P variety, unless there's a
	// very good reason not to have them as local params to sub command."
	cobra.OnInitialize(initConfig, initLog)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.Flags().UintVarP(&logLevel, "logLevel", "l", 1,
		"Level of debugging to display. 0 = info, 1 = debug, >1 = trace")

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c",
		"", "Sets a custom config file path")

	rootCmd.Flags().BoolVar(&noTLS, "noTLS", false,
		"Runs without TLS enabled")

	rootCmd.Flags().StringP("close-timeout", "t", "60s",
		"Amount of time to wait for rounds to stop running after"+
			" receiving the SIGUSR1 and SIGTERM signals")

	rootCmd.Flags().StringP("kill-timeout", "k", "60s",
		"Amount of time to wait for round creation to stop after"+
			" receiving the SIGUSR2 and SIGTERM signals")

	err := viper.BindPFlag("closeTimeout",
		rootCmd.Flags().Lookup("close-timeout"))
	if err != nil {
		jww.FATAL.Panicf("could not bind flag: %+v", err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Use default config location if none is passed
	if cfgFile == "" {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(1)
		}

		cfgFile = home + "/.elixxir/registration.yaml"

	}

	validConfig := true
	f, err := os.Open(cfgFile)
	if err != nil {
		jww.ERROR.Printf("Unable to open config file (%s): %+v", cfgFile, err)
		validConfig = false
	}
	_, err = f.Stat()
	if err != nil {
		jww.ERROR.Printf("Invalid config file (%s): %+v", cfgFile, err)
		validConfig = false
	}
	err = f.Close()
	if err != nil {
		jww.ERROR.Printf("Unable to close config file (%s): %+v", cfgFile, err)
		validConfig = false
	}

	// Set the config file if it is valid
	if validConfig {
		// Set the config path to the directory containing the config file
		// This may increase the reliability of the config watching, somewhat
		cfgDir, _ := path.Split(cfgFile)
		viper.AddConfigPath(cfgDir)

		viper.SetConfigFile(cfgFile)
		viper.AutomaticEnv() // read in environment variables that match

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err != nil {
			jww.ERROR.Printf("Unable to parse config file (%s): %+v", cfgFile, err)
			validConfig = false
		}
		viper.WatchConfig()
	}
}

// initLog initializes logging thresholds and the log path.
func initLog() {
	if viper.Get("logPath") != nil {
		vipLogLevel := viper.GetUint("logLevel")

		// Check the level of logs to display
		if vipLogLevel > 1 {
			// Set the GRPC log level
			err := os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "info")
			if err != nil {
				jww.ERROR.Printf("Could not set GRPC_GO_LOG_SEVERITY_LEVEL: %+v", err)
			}

			err = os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "99")
			if err != nil {
				jww.ERROR.Printf("Could not set GRPC_GO_LOG_VERBOSITY_LEVEL: %+v", err)
			}
			// Turn on trace logs
			jww.SetLogThreshold(jww.LevelTrace)
			jww.SetStdoutThreshold(jww.LevelTrace)
			mixmessages.TraceMode()
		} else if vipLogLevel == 1 {
			// Turn on debugging logs
			jww.SetLogThreshold(jww.LevelDebug)
			jww.SetStdoutThreshold(jww.LevelDebug)
			mixmessages.DebugMode()
		} else {
			// Turn on info logs
			jww.SetLogThreshold(jww.LevelInfo)
			jww.SetStdoutThreshold(jww.LevelInfo)
		}

		// Create log file, overwrites if existing
		logPath := viper.GetString("logPath")
		fullLogPath, _ := utils.ExpandPath(logPath)
		logFile, err := os.OpenFile(fullLogPath,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644)
		if err != nil {
			jww.WARN.Println("Invalid or missing log path, default path used.")
		} else {
			jww.SetLogOutput(logFile)
		}
	}
}

// ReceiveExitSignal signals a stop chan when it receives
// SIGTERM or SIGINT
func ReceiveExitSignal() chan os.Signal {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return c
}
