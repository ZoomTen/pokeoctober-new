package odsutil

import (
	"fmt"
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	o, e := os.Open("Untitled.ods")
	if e != nil {
		t.Fatal(e)
		return
	}
	info, e := o.Stat()
	if e != nil {
		t.Fatal(e)
		return
	}
	size := info.Size()
	s, e := ParseOds(o, size)
	if e != nil {
		t.Fatal(e)
		return
	}
	s1 := s.GetSheetByName("Sheet1")
	if s1 == nil {
		t.Fatalf("no sheet found")
		return
	}
	fmt.Println(s1.ValueAt(0, 0))
	fmt.Println(s1.ValueAt(1, 1))
	fmt.Println(s1.ValueAt(2, 2))
	fmt.Println(s1.ValueAt(3, 3))
	fmt.Println(s1.ValueAt(4, 4))
}
