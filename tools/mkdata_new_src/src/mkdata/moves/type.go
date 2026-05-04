package moves

import (
	"mkdata/utils"
	"strings"
)

type move struct {
  title string
  constant string
  movetype string
  power int
  accuracy int
  pp int
  fxchance int // 0 to 100
  cancrit bool
}

type State struct {
  moves *utils.OrderedMap
  
  // write
  Files Files
}

type Files struct {
  // const XXX
  MoveConstants strings.Builder `file:"move_constants.asm"`
  // li "XXX"
  MoveNames strings.Builder `file:"move_names.asm"`
  // MoveDescriptions: dw ...
  MoveDesc strings.Builder `file:"move_description_pointers.asm"`
  // Moves: move ...
  MoveAttributes strings.Builder `file:"move_attributes.asm"`
  // BattleAnimations:: dw ...
  MoveAnimations strings.Builder `file:"move_animation_pointers.asm"`
  // CriticalHitMoves: dw ...
  CritMoves strings.Builder `file:"critical_hit_moves.asm"`
}
