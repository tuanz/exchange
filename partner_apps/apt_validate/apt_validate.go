package main

import (
	"flag"
	"fmt"
	"github.com/APTrust/exchange/partner_apps/common"
	"github.com/APTrust/exchange/util/fileutil"
	"github.com/APTrust/exchange/util/storage"
	"github.com/APTrust/exchange/validation"
	"os"
	"path/filepath"
)

func main() {
	pathToConfigFile, pathToOutFile, preserveAttrs := parseCommandLine()
	configAbsPath, err := filepath.Abs(pathToConfigFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(common.EXIT_RUNTIME_ERR)
	}
	pathToBag, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(common.EXIT_RUNTIME_ERR)
	}
	conf, errors := validation.LoadBagValidationConfig(configAbsPath)
	if errors != nil && len(errors) > 0 {
		fmt.Fprintln(os.Stderr, "Could not load bag validation config: ", errors[0])
		os.Exit(common.EXIT_RUNTIME_ERR)
	}
	validator, err := validation.NewValidator(pathToBag, conf, preserveAttrs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating validator: ", err.Error())
		os.Exit(common.EXIT_RUNTIME_ERR)
	}
	summary, err := validator.Validate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "The validator encountered an error: ", err.Error())
		os.Exit(common.EXIT_RUNTIME_ERR)
	}
	exitCode := common.EXIT_OK
	if summary.HasErrors() {
		cleanup(validator.DBName())
		fmt.Println("Bag is not valid")
		fmt.Println(summary.AllErrorsAsString())
		exitCode = common.EXIT_BAG_INVALID
	} else {
		fmt.Println("Bag is valid")
	}
	if pathToOutFile != "" {
		printOutput(validator, pathToOutFile)
	}
	cleanup(validator.DBName())
	os.Exit(exitCode)
}

func printOutput(validator *validation.Validator, pathToOutFile string) {
	file, err := os.Create(pathToOutFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't open output file: %v\n", err)
		return
	}
	defer file.Close()

	db, err := storage.NewBoltDB(validator.DBName())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't open db: %v\n", err)
		return
	}

	db.DumpJson(file)
}

func cleanup(filePath string) {
	if fileutil.LooksSafeToDelete(filePath, 12, 3) {
		os.Remove(filePath)
	}
}

func parseCommandLine() (pathToConfigFile, pathToOutFile string, preserveAttrs bool) {
	var help bool
	var version bool
	flag.StringVar(&pathToConfigFile, "config", "", "Path to bag validation config file")
	flag.StringVar(&pathToOutFile, "outfile", "", "Path to file for dumping JSON output")
	flag.BoolVar(&preserveAttrs, "attrs", false, "Preserve attributes")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.BoolVar(&version, "version", false, "Show version")

	flag.Parse()

	if version {
		fmt.Println(common.GetVersion())
		os.Exit(common.EXIT_NO_OP)
	}
	if help || pathToConfigFile == "" || flag.Arg(0) == "" {
		printUsage()
		os.Exit(common.EXIT_USER_ERR)
	}
	return pathToConfigFile, pathToOutFile, preserveAttrs
}

// Tell the user about the program.
func printUsage() {
	message := `
apt_validate validates bags according to the specified config file.

Usage:

apt_validate --config=<config_file> \
             [--attrs=<true|false>] \
             [--outfile=<path_to_output_file>] \
             path_to_bag

apt_validate --help
apt_validate --version

Options

--attrs option is not required. It indicates whether you want to preserve
detailed information in the JSON output when parsing the bag. This uses
more memory, and is generally not useful to users outside of APTrust.
The additional information generated by the --attrs flag is used by APTrust
during the ingest proces. Timestamps and UUIDs change each time you run
the validator.

--config is required and should be the path to a bag validation
config file that describes the validation rules. An example can be found at
https://github.com/APTrust/exchange/blob/master/config/aptrust_bag_validation_config.json
but the config file must exist on the local drive.

--help prints this help message and exits.

--outfile option is not required. If specified, the validator will dump
JSON information about the bag and its contents to this file. That info may be
useful, especially when combined with --attrs=true, in cases where you're trying
to debug your bagging process.

--version prints version info and exits.

Arguments

The path_to_bag parameter is required. It should be the absolute path
to the directory containing the untarred bag, or to a tarred bag file.

Exit codes:

0 - Bag is valid
1 - Validation could not be completed, typically because of a problem
	finding or reading the config file, or finding or reading the bag.
2 - Validation completed and bag is invalid.
3 - Operation could not be completed due to usage error (e.g. missing params)

`
	fmt.Println(message)
}
