Flypoints:
; entries correspond to FLY_* constants
; Johto
	; landmark, spawn point
	table_width 2
	db LANDMARK_NEW_BARK_TOWN,    SPAWN_NEW_BARK
; Kanto
; TODO: SPAWN_INDIGO
	db LANDMARK_INDIGO_PLATEAU,   SPAWN_N_A
	assert_table_length NUM_FLYPOINTS
	db -1 ; end
