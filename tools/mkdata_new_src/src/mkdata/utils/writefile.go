package utils

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

func Wr1file(outdir string, which strings.Builder, name string) error {
	c := strings.NewReader(which.String())
	f, e := os.Create(outdir + "/" + name)
	if e != nil {
		fmt.Fprintf(os.Stderr, "couldn't create file '%s': %s\n", name, e.Error())
		return e
	}
	_, e = io.Copy(f, c)
	if e != nil {
		f.Close()
		fmt.Fprintf(os.Stderr, "couldn't write file '%s': %s\n", name, e.Error())
		return e
	}
	f.Close()
	return nil
}

/* Writes files in a struct full of string.Builders (`source`) according to
   what's written their individual `file` tags. Creates the `outdir` if it
   doesn't exist already. */
func Wrfiles(outdir string, source interface{}) error {
	e := os.MkdirAll(outdir, 0744)
	if e != nil {
		fmt.Fprintf(os.Stderr, "could not mkdir %s: %s\n", outdir, e.Error())
		return e
	}
	t := reflect.TypeOf(source)
	v := reflect.ValueOf(source)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		sb, ok := v.Field(i).Interface().(strings.Builder)
		if !ok { 
			fmt.Fprintf(os.Stderr, "field '%s' is not a strings.Builder\n", f.Name)
			continue
		}
		t := f.Tag.Get("file")
		if t == "" {
			fmt.Fprintf(os.Stderr, "field '%s' has no file tag\n", f.Name)	
			continue
		}
		e := Wr1file(outdir, sb, t)
		if e != nil { return e }
	}
	return nil
}
