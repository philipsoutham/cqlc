package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/philipsoutham/cqlc/generator"
)

var opts generator.Options
var parser = flags.NewParser(&opts, flags.Default)

var VERSION string = "0.9.29"

func init() {
	opts.Version = printVersionAndExit
}

func main() {

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}

	if err := generator.Generate(&opts, VERSION); err != nil {
		if err == generator.ErrInvalidOptions {
			parser.WriteHelp(os.Stderr)
			os.Exit(1)
		} else {
			log.Fatalln(err)
		}
	}
}

func printVersionAndExit() {
	fmt.Fprintf(os.Stderr, "%s %s\n", "cqlc", VERSION)
	os.Exit(0)
}
