DEF __tmhm_value__ = 1

MACRO add_tmnum
	DEF \1_TMNUM EQU __tmhm_value__
	DEF __tmhm_value__ += 1
ENDM

MACRO add_tm
; Defines three constants:
; - TM_\1: the item id, starting at $bf
; - \1_TMNUM: the learnable TM/HM flag, starting at 1
; - TM##_MOVE: alias for the move id, equal to the value of \1
	const TM_\1
	DEF TM{02d:__tmhm_value__}_MOVE = \1
	add_tmnum \1
ENDM

; see data/moves/tmhm_moves.asm for moves
	const_def 1
DEF TM01 EQU const_value
	add_tm DYNAMICPUNCH ; 01
	add_tm HEADBUTT     ; 
	add_tm CURSE        ; c4
	add_tm ROLLOUT      ; c5
	add_tm ROAR         ; c6
	add_tm TOXIC        ; c7
	add_tm ZAP_CANNON   ; c8
	add_tm ROCK_SMASH   ; c9
	add_tm PSYCH_UP     ; ca
	add_tm HIDDEN_POWER ; cb
	add_tm SUNNY_DAY    ; cc
	add_tm SWEET_SCENT  ; cd
	add_tm SNORE        ; ce
	add_tm BLIZZARD     ; cf
	add_tm HYPER_BEAM   ; d0
	add_tm ICY_WIND     ; d1
	add_tm PROTECT      ; d2
	add_tm RAIN_DANCE   ; d3
	add_tm GIGA_DRAIN   ; d4
	add_tm ENDURE       ; d5
	add_tm FRUSTRATION  ; d6
	add_tm SOLARBEAM    ; d7
	add_tm IRON_TAIL    ; d8
	add_tm DRAGONBREATH ; d9
	add_tm THUNDER      ; da
	add_tm EARTHQUAKE   ; db
	add_tm RETURN       ; dc
	add_tm DIG          ; dd
	add_tm PSYCHIC_M    ; de
	add_tm SHADOW_BALL  ; df
	add_tm MUD_SLAP     ; e0
	add_tm DOUBLE_TEAM  ; e1
	add_tm ICE_PUNCH    ; e2
	add_tm SWAGGER      ; e3
	add_tm SLEEP_TALK   ; e4
	add_tm SLUDGE_BOMB  ; e5
	add_tm SANDSTORM    ; e6
	add_tm FIRE_BLAST   ; e7
	add_tm SWIFT        ; e8
	add_tm DEFENSE_CURL ; e9
	add_tm THUNDERPUNCH ; ea
	add_tm DREAM_EATER  ; eb
	add_tm DETECT       ; ec
	add_tm REST         ; ed
	add_tm ATTRACT      ; ee
	add_tm THIEF        ; ef
	add_tm STEEL_WING   ; f0
	add_tm FIRE_PUNCH   ; f1
	add_tm FURY_CUTTER  ; f2
	add_tm NIGHTMARE    ; f3
DEF NUM_TMS EQU __tmhm_value__ - 1

MACRO add_hm
; Defines three constants:
; - HM_\1: the item id, starting at $f3
; - \1_TMNUM: the learnable TM/HM flag, starting at 51
; - HM##_MOVE: alias for the move id, equal to the value of \1
	const HM_\1
	DEF HM_VALUE = __tmhm_value__ - NUM_TMS
	DEF HM{02d:HM_VALUE}_MOVE = \1
	add_tmnum \1
ENDM

DEF HM01 EQU const_value
	add_hm CUT          ; f4
	add_hm FLY          ; f5
	add_hm SURF         ; f6
	add_hm STRENGTH     ; f7
	add_hm FLASH        ; f8
	add_hm WHIRLPOOL    ; f9
	add_hm WATERFALL    ; fa
DEF NUM_HMS EQU __tmhm_value__ - NUM_TMS - 1

MACRO add_mt
; Defines two constants:
; - \1_TMNUM: the learnable TM/HM flag, starting at 58
; - MT##_MOVE: alias for the move id, equal to the value of \1
	DEF MT_VALUE = __tmhm_value__ - NUM_TMS - NUM_HMS
	DEF MT{02d:MT_VALUE}_MOVE = \1
	add_tmnum \1
ENDM

DEF MT01 EQU const_value
	add_mt FLAMETHROWER
	add_mt THUNDERBOLT
	add_mt ICE_BEAM
DEF NUM_TUTORS = __tmhm_value__ - NUM_TMS - NUM_HMS - 1

DEF NUM_TM_HM_TUTOR EQU NUM_TMS + NUM_HMS + NUM_TUTORS
