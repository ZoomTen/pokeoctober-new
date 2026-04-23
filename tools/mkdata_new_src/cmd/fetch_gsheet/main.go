package main

import (
	"fmt"
	"io"
	"odsutil"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <doc ID> <output.ods>\n", os.Args[0])
		os.Exit(1)
	}

	docId := os.Args[1]
	oname := os.Args[2]

	e := odsutil.FetchUrlAndPipe(
		fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=ods", docId),
		mkFunc(oname),
	)
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
	fmt.Println("Download OK")
}

func mkFunc(oname string) func(io.ReaderAt, int64)error{
	return func(r io.ReaderAt, s int64) error {
		ofile, e := os.OpenFile(
			oname,
			os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
			0755,
		)
		if e != nil {
			return e
		}
		defer ofile.Close()
		return odsutil.ResetZipTimestamps(r, s, ofile)
	}
}
