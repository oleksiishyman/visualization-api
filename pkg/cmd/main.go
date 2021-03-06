package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"visualization-api/pkg/config"
	"visualization-api/pkg/logging"
)

var (
	logRotate  *log.RotateWriter
	version    = "UNDEFINED"
	gitVersion = "UNDEFINED"
)

func exitWithError(err error) {
	fmt.Println(err)
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
		4 - initialize signals handler, to close file in rotation logger
	*/

	// initialize config
	errorParsingConfig := config.InitializeConfig()
	if errorParsingConfig != nil {
		exitWithError(errorParsingConfig)
	}
	CONF := config.GetConfig()

	if CONF.ShowVersion {
		fmt.Printf("visualization-api version %s %s \n", version, gitVersion)
		os.Exit(0)
	}

	// create rotation logger
	var rotateInitError error
	logRotate, rotateInitError = log.NewRotateWriter(CONF.LogFilePath)
	if rotateInitError != nil {
		exitWithError(rotateInitError)
	}

	// initialize logger
	log.InitializeLogger(logRotate, CONF.ConsoleDebug, CONF.LogLevel)

	cleanupOnExit()
}
