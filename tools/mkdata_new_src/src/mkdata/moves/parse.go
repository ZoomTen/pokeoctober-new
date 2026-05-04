package moves

import (
	"fmt"
	"mkdata/utils"
	"odsutil"
	"os"
	"strconv"
	"strings"
)

// columns of item sheet
const (
  cMoveName = iota
  cDisplayName
  cEffectConstant
  cType
  cPower
  cAccuracy
  cPp
  cEffectChance
  cCritChance
  c_
)

func GetMoves(sheet odsutil.OdsSheet) (State, error) {
  s := State{moves: utils.NewOrderedMap()}
  if sheet.NRows() < 2 {
    return s, nil
  }
  var nowMove move
  var moveName string
  var e error
  for i := 1; i < sheet.NRows(); i++ { // skip title
    moveName = sheet.ValueAt(i, cMoveName)
    if moveName == "" { break }

    nowMove.title = sheet.ValueAt(i, cDisplayName)
    nowMove.constant = sheet.ValueAt(i, cEffectConstant)
    if nowMove.constant == "" {
      fmt.Fprintf(os.Stderr, "row %d: blank effect constant, changing to NORMAL_HIT\n", i+1)
      nowMove.constant = "NORMAL_HIT"
    }
    nowMove.movetype = sheet.ValueAt(i, cMoveName)
    nowMove.cancrit = (strings.ToLower(sheet.ValueAt(i, cCritChance)) == "yes")
    pow_ := sheet.ValueAt(i, cPower)
    if e != nil {
      e = fmt.Errorf("row %d: invalid power value '%s'", i+1, pow_)
      fmt.Fprintf(os.Stderr, "%s\n", e)
      continue
      // return s, e
    }
    acc_ := sheet.ValueAt(i, cAccuracy)
    if e != nil {
      e = fmt.Errorf("row %d: invalid accuracy value '%s'", i+1, acc_)
      fmt.Fprintf(os.Stderr, "%s\n", e)
      continue
      // return s, e
    }
    pp_ := sheet.ValueAt(i, cPp)
    if e != nil {
      e = fmt.Errorf("row %d: invalid PP value '%s'", i+1, pp_)
      fmt.Fprintf(os.Stderr, "%s\n", e)
      continue
      // return s, e
    }
    fxch_ := sheet.ValueAt(i, cEffectChance)
    if e != nil {
      e = fmt.Errorf("row %d: invalid effect chance value '%s'", i+1, fxch_)
      fmt.Fprintf(os.Stderr, "%s\n", e)
      continue
      // return s, e
    }
    pow, e := strconv.Atoi(pow_)
    if e != nil {
      e = fmt.Errorf("row %d: invalid power value '%s'", i+1, pow_)
      fmt.Fprintf(os.Stderr, "%s\n", e)
      continue
      // return s, e
    }
    nowMove.power = pow
    acc, e := strconv.Atoi(acc_)
    if e != nil {
      e = fmt.Errorf("row %d: invalid accuracy value '%s'", i+1, acc_)
      fmt.Fprintf(os.Stderr, "%s\n", e)
      continue
      // return s, e
    }
    nowMove.accuracy = acc
    pp, e := strconv.Atoi(pp_)
    if e != nil {
      e = fmt.Errorf("row %d: invalid pp value '%s'", i+1, pp_)
      fmt.Fprintf(os.Stderr, "%s\n", e)
      continue
      // return s, e
    }
    nowMove.pp = pp
    fxch, e := strconv.Atoi(fxch_)
    if e != nil {
      e = fmt.Errorf("row %d: invalid effect chance value '%s'", i+1, fxch_)
      fmt.Fprintf(os.Stderr, "%s\n", e)
      continue
      // return s, e
    }
    nowMove.fxchance = fxch
    
    s.moves.Set(moveName, nowMove)
  }
  return s, nil
}
