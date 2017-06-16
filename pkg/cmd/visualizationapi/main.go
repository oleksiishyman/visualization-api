package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	flag "github.com/spf13/pflag"

	"visualization-api/pkg/config"
	"visualization-api/pkg/database"
	"visualization-api/pkg/http_endpoint"
	"visualization-api/pkg/http_endpoint/common"
	"visualization-api/pkg/logging"
	"visualization-api/pkg/openstack"
)

var (
	logRotate  *log.RotateWriter
	version    = "UNDEFINED"
	gitVersion = "UNDEFINED"

	//app level flags
	versionParam = flag.Bool("version", false, "Prints version information")
)

func exitWithError(err error, optional ...string) {
	fmt.Println(optional, err)
	os.Exit(1)
}

func cleanupOnExit() {
	// this function is used to perform all cleanup on application exit
	// such as file descriptor close
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigc
		log.Logger.Info("Caught signal '", s, "' shutting down")
		// close global descriptor
		logRotate.Lock.Lock()
		defer logRotate.Lock.Unlock()
		err := logRotate.Fp.Close()
		if err != nil {
			fmt.Println(err)
		}
		os.Exit(0)
	}()
}

func main() {

	/*
		APP INITIALIZATION STEPS
		1 - initialize config module. It would read config file, env variables
			and flags and merge them into one structure
		2 - initialize io.Writer that rotates files it is writing to
		3 - initialize logging module with rotation writer, created in step 2
		4 - initialize database connection
		5 - intiialize openstack client
		6 - initialize signals handler, to close file in rotation logger
		7 - initialize http server
	*/

	flag.Parse()

	if *versionParam {
		fmt.Printf("visualization-api version %s %s \n", version, gitVersion)
		os.Exit(0)
	}

	// initialize config
	errorParsingConfig := config.InitializeConfig()
	if errorParsingConfig != nil {
		exitWithError(errorParsingConfig)
	}
	CONF := config.GetConfig()

	// create rotation logger
	var rotateInitError error
	logRotate, rotateInitError = log.NewRotateWriter(CONF.LogFilePath)
	if rotateInitError != nil {
		exitWithError(rotateInitError)
	}

	// initialize logger
	log.InitializeLogger(logRotate, CONF.ConsoleDebug, CONF.LogLevel)

	// initialize database connection
	databaseInitializationError := db.InitializeEngine(
		CONF.MysqlUsername,
		CONF.MysqlPassword,
		CONF.MysqlHost,
		CONF.MysqlDatabaseName,
		CONF.MysqlPort,
	)
	if databaseInitializationError != nil {
		exitWithError(databaseInitializationError)
	}

	openstackCli, errorInitializingOpenstackCli := openstack.NewOpenstackClient(
		CONF.OpenstackAuthURL,
		CONF.OpenstackUsername,
		CONF.OpenstackPassword,
		CONF.OpenstackProject,
		CONF.OpenstackDomain,
	)
	if errorInitializingOpenstackCli != nil {
		exitWithError(errorInitializingOpenstackCli, "openstack initialization")
	}

	cleanupOnExit()

	errorInitializingAPI := endpoint.Serve(
		CONF.JWTSecret,
		CONF.HTTPPort,
		&common.ClientContainer{openstackCli},
	)
	if errorInitializingAPI != nil {
		exitWithError(errorInitializingAPI)
	}
}
