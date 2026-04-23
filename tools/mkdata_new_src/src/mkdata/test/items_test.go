package test

import (
	"mkdata/items"
	"odsutil"
	"os"
	"testing"
)

func TestParseItems(t *testing.T) {
  o, e := os.Open("test.ods")
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
  s, e := odsutil.ParseOds(o, size)
  if e != nil {
    t.Fatal(e)
    return
  }
  s1 := s.GetSheetByName("Items Master")
  if s1 == nil {
    t.Fatalf("no sheet found")
    return
  }
  i, e := items.GetItems(s1)
  if e != nil {
    t.Fatalf("can't get items: %s", e.Error())
    return
  }
  t.Logf("--> %s", i.Files.NormalItemConstants.String())
}
