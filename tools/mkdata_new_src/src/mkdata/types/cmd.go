package types

import (
	"flag"
	"fmt"
	"os"
)

func ProcessArgs(fs *flag.FlagSet, args []string) {
	var (
		in  = fs.String("in", "", "input xlsx")
		out = fs.String("out", "", "output folder")
	)
	fs.Parse(args)
	if fs.NArg() > 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "No positional arguments allowed\n")
		fs.Usage()
		os.Exit(1)
	}
	set := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["in"] {
		fmt.Fprintf(flag.CommandLine.Output(), "Must set input file\n")
		fs.Usage()
		os.Exit(1)
	}
	if !set["out"] {
		fmt.Fprintf(flag.CommandLine.Output(), "Must set output folder\n")
		fs.Usage()
		os.Exit(1)
	}
	fmt.Println(in)
	fmt.Println(out)
	fmt.Println("conversion OK")
}
