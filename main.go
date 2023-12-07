// Copyright (C) 2020-2023 Red Hat, Inc.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/test-network-function/cnf-certification-test/pkg/certsuite"
	"github.com/test-network-function/cnf-certification-test/pkg/versions"

	"github.com/test-network-function/cnf-certification-test/cnf-certification-test/webserver"

	"github.com/test-network-function/cnf-certification-test/internal/cli"
	"github.com/test-network-function/cnf-certification-test/internal/clientsholder"
	"github.com/test-network-function/cnf-certification-test/internal/log"
	"github.com/test-network-function/cnf-certification-test/pkg/configuration"
)

const (
	claimPathFlagKey              = "claimloc"
	CnfCertificationTestSuiteName = "CNF Certification Test Suite"
	defaultClaimPath              = "."
	defaultCliArgValue            = ""
	junitFlagKey                  = "junit"
	TNFReportKey                  = "cnf-certification-test"
	extraInfoKey                  = "testsExtraInfo"
	noLabelsExpr                  = "none"
	logFileName                   = "cnf-certsuite.log"
	logFilePermissions            = 0o644
)

const (
	labelsFlagName         = "label-filter"
	labelsFlagDefaultValue = "common"

	labelsFlagUsage = "--label-filter <expression>  e.g. --label-filter 'access-control && !access-control-sys-admin-capability'"

	timeoutFlagName         = "timeout"
	timeoutFlagDefaultvalue = 24 * time.Hour

	timeoutFlagUsage = "--timeout <time>  e.g. --timeout 30m  or -timeout 1h30m"

	listFlagName         = "list"
	listFlagDefaultValue = false

	listFlagUsage = "--list Shows all the available checks/tests. Can be filtered with --label-filter."

	serverModeFlagName         = "serverMode"
	serverModeFlagDefaultValue = false

	serverModeFlagUsage = "--serverMode or -serverMode runs in web server mode."
)

var (
	claimPath *string

	// labelsFlag holds the labels expression to filter the checks to run.
	labelsFlag     *string
	timeoutFlag    *string
	listFlag       *bool
	serverModeFlag *bool
)

func init() {
	claimPath = flag.String(claimPathFlagKey, defaultClaimPath,
		"the path where the claimfile will be output")

	labelsFlag = flag.String(labelsFlagName, labelsFlagDefaultValue, labelsFlagUsage)
	timeoutFlag = flag.String(timeoutFlagName, timeoutFlagDefaultvalue.String(), timeoutFlagUsage)
	listFlag = flag.Bool(listFlagName, listFlagDefaultValue, listFlagUsage)
	serverModeFlag = flag.Bool(serverModeFlagName, serverModeFlagDefaultValue, serverModeFlagUsage)

	flag.Parse()
	if *labelsFlag == "" {
		*labelsFlag = noLabelsExpr
	}
}

func getK8sClientsConfigFileNames() []string {
	params := configuration.GetTestParameters()
	fileNames := []string{}
	if params.Kubeconfig != "" {
		// Add the kubeconfig path
		fileNames = append(fileNames, params.Kubeconfig)
	}
	if params.Home != "" {
		kubeConfigFilePath := filepath.Join(params.Home, ".kube", "config")
		// Check if the kubeconfig path exists
		if _, err := os.Stat(kubeConfigFilePath); err == nil {
			log.Info("kubeconfig path %s is present", kubeConfigFilePath)
			// Only add the kubeconfig to the list of paths if it exists, since it is not added by the user
			fileNames = append(fileNames, kubeConfigFilePath)
		} else {
			log.Info("kubeconfig path %s is not present", kubeConfigFilePath)
		}
	}

	return fileNames
}

func createLogFile(outputDir string) (*os.File, error) {
	logFilePath := outputDir + "/" + logFileName
	err := os.Remove(logFilePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("could not delete old log file, err: %v", err)
	}

	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE, logFilePermissions)
	if err != nil {
		return nil, fmt.Errorf("could not open a new log file, err: %v", err)
	}

	return logFile, nil
}

func setupLogger(logFile *os.File) {
	logLevel, err := log.ParseLevel(configuration.GetTestParameters().LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse log level, err: %v. Defaulting to DEBUG.", err)
	}

	log.SetupLogger(logFile, logLevel)
	log.Info("Log file: %s (level=%s)", logFileName, logLevel.String())
}

//nolint:funlen
func main() {
	err := configuration.LoadEnvironmentVariables()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not load the environment variables, err: %v", err)
		os.Exit(1)
	}

	logFile, err := createLogFile(*claimPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create the log file, err: %v", err)
		os.Exit(1)
	}
	defer logFile.Close()

	setupLogger(logFile)

	log.Info("TNF Version         : %v", versions.GitVersion())
	log.Info("Claim Format Version: %s", versions.ClaimFormatVersion)
	log.Info("Labels filter       : %v", *labelsFlag)

	cli.PrintBanner()

	fmt.Printf("CNFCERT version: %s\n", versions.GitVersion())
	fmt.Printf("Claim file version: %s\n", versions.ClaimFormatVersion)
	fmt.Printf("Checks filter: %s\n", *labelsFlag)
	fmt.Printf("Output folder: %s\n", *claimPath)
	fmt.Printf("Log file: %s\n", logFileName)
	fmt.Printf("\n")

	fmt.Println("Building test environment...")
	fmt.Printf("\n")

	_ = clientsholder.GetClientsHolder(getK8sClientsConfigFileNames()...)

	certsuite.LoadChecksDB(*labelsFlag)

	if *listFlag {
		// ToDo: List all the available checks, filtered with --labels.

		fmt.Fprint(os.Stderr, "Checks listing is not implemented yet")
		os.Exit(1) //nolint:gocritic
	}

	// Diagnostic functions will run when no labels are provided.
	if *labelsFlag == noLabelsExpr {
		log.Warn("CNF Certification Suite will run in diagnostic mode so no test case will be launched.")
	}

	var timeout time.Duration
	timeout, err = time.ParseDuration(*timeoutFlag)
	if err != nil {
		log.Error("Failed to parse timeout flag %v: %v, using default timeout value %v", *timeoutFlag, err, timeoutFlagDefaultvalue)
		timeout = timeoutFlagDefaultvalue
	}

	// Set clientsholder singleton with the filenames from the env vars.
	log.Info("Output folder for the claim file: %s", *claimPath)
	if *serverModeFlag {
		log.Info("Running CNF Certification Suite in web server mode.")
		webserver.StartServer(*claimPath)
	} else {
		log.Info("Running CNF Certification Suite in stand-alone mode.")
		certsuite.Run(*labelsFlag, *claimPath, timeout)
	}
}