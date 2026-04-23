package items

import (
	"mkdata/utils"
	"strings"
)

type item struct {
  title              string
  price              int
  heldEffect         string
  property           string
  fieldSelectAction  string
  battleSelectAction string
  fieldEffect        string
  parameter          int
}

type State struct {
  // pocket
  item *utils.OrderedMap
  key *utils.OrderedMap
  ball *utils.OrderedMap
  
  // write
  Files Files
}

type Files struct {
  // const XXX
  NormalItemConstants strings.Builder `file:"normal_item_constants.asm"`
  // ItemNames:: db ...
  NormalItemNames strings.Builder `file:"normal_item_names.asm"`
  // ItemAttributes1: item_attribute ...
  NormalItemAttributes strings.Builder `file:"normal_item_attributes.asm"`
  // ItemEffects1: dw ...
  NormalItemEffects strings.Builder `file:"normal_item_effect_pointers.asm"`
  // ItemDescriptions1: dw ...
  NormalItemDesc strings.Builder `file:"normal_item_description_pointers.asm"`

  // const XXX
  KeyItemConstants strings.Builder `file:"key_item_constants.asm"`
  // db ...
  KeyItemNames strings.Builder `file:"key_item_names.asm"`
  // KeyItemAttributes: item_attribute ...
  KeyItemAttributes strings.Builder `file:"key_item_attributes.asm"`
  // KeyItemEffects: dw ...
  KeyItemEffects strings.Builder `file:"key_item_effect_pointers.asm"`
  // KeyItemDescriptions: dw ...
  KeyItemDesc strings.Builder `file:"key_item_description_pointers.asm"`


  // const XXX
  BallItemConstants strings.Builder `file:"ball_item_constants.asm"`
  // db ...
  BallItemNames strings.Builder `file:"ball_item_names.asm"`
  // BallItemAttributes: item_attribute ...
  BallItemAttributes strings.Builder `file:"ball_item_attributes.asm"`
  // BallItemEffects: dw ...
  BallItemEffects strings.Builder `file:"ball_item_effect_pointers.asm"`
  // BallItemDescriptions: dw ...
  BallItemDesc strings.Builder `file:"ball_item_description_pointers.asm"`
}
