package items

import (
	"fmt"
	"mkdata/utils"
	"odsutil"
	"strconv"
)

// columns of item sheet
const (
  ciItemName = iota
  ciDisplayName
  ciPocket
  ciPrice
  ciHeldEffect
  ciProperty
  ciFieldMenu
  ciBattleMenu
  ciFieldEffect
  ciParameter
  ci_
)

func GetItems(sheet odsutil.OdsSheet) (State, error) {
  s := State{
    item: utils.NewOrderedMap(),
    key: utils.NewOrderedMap(),
    ball: utils.NewOrderedMap(),
  }
  if sheet.NRows() < 2 {
    return s, nil
  }
  var whichPocket *utils.OrderedMap
  var pocketVal string
  var itemName string
  var e error
  for i := 1; i < sheet.NRows(); i++ { // skip title
    itemName = sheet.ValueAt(i, ciItemName)
    if itemName == "" { break }

    pocketVal = sheet.ValueAt(i, ciPocket)

    switch pocketVal {
    case "Item":
      whichPocket = s.item
    case "Key Item":
      whichPocket = s.key
    case "Ball":
      whichPocket = s.ball
    default:
      e =  fmt.Errorf("row %d: invalid pocket '%s'", i+1, pocketVal)
      return s, e
    }

    item := item{}
    item.title = sheet.ValueAt(i, ciDisplayName)
    item.heldEffect = sheet.ValueAt(i, ciHeldEffect)
    item.property = sheet.ValueAt(i, ciProperty)
    item.fieldSelectAction = sheet.ValueAt(i, ciFieldMenu)
    item.battleSelectAction = sheet.ValueAt(i, ciBattleMenu)
    item.fieldEffect = sheet.ValueAt(i, ciFieldEffect)
    item.price, e = strconv.Atoi(sheet.ValueAt(i, ciPrice))
    if e != nil {
      e = fmt.Errorf("row %d: cannot parse price (%s)", i+1, e.Error())
      return s, e
    }
    item.parameter, e = strconv.Atoi(sheet.ValueAt(i, ciParameter))
    if e != nil {
      e = fmt.Errorf("row %d: cannot parse param (%s)", i+1, e.Error())
      return s, e
    }
    whichPocket.Set(itemName, item)
  }
  return s, nil
}
