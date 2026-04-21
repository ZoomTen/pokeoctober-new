module mkdata_cmd

go 1.10

// 1.10 predates the modules system, I'm just using this file to make my LSP happy
// yes I'm using the Go LSP. So convenient

replace mkdata => ./src/mkdata

replace odsutil => ./src/odsutil

require mkdata v0.0.0-00010101000000-000000000000
require odsutil v0.0.0-00010101000000-000000000000
