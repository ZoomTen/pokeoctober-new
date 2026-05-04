package moves

import (
	"flag"
	"fmt"
	"mkdata/utils"
	"odsutil"
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
	o, e := odsutil.ParseOdsFromFileName(*in)
	if e != nil {
		fmt.Fprintf(os.Stderr, "cannot open file '%s': %s\n", *in, e.Error())
		os.Exit(1)
	}
	s := o.GetSheetByName("Moves")
	if s == nil {
		fmt.Fprintf(os.Stderr, "can't open sheet 'Moves'\n")
		os.Exit(1)
	}
	ss, e := GetMoves(s)
	if e != nil {
		fmt.Fprintf(os.Stderr, "can't parse sheet 'Moves': %s\n", e.Error())
		os.Exit(1)
	}
	ss.Write()
	e = utils.Wrfiles(*out, ss.Files)
	if e != nil {
		fmt.Fprintf(os.Stderr, "couldn't write item data: %s\n", e.Error())
		os.Exit(1)
	}
	fmt.Println("conversion OK")
}
