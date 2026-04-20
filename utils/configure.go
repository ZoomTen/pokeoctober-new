/*========== Configure script ==========
  This generates a build directory for use with the Ninja build system.
  ```
  go run utils/configure.go
  cd build
  ninja
  ```

  Minimum supported Go version: 1.10.
  If `gopls` warns about "modernizing", ignore it. */

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

/*========== Runtime options ==========
  When invoking `configure`, you can set these flags to pick where to find
  and place things. */

var srcDir = flag.String("src", ".", "Source directory")
var buildDir = flag.String("build", "build", "Build directory")
var toolsDir = flag.String("tools", "tools", "Directory with precompiled tools")

/*========== Main source files ==========
  These are the main source files that the project uses. They will each be
  compiled into their own .o files.

  `scan_includes` will be called on each of them to determine every single dependency.

  The scanned dependencies are then subject to `MakeRule` to determine whether or not
  a dependency is one that needs to be created. */

var asmFiles = []string{
  "audio.asm",
  "home.asm",
  "main.asm",
  "ram.asm",
  "data/text/common.asm",
  "data/maps/map_data.asm",
  "data/pokemon/dex_entries.asm",
  "data/pokemon/egg_moves.asm",
  "data/pokemon/evos_attacks.asm",
  "engine/movie/credits.asm",
  "engine/overworld/events.asm",
  "gfx/misc.asm",
  "gfx/pics.asm",
  "gfx/sprites.asm",
  "gfx/tilesets.asm",
  "lib/mobile/main.asm",
  "lib/mobile/mail.asm",
}

/*========== Needed tools ==========
  To add a required tool in here, simply add its EXACT NAME here.
  It will be searched in `toolsDir`` first, then in the system PATH. */

var tools = []string{
  "rgbasm",
  "rgbgfx",
  "rgblink",
  "rgbfix",
  "scan_includes",
  "png_dimensions",
  "pokemon_animation",
  "pokemon_animation_graphics",
  "gfx",
  "lzcomp",
  "cat",
  "tr",
  "gbcpal",
  "stadium",
}

/*========== Rule matching ==========
  `MakeRule` returns the build rule for a dependency.

  `depName` represents an arbitrary dependency. Since we're inheriting a build system
  that intermingles compiled files with that of source files, whether or not
  dependencies come from the source directory or the build directory cannot be known
  from the outset, and is determined dynamically, using `fileExists`:

    "if it's not present in the source directory, this must be something
     that is built"

  Resolving such things is deferred until `resolveTargets`, in which all paths are
  transformed into absolute paths.

  Even so, since flags are optional, freeform, and would add more checks to the script,
  they are exempt from this procedure and so `srcDir` or `buildDir` can be appended
  manually to them, which is why both are specified in this function's parameters. */

/*  Return values for `MakeRule`:
    - nil        => not yet resolvable
    - &Stmt{}    => no rule needed
    - &Stmt{...} => resolved */
func MakeRule(dep string, targets map[string]*Stmt, srcDir, buildDir string) (*Stmt, []string) {
  /* First, the Very Special™ cases.
     This is for specific files; there are whole build statements for them. */
  switch dep {
  case "gfx/sgb/sgb_border.sgb.tilemap":
    return refineRule(Stmt{
      rule:   "SGB_TILEMAP",
      output: dep,
      inputs: []string{"gfx/sgb/sgb_border.bin"},
      flags:  map[string]string{},
    }, targets, srcDir)
  case "gfx/trade/game_boy_cable.2bpp":
    return refineRule(Stmt{
      rule:   "CAT",
      output: dep,
      inputs: []string{
        "gfx/trade/game_boy.2bpp",
        "gfx/trade/link_cable.2bpp",
      },
      flags: map[string]string{},
    }, targets, srcDir)
  /* Egg has no normal.gbcpal; its front sprite uses front.gbcpal directly. */
  case "gfx/pokemon/egg/front.2bpp":
    return refineRule(Stmt{
      rule:           "2BPP_PAL",
      output:         dep,
      inputs:         []string{"gfx/pokemon/egg/front.png"},
      implicitInputs: []string{"gfx/pokemon/egg/front.gbcpal"},
      flags: map[string]string{
        "palettefile": fmt.Sprintf("%s/gfx/pokemon/egg/front.gbcpal", buildDir),
      },
    }, targets, srcDir)
  /* All unown variants share a single normal.gbcpal built from every variant's palettes. */
  case "gfx/pokemon/unown/normal.gbcpal":
    inputs := make([]string, 0, 52)
    for c := 'a'; c <= 'z'; c++ {
      inputs = append(inputs,
        fmt.Sprintf("gfx/pokemon/unown_%c/front.gbcpal", c),
        fmt.Sprintf("gfx/pokemon/unown_%c/back.gbcpal", c),
      )
    }
    return refineRule(Stmt{
      rule:   "GBCPAL_CHECK",
      output: dep,
      inputs: inputs,
      flags:  map[string]string{},
    }, targets, srcDir)
  case "gfx/title/crystal.2bpp":
    return refineRule(Stmt{
      rule:   "2BPP_GFX",
      output: dep,
      inputs: []string{"gfx/title/crystal.png"},
      flags: map[string]string{
        "gfx": fmt.Sprintf("--interleave --png=%s/gfx/title/crystal.png", srcDir),
      },
    }, targets, srcDir)
  }

  /* Then, the semi-special cases. */
  
  /* The base unown/ directory has no sprites.
     Only the variant directories (unown_a/, ...) do. */
  if strings.HasPrefix(dep, "gfx/pokemon/unown/") && dep != "gfx/pokemon/unown/normal.gbcpal" {
    return &Stmt{}, []string{}
  }
  /* Animation frames and tilemaps are generated
     from the graphics and dimensions files. */
  if m := rePkmnAnimGfx.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "PKMN_ANIMATION_GFX",
      output: dep,
      inputs: []string{
        fmt.Sprintf("gfx/pokemon/%s/front.2bpp", m[1]),
        fmt.Sprintf("gfx/pokemon/%s/front.dimensions", m[1]),
      },
      flags: map[string]string{},
    }, targets, srcDir)
  }
  if m := rePkmnAnimTmap.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "PKMN_ANIMATION_TMAP",
      output: dep,
      inputs: []string{
        fmt.Sprintf("gfx/pokemon/%s/front.2bpp", m[1]),
        fmt.Sprintf("gfx/pokemon/%s/front.dimensions", m[1]),
      },
      flags: map[string]string{},
    }, targets, srcDir)
  }
  /* Animation "deltas" are calculated from the built tilemaps. */
  if m := rePkmnBitmask.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "PKMN_ANIMATION_BITMASK",
      output: dep,
      inputs: []string{
        fmt.Sprintf("gfx/pokemon/%s/front.animated.tilemap", m[1]),
        fmt.Sprintf("gfx/pokemon/%s/front.dimensions", m[1]),
      },
      flags: map[string]string{},
    }, targets, srcDir)
  }
  if m := rePkmnFrames.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "PKMN_ANIMATION_FRAMES",
      output: dep,
      inputs: []string{
        fmt.Sprintf("gfx/pokemon/%s/front.animated.tilemap", m[1]),
        fmt.Sprintf("gfx/pokemon/%s/front.dimensions", m[1]),
      },
      flags: map[string]string{},
    }, targets, srcDir)
  }
  /* Unown variants share one normal.gbcpal from the base unown/ directory.
     These must precede the general gfx/pokemon/(.+)/back|front patterns,
     since those would otherwise match and generate a per-variant path that
     doesn't exist. */
  if m := rePkmnUnownBack.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:           "2BPP_PAL",
      output:         dep,
      inputs:         []string{fmt.Sprintf("gfx/pokemon/%s/back.png", m[1])},
      implicitInputs: []string{"gfx/pokemon/unown/normal.gbcpal"},
      flags: map[string]string{
        "palettefile": fmt.Sprintf("%s/gfx/pokemon/unown/normal.gbcpal", buildDir),
        "rgbgfx":      "--columns",
      },
    }, targets, srcDir)
  }
  if m := rePkmnUnownFront.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:           "2BPP_PAL",
      output:         dep,
      inputs:         []string{fmt.Sprintf("gfx/pokemon/%s/front.png", m[1])},
      implicitInputs: []string{"gfx/pokemon/unown/normal.gbcpal"},
      flags: map[string]string{
        "palettefile": fmt.Sprintf("%s/gfx/pokemon/unown/normal.gbcpal", buildDir),
      },
    }, targets, srcDir)
  }
  /* General pokemon back/front sprites. */
  if m := rePkmnBack.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:           "2BPP_PAL",
      output:         dep,
      inputs:         []string{fmt.Sprintf("gfx/pokemon/%s/back.png", m[1])},
      implicitInputs: []string{fmt.Sprintf("gfx/pokemon/%s/normal.gbcpal", m[1])},
      flags: map[string]string{
        "palettefile": fmt.Sprintf("%s/gfx/pokemon/%s/normal.gbcpal", buildDir, m[1]),
        "rgbgfx":      "--columns",
      },
    }, targets, srcDir)
  }
  if m := rePkmnFront.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:           "2BPP_PAL",
      output:         dep,
      inputs:         []string{fmt.Sprintf("gfx/pokemon/%s/front.png", m[1])},
      implicitInputs: []string{fmt.Sprintf("gfx/pokemon/%s/normal.gbcpal", m[1])},
      flags: map[string]string{
        "palettefile": fmt.Sprintf("%s/gfx/pokemon/%s/normal.gbcpal", buildDir, m[1]),
      },
    }, targets, srcDir)
  }
  if m := rePkmnGbcpal.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "GBCPAL_CHECK",
      output: dep,
      inputs: []string{
        fmt.Sprintf("gfx/pokemon/%s/front.gbcpal", m[1]),
        fmt.Sprintf("gfx/pokemon/%s/back.gbcpal", m[1]),
      },
      flags: flagsFor(dep),
    }, targets, srcDir)
  }
  /* Trainer sprites use a per-trainer .gbcpal and are always column-ordered. */
  if m := reTrainer.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:           "2BPP_PAL",
      output:         dep,
      inputs:         []string{fmt.Sprintf("gfx/trainers/%s.png", m[1])},
      implicitInputs: []string{fmt.Sprintf("gfx/trainers/%s.gbcpal", m[1])},
      flags: map[string]string{
        "palettefile": fmt.Sprintf("%s/gfx/trainers/%s.gbcpal", buildDir, m[1]),
        "rgbgfx":      "--columns",
      },
    }, targets, srcDir)
  }
  /* Title graphics with special interleaving.
     crystal.2bpp already handled above. */
  if m := reTitleInterleave.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "2BPP",
      output: dep,
      inputs: []string{fmt.Sprintf("gfx/title/%s.png", m[1])},
      flags: map[string]string{
        "gfx": fmt.Sprintf("--interleave --png=%s/gfx/title/%s.png", srcDir, m[1]),
      },
    }, targets, srcDir)
  }
  /* Slot machine sttuff. */
  if m := reSlots.FindStringSubmatch(dep); m != nil {
    gfxFlags := fmt.Sprintf("--interleave --png=%s/gfx/slots/%s.png", srcDir, m[1])
    if m[1] == "slots_3" {
      gfxFlags += " --remove-duplicates --keep-whitespace --remove-xflip"
    }
    return refineRule(Stmt{
      rule:   "2BPP_GFX",
      output: dep,
      inputs: []string{fmt.Sprintf("gfx/slots/%s.png", m[1])},
      flags:  map[string]string{"gfx": gfxFlags},
    }, targets, srcDir)
  }

  /* Next up, the general cases, with a LOT of edges. The edges in question take
     the form of `fileFlags`, a big look-up table of file-specific flag overrides. */

  /* General 2bpp/1bpp graphics.
     A `gfx` flag in fileFlags means a post-processing step is needed, which is encoded
     as a separate rule variant (_GFX suffix). */
  if m := re2bpp.FindStringSubmatch(dep); m != nil {
    flags := flagsFor(dep)
    rule := "2BPP"
    if _, hasGfx := flags["gfx"]; hasGfx { rule = "2BPP_GFX" }
    return refineRule(Stmt{
      rule:   rule,
      output: dep,
      inputs: []string{fmt.Sprintf("%s.png", m[1])},
      flags:  flags,
    }, targets, srcDir)
  }
  if m := re1bpp.FindStringSubmatch(dep); m != nil {
    flags := flagsFor(dep)
    rule := "1BPP"
    if _, hasGfx := flags["gfx"]; hasGfx { rule = "1BPP_GFX" }
    return refineRule(Stmt{
      rule:   rule,
      output: dep,
      inputs: []string{fmt.Sprintf("%s.png", m[1])},
      flags:  flags,
    }, targets, srcDir)
  }
  if m := reLz.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "LZ",
      output: dep,
      inputs: []string{m[1]},
      flags:  map[string]string{},
    }, targets, srcDir)
  }
  if m := reDimensions.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "DIMENSIONS",
      output: dep,
      inputs: []string{fmt.Sprintf("%s.png", m[1])},
      flags:  map[string]string{},
    }, targets, srcDir)
  }
  if m := reGbcpal.FindStringSubmatch(dep); m != nil {
    return refineRule(Stmt{
      rule:   "GBCPAL",
      output: dep,
      inputs: []string{fmt.Sprintf("%s.png", m[1])},
      flags:  flagsFor(dep),
    }, targets, srcDir)
  }

  /* Everything else explicitly ignored */
  return &Stmt{}, []string{}
}

/* The big look-up table of file-specific flags in question. */
var fileFlags = map[string]map[string]string{
  /* Pokemon / trainer sprites */
  "gfx/pokemon/egg/unused_front.2bpp":    {"rgbgfx": "--columns"},
  "gfx/player/chris.2bpp":                {"rgbgfx": "--columns"},
  "gfx/player/chris_back.2bpp":           {"rgbgfx": "--columns"},
  "gfx/player/kris.2bpp":                 {"rgbgfx": "--columns"},
  "gfx/player/kris_back.2bpp":            {"rgbgfx": "--columns"},
  "gfx/trainer_card/chris_card.2bpp":     {"rgbgfx": "--columns"},
  "gfx/trainer_card/kris_card.2bpp":      {"rgbgfx": "--columns"},
  "gfx/trainer_card/leaders.2bpp":        {"gfx": "--trim-whitespace"},
  "gfx/battle/dude.2bpp":                 {"rgbgfx": "--columns"},
  "gfx/new_game/shrink1.2bpp":            {"rgbgfx": "--columns"},
  "gfx/new_game/shrink2.2bpp":            {"rgbgfx": "--columns"},
  "gfx/pokedex/question_mark.2bpp":       {"rgbgfx": "--columns"},
  "gfx/pokegear/pokegear.2bpp":           {"rgbgfx": "--trim-end 2"},

  /* Post-processing by `tools/gfx` */
  "gfx/mail/dragonite.1bpp":              {"gfx": "--remove-whitespace"},
  "gfx/mail/large_note.1bpp":             {"gfx": "--remove-whitespace"},
  "gfx/mail/surf_mail_border.1bpp":       {"gfx": "--remove-whitespace"},
  "gfx/mail/flower_mail_border.1bpp":     {"gfx": "--remove-whitespace"},
  "gfx/mail/litebluemail_border.1bpp":    {"gfx": "--remove-whitespace"},
  "gfx/font/unused_bold_font.1bpp":       {"gfx": "--trim-whitespace"},
  "gfx/pokedex/pokedex.2bpp":             {"gfx": "--trim-whitespace"},
  "gfx/pokedex/pokedex_sgb.2bpp":         {"gfx": "--trim-whitespace"},
  "gfx/pokedex/slowpoke.2bpp":            {"gfx": "--trim-whitespace"},
  "gfx/pokegear/pokegear_sprites.2bpp":   {"gfx": "--trim-whitespace"},
  "gfx/mystery_gift/mystery_gift.2bpp":   {"gfx": "--trim-whitespace"},
  "gfx/title/logo.2bpp":                  {"rgbgfx": "--trim-end 4"},
  "gfx/trade/ball.2bpp":                  {"gfx": "--remove-whitespace"},
  "gfx/trade/game_boy.2bpp":              {"gfx": "--remove-duplicates --preserve=0x23,0x27"},
  "gfx/slots/slots_1.2bpp":               {"gfx": "--trim-whitespace"},
  "gfx/card_flip/card_flip_1.2bpp":       {"gfx": "--trim-whitespace"},
  "gfx/card_flip/card_flip_2.2bpp":       {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/angels.2bpp":         {"gfx": "--trim-whitespace"},
  "gfx/battle_anims/beam.2bpp":           {"gfx": "--remove-xflip --remove-yflip --remove-whitespace"},
  "gfx/battle_anims/bubble.2bpp":         {"gfx": "--trim-whitespace"},
  "gfx/battle_anims/charge.2bpp":         {"gfx": "--trim-whitespace"},
  "gfx/battle_anims/egg.2bpp":            {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/explosion.2bpp":      {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/hit.2bpp":            {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/horn.2bpp":           {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/lightning.2bpp":      {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/misc.2bpp":           {"gfx": "--remove-duplicates --remove-xflip"},
  "gfx/battle_anims/noise.2bpp":          {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/objects.2bpp":        {"gfx": "--remove-whitespace --remove-xflip"},
  "gfx/battle_anims/pokeball.2bpp":       {"gfx": "--remove-xflip --keep-whitespace"},
  "gfx/battle_anims/reflect.2bpp":        {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/rocks.2bpp":          {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/skyattack.2bpp":      {"gfx": "--remove-whitespace"},
  "gfx/battle_anims/status.2bpp":         {"gfx": "--remove-whitespace"},
  "gfx/overworld/chris_fish.2bpp":        {"gfx": "--trim-whitespace"},
  "gfx/overworld/kris_fish.2bpp":         {"gfx": "--trim-whitespace"},
  "gfx/sprites/big_onix.2bpp":            {"gfx": "--remove-whitespace --remove-xflip"},
  "gfx/sgb/sgb_border.2bpp":              {"gfx": "--trim-whitespace"},
  "gfx/mobile/ascii_font.2bpp":           {"gfx": "--trim-whitespace"},
  "gfx/mobile/dialpad.2bpp":              {"gfx": "--trim-whitespace"},
  "gfx/mobile/dialpad_cursor.2bpp":       {"gfx": "--trim-whitespace"},
  "gfx/mobile/electro_ball.2bpp":         {"gfx": "--remove-duplicates --remove-xflip --preserve=0x39"},
  "gfx/mobile/mobile_splash.2bpp":        {"gfx": "--remove-duplicates --remove-xflip"},
  "gfx/mobile/card.2bpp":                 {"gfx": "--trim-whitespace"},
  "gfx/mobile/card_2.2bpp":               {"gfx": "--trim-whitespace"},
  "gfx/mobile/card_folder.2bpp":          {"gfx": "--trim-whitespace"},
  "gfx/mobile/phone_tiles.2bpp":          {"gfx": "--remove-whitespace"},
  "gfx/mobile/pichu_animated.2bpp":       {"gfx": "--trim-whitespace"},
  "gfx/mobile/stadium2_n64.2bpp":         {"gfx": "--trim-whitespace"},

  /* Flag overrides for `tools/gbcpal` */
  "gfx/pokemon/spearow/normal.gbcpal":    {"gbcpal": "--reverse"},
  "gfx/pokemon/fearow/normal.gbcpal":     {"gbcpal": "--reverse"},
  "gfx/pokemon/farfetch_d/normal.gbcpal": {"gbcpal": "--reverse"},
  "gfx/pokemon/hitmonlee/normal.gbcpal":  {"gbcpal": "--reverse"},
  "gfx/pokemon/scyther/normal.gbcpal":    {"gbcpal": "--reverse"},
  "gfx/pokemon/jynx/normal.gbcpal":       {"gbcpal": "--reverse"},
  "gfx/pokemon/porygon/normal.gbcpal":    {"gbcpal": "--reverse"},
  "gfx/pokemon/porygon2/normal.gbcpal":   {"gbcpal": "--reverse"},
  "gfx/trainers/swimmer_m.gbcpal":        {"gbcpal": "--reverse"},
}

/* Pre-compiled patterns for `MakeRule``. */
var (
  rePkmnAnimGfx    = regexp.MustCompile(`gfx/pokemon/(.+)/front\.animated\.2bpp$`)
  rePkmnAnimTmap   = regexp.MustCompile(`gfx/pokemon/(.+)/front\.animated\.tilemap$`)
  rePkmnBitmask    = regexp.MustCompile(`gfx/pokemon/(.+)/bitmask\.asm$`)
  rePkmnFrames     = regexp.MustCompile(`gfx/pokemon/(.+)/frames\.asm$`)
  rePkmnUnownBack  = regexp.MustCompile(`gfx/pokemon/(unown_[^/]+)/back\.2bpp$`)
  rePkmnUnownFront = regexp.MustCompile(`gfx/pokemon/(unown_[^/]+)/front\.2bpp$`)
  rePkmnBack       = regexp.MustCompile(`gfx/pokemon/(.+)/back\.2bpp$`)
  rePkmnFront      = regexp.MustCompile(`gfx/pokemon/(.+)/front\.2bpp$`)
  rePkmnGbcpal     = regexp.MustCompile(`gfx/pokemon/(.+)/normal\.gbcpal$`)
  reTrainer        = regexp.MustCompile(`gfx/trainers/(.+)\.2bpp$`)
  reTitleInterleave = regexp.MustCompile(`gfx/title/(crystal|old_fg)\.2bpp$`)
  reSlots          = regexp.MustCompile(`gfx/slots/(slots_2|slots_3)\.2bpp$`)
  re2bpp           = regexp.MustCompile(`(.+)\.2bpp$`)
  re1bpp           = regexp.MustCompile(`(.+)\.1bpp$`)
  reLz             = regexp.MustCompile(`(.+)\.lz$`)
  reDimensions     = regexp.MustCompile(`(.+)\.dimensions$`)
  reGbcpal         = regexp.MustCompile(`(.+)\.gbcpal$`)
)

/*========== Script entry point ==========
  The thing that runs when you execute `configure.go`.

  This is also where the actual `rules` are defined, since in this process
  the tools are found. If I were to place it as a global variable, the tools
  to be used are unknowable, and I would need to set it again anyway. */

var reAsmToObj = regexp.MustCompile(`(.+)\.asm$`)
func main() {
  /* This enables the `-h` handler for help.
     It stops the program on this handler, so you can read up what you can configure. */
  flag.Parse()

  /* First, let's "canonize" the directories in use.
     This will hopefully eliminate any ambiguities later. Whereas
     `srcDir` `buildDir` `toolsDir` are relative to the current working directory,
     they get resolved to absolute paths in `sd` `bd` and `td`. */
  sd, e := filepath.Abs(*srcDir)
  if e != nil { eexit("can't get srcdir: " + e.Error()) }
  bd, e := filepath.Abs(*buildDir)
  if e != nil { eexit("can't get build dir: " + e.Error()) }
  td, e := filepath.Abs(*toolsDir)
  if e != nil { eexit("can't get tools dir: " + e.Error()) }

  /* Index all source files upfront to avoid repeated os.Stat calls later. */
  fmt.Fprintln(os.Stderr, "scanning source directory")
  filepath.Walk(sd, func(path string, info os.FileInfo, err error) error {
    if err != nil { return nil }
    if info.IsDir() {
      abs, _ := filepath.Abs(info.Name())
      if info.Name() == ".git" { return filepath.SkipDir }
      /* skip build dir if nested in src */
      if abs == bd             { return filepath.SkipDir }
    } else {
      fileExistence[path] = struct{}{}
    }
    return nil
  })

  /* Find all tools. */
  foundTools, e := FindTools(td)
  if e != nil { eexit("cannot find tool: " + e.Error()) }

  /* Now that the tools are known, we can construct the actual rules. */
  rules := map[string]*Rule{
    "LZ": {name: "LZ",
      command: fmt.Sprintf(
        "%s -- $in $out",
        foundTools["lzcomp"]),
    },
    "CAT": {name: "CAT",
      command: fmt.Sprintf(
        "%s $in > $out",
        foundTools["cat"]),
    },
    "LINK_AND_FIX": {name: "LINK_AND_FIX",
      command: fmt.Sprintf(
        "%s $rgblink -l $layoutfile -n $symfile -m $mapfile -o $out $in && " + 
        "%s $rgbfix $out && " +
        "%s $out",
        foundTools["rgblink"],
        foundTools["rgbfix"],
        foundTools["stadium"]),
    },
    "PKMN_ANIMATION_BITMASK": {name: "PKMN_ANIMATION_BITMASK",
      command: fmt.Sprintf(
        "%s -b $in > $out",
        foundTools["pokemon_animation"]),
    },
    "PKMN_ANIMATION_FRAMES": {name: "PKMN_ANIMATION_FRAMES",
      command: fmt.Sprintf(
        "%s -f $in > $out",
        foundTools["pokemon_animation"]),
    },
    "PKMN_ANIMATION_GFX": {name: "PKMN_ANIMATION_GFX",
      command: fmt.Sprintf(
        "%s -o $out $in",
        foundTools["pokemon_animation_graphics"]),
    },
    "PKMN_ANIMATION_TMAP": {name: "PKMN_ANIMATION_TMAP",
      command: fmt.Sprintf(
        "%s -t $out $in",
        foundTools["pokemon_animation_graphics"]),
    },
    "DIMENSIONS": {name: "DIMENSIONS",
      command: fmt.Sprintf(
        "%s $in $out",
        foundTools["png_dimensions"]),
    },
    "SGB_TILEMAP": {name: "SGB_TILEMAP",
      command: fmt.Sprintf(
        "%s < $in -d '\\000' > $out",
        foundTools["tr"]),
    },
    "ASM": {name: "ASM",
      command: fmt.Sprintf(
        "%s -Weverything -Wtruncation=1 $rgbasm -o $out $in",
        foundTools["rgbasm"]),
    },
    "GBCPAL": {name: "GBCPAL",
      command: fmt.Sprintf(
        "%s -p $out $in && %s $gbcpal $out $out",
        foundTools["rgbgfx"], foundTools["gbcpal"]),
    },
    "GBCPAL_CHECK": {name: "GBCPAL_CHECK",
      command: fmt.Sprintf(
        "%s $gbcpal $out $in",
        foundTools["gbcpal"]),
    },
    "2BPP": {name: "2BPP",
      command: fmt.Sprintf(
        "%s --colors dmg $rgbgfx -o $out $in",
        foundTools["rgbgfx"]),
    },
    "2BPP_GFX": {name: "2BPP_GFX",
      command: fmt.Sprintf(
        "%s --colors dmg $rgbgfx -o $out $in && %s $gfx -o $out $out",
        foundTools["rgbgfx"], foundTools["gfx"]),
    },
    "2BPP_PAL": {name: "2BPP_PAL",
      command: fmt.Sprintf(
        "%s --colors gbc:$palettefile $rgbgfx -o $out $in",
        foundTools["rgbgfx"]),
    },
    "1BPP": {name: "1BPP",
      command: fmt.Sprintf(
        "%s --colors dmg $rgbgfx --depth 1 -o $out $in",
        foundTools["rgbgfx"]),
    },
    "1BPP_GFX": {name: "1BPP_GFX",
      command: fmt.Sprintf(
        "%s --colors dmg $rgbgfx --depth 1 -o $out $in && %s $gfx --depth 1 -o $out $out",
        foundTools["rgbgfx"], foundTools["gfx"]),
    },
  }

  /* Show the current status before we begin configuring. */
  fmt.Fprintf(os.Stderr, "Source directory: %s\n", sd)
  fmt.Fprintf(os.Stderr, "Build directory:  %s\n", bd)
  fmt.Fprintf(os.Stderr, "Tools directory:  %s\n", td)
  foundTools.ListTools()

  /* Scan the sources of each individual asm file and look for dependencies. */
  perFileDeps, totalDeps, e := ScanSources(foundTools["scan_includes"], sd, asmFiles)
  if e != nil { eexit("could not scan includes: " + e.Error()) }

  /* Scan the include files, as well. */
  _, includeDeps, e := ScanSources(foundTools["scan_includes"], sd, []string{"includes.asm"})
  if e != nil { eexit("could not scan deps of includes.asm: " + e.Error()) }

  /* Prepare the ninja file. */
  n := NinjaFile{Rules: rules}

  /* Resolve all discovered dependencies into build statements. */
  targets := resolveTargets(totalDeps, sd, bd, MakeRule)
  for _, b := range targets {
    if b.rule == "" { continue }
    n.Statements = append(n.Statements, b)
  }
  
  /* Resolve the direct include dependencies. */
  includeDepsResolved := make([]string, len(includeDeps))
  for i, j := range includeDeps {
    includeDepsResolved[i] = resolvePathToSrcOrBuild(j, sd, bd)
  }

  /* Create rules for every object file. These are compiled directly from their
     source asm files. */
  objFilesList := make([]string, len(asmFiles))
  for i, a := range asmFiles {
    objName := filepath.Join(bd, reAsmToObj.ReplaceAllString(a, "$1.o"))
    objFilesList[i] = objName
    var objDeps []string
    if fdeps, ok := perFileDeps[a]; ok {
      objDeps = make([]string, len(fdeps))
      for j, l := range fdeps { objDeps[j] = resolvePathToSrcOrBuild(l, sd, bd) }
    }
    n.Statements = append(n.Statements, &Stmt{
      rule:           "ASM",
      output:         objName,
      inputs:         []string{filepath.Join(sd, a)},
      implicitInputs: append(includeDepsResolved, objDeps...),
      flags: map[string]string{
        "rgbasm": fmt.Sprintf(
          "-Q8 -P %s -E -I %s -I %s",
          filepath.Join(sd, "includes.asm"), sd, bd),
      },
    })
  }

  /* Create rule for the final ROM. */
  n.Statements = append(n.Statements, &Stmt{
    rule:   "LINK_AND_FIX",
    output: "pokecrystal.gbc",
    inputs: objFilesList,
    flags: map[string]string{
      "layoutfile": filepath.Join(sd, "layout.link"),
      "symfile":    filepath.Join(bd, "pokecrystal.sym"),
      "mapfile":    filepath.Join(bd, "pokecrystal.map"),
      "rgblink":    "-Weverything -Wtruncation=1",
      "rgbfix":     "-Weverything -Cjv -t PM_CRYSTAL -k 01 -l 0x33 -m MBC3+TIMER+RAM+BATTERY -r 3 -p 0 -i BYTE -n 0",
    },
  })

  /* Do a final check to see if all rules are defined. */
  if missing := n.CheckRules(); len(missing) > 0 {
    for _, x := range missing { fmt.Fprintf(os.Stderr, "No rule '%s'!\n", x) }
    eexit("Some rules were not defined!")
  }

  /* For convenience, create the directories. */
  mentioned := make(map[string]struct{})
  for _, st := range n.Statements {
    mentioned[filepath.Dir(st.output)] = struct{}{}
    for _, fn := range st.inputs         { mentioned[filepath.Dir(fn)] = struct{}{} }
    for _, fn := range st.implicitInputs  { mentioned[filepath.Dir(fn)] = struct{}{} }
    for _, fn := range st.implicitOutputs { mentioned[filepath.Dir(fn)] = struct{}{} }
  }
  for d := range mentioned {
    if d == "." || fileExists(d) { continue }
    if e := os.MkdirAll(d, 0755); e != nil {
      ewarn(fmt.Sprintf("can't create directory: %s", d))
      continue
    }
    // fmt.Fprintf(os.Stderr, "created %s\n", d)
  }

  /* Write the final Ninja build file. */
  f, e := os.OpenFile(filepath.Join(bd, "build.ninja"), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0777)
  if e != nil { eexit(fmt.Sprintf("cannot open build.ninja for writing: %s", e.Error())) }
  defer f.Close()
  n.Print(f)
}

/*========== Utilities ==========*/

type NinjaFile struct {
  Rules      map[string]*Rule
  Statements []*Stmt
}

func (n NinjaFile) Print(f io.Writer) {
  for _, r := range n.Rules { r.Print(f) }
  for _, s := range n.Statements { s.Print(f) }
}

func (n NinjaFile) CheckRules() []string {
  missing := map[string]struct{}{}
  for _, s := range n.Statements {
    if _, ok := n.Rules[s.rule]; !ok { missing[s.rule] = struct{}{} }
  }
  list := make([]string, 0, len(missing))
  for k := range missing { list = append(list, k) }
  return list
}

/* Needed to deal with some Windows stuff.. */
func safe(i string) string {
  return strings.Replace(i, ":", "$:", -1)
}

/* Rule represents a Ninja `rule` block. */
type Rule struct {
  name    string
  command string
}

func (r Rule) Print(f io.Writer) {
  cmd := r.command
  if runtime.GOOS == "windows" {
    cmd = fmt.Sprintf("cmd /c \"%s\"", cmd)
  }
  fmt.Fprintf(f, "rule %s\n command = %s\n",
    r.name,
    cmd,
  )
}

/* Stmt represents a Ninja `build` statement. */
type Stmt struct {
  rule            string
  output          string
  implicitOutputs []string
  inputs          []string
  implicitInputs  []string
  flags           map[string]string
}

func (s Stmt) Print(f io.Writer) {
  fmt.Fprintf(f, "build %s ", safe(s.output))
  if len(s.implicitOutputs) > 0 {
    fmt.Fprint(f, "| ")
    for _, x := range s.implicitOutputs { fmt.Fprintf(f, "%s ", safe(x)) }
  }
  fmt.Fprintf(f, ": %s ", s.rule)
  for _, x := range s.inputs { fmt.Fprintf(f, "%s ", safe(x)) }
  if len(s.implicitInputs) > 0 {
    fmt.Fprint(f, "| ")
    for _, x := range s.implicitInputs { fmt.Fprintf(f, "%s ", safe(x)) }
  }
  fmt.Fprintln(f)
  for k, v := range s.flags { fmt.Fprintf(f, " %s = %s\n", k, safe(v)) }
}

type NeededTools map[string]string

func FindTools(dir string) (NeededTools, error) {
  t := make(NeededTools)
  allFound := true
  for _, name := range tools {
    binName := name
    if runtime.GOOS == "windows" {
      binName += ".exe"
    }
    p := filepath.Join(dir, binName)
    if !fileExists(p) {
      var err error
      p, err = exec.LookPath(name)
      if err != nil {
        ewarn(fmt.Sprintf("tool '%s' not found", name))
        allFound = false
        continue
      }
    }
    t[name] = p
  }
  if !allFound {
    return t, fmt.Errorf("not all tools were found")
  }
  return t, nil
}

func (v NeededTools) ListTools() {
  fmt.Fprintln(os.Stderr, "Found tools:")
  for _, name := range tools {
    fmt.Fprintf(os.Stderr, "% 30s -> %s\n", name, v[name])
  }
}

/* flagsFor returns a copy of the `fileFlags` entry for dep, or an empty map. */
func flagsFor(dep string) map[string]string {
  flags := map[string]string{}
  for k, v := range fileFlags[dep] { flags[k] = v }
  return flags
}

/*========== Dependency resolution ==========
  The heart of it all, I think. It iteratively resolves all dependency rules
  until it's pretty much stable. Because a dependency's inputs may themselves be
  built targets, it runs multiple passes, adding newly discovered inputs each time. */

const maxTargetPasses = 1000
type resolveResult struct {
  dep            string
  resolvedTarget *Stmt
  newInputs      []string
}

func resolveTargets(
  allDeps []string,
  sourceDir, buildDir string,
  matchFn func(string, map[string]*Stmt, string, string) (*Stmt, []string),
) map[string]*Stmt {
  /* Maps a target to the rule that makes it */
  targets        := make(map[string]*Stmt)

  /* Initial queue of dependencies */
  unresolvedSet  := make(map[string]bool)
  unresolvedList := append([]string{}, allDeps...)
  for _, dep := range allDeps { unresolvedSet[dep] = true }

  /* Resolve iteratively */
  for pp := 0; pp < maxTargetPasses; pp++ {
    fmt.Fprintf(os.Stderr, "Pass %d: %d resolved, %d remaining\n", pp, len(targets), len(unresolvedList))
    if len(unresolvedList) == 0 { break }

    /* We aren't writing to `targets` this pass, so make a copy for the goroutines
       below to operate on safely */
    targetsThis := make(map[string]*Stmt, len(targets))
    for a, b := range targets { targetsThis[a] = b }

    resultsChan := make(chan resolveResult, len(unresolvedList))
    for _, dep := range unresolvedList {
      go func(d string) {
        r, newInputs := matchFn(d, targetsThis, sourceDir, buildDir)
        resultsChan <- resolveResult{dep: d, resolvedTarget: r, newInputs: newInputs}
      }(dep)
    }

    var nextUnresolved []string
    resolvedThisPass := 0
    for i := 0; i < len(unresolvedList); i++ {
      r := <-resultsChan
      /* Successfully resolved this dep */
      if r.resolvedTarget != nil {
        targets[r.dep] = r.resolvedTarget
        delete(unresolvedSet, r.dep)
        resolvedThisPass++
        continue
      }
      /* Otherwise, check if there are sub-dependencies I need to add */
      for _, inp := range r.newInputs {
        if _, done := targets[inp]; !done && !unresolvedSet[inp] {
          unresolvedSet[inp] = true
          nextUnresolved = append(nextUnresolved, inp)
        }
      }
      /* Keep current dep in the unresolved list for the next pass */
      nextUnresolved = append(nextUnresolved, r.dep)
    }

    /* Update the list for the next pass */
    unresolvedList = nextUnresolved

    /* Fixed point, nothing more to resolve */
    if resolvedThisPass == 0 { break }
  }

  /* If there are any dependencies left unresolved, report that. */
  for _, dep := range unresolvedList {
    if _, ok := targets[dep]; !ok {
      fmt.Fprintf(os.Stderr, "ERROR: Could not resolve inputs for: %s\n", dep)
    }
  }

  /* Finally, remove ambiguities as to where the dependencies are located.
     Turning all the relative directories into absolute paths. */
  fmt.Fprintln(os.Stderr, "Removing target ambiguities")
  disambiguated := make(map[string]*Stmt, len(targets))
  for i, v := range targets {
    for a, b := range v.inputs         { v.inputs[a]         = resolvePathToSrcOrBuild(b, sourceDir, buildDir) }
    for a, b := range v.implicitInputs  { v.implicitInputs[a]  = resolvePathToSrcOrBuild(b, sourceDir, buildDir) }
    for a, b := range v.implicitOutputs { v.implicitOutputs[a] = resolvePathToSrcOrBuild(b, sourceDir, buildDir) }
    if filepath.IsAbs(i) { continue }
    srcpath := filepath.Join(sourceDir, i)
    if fileExists(srcpath) {
      v.output = srcpath
      disambiguated[srcpath] = v
    } else {
      if v.rule == "" { ewarn(fmt.Sprintf("Build %s, but it has no rule\n", i)) }
      bpath := filepath.Join(buildDir, i)
      v.output = bpath
      disambiguated[bpath] = v
    }
  }

  fmt.Fprintln(os.Stderr, "Dependency calculation OK")
  return disambiguated
}

func resolvePathToSrcOrBuild(i, sourceDir, buildDir string) string {
  if filepath.IsAbs(i) { return i }
  srcpath := filepath.Join(sourceDir, i)
  if fileExists(srcpath) { return srcpath }
  return filepath.Join(buildDir, i)
}

/*========== Scan sources using `scanTool` ==========
  `scanTool` is almost always gonna be `scan_includes` from the found tools.
  Returns:
    1. map[string][]string: A list of dependencies for each `asmFile`.
    2. []string Unique dependencies in total.
    3. An error.
*/

type CommandResult struct {
  origin string
  out    []string
  e      error
}

func ScanSources(scanTool, sourceDir string, asmFiles []string) (map[string][]string, []string, error) {
  var w sync.WaitGroup
  resCh := make(chan CommandResult, len(asmFiles))

  perAsmDeps := make(map[string][]string)
  allDeps    := make(map[string]struct{})

  for _, asmFile := range asmFiles {
    w.Add(1)
    go func(f string) {
      defer w.Done()
      out, e := scanFile(scanTool, f)
      resCh <- CommandResult{origin: f, out: out, e: e}
      fmt.Fprintf(os.Stderr, "%s has %d dependencies\n", f, len(out))
    }(asmFile)
  }
  go func() { w.Wait(); close(resCh) }()

  for r := range resCh {
    if r.e != nil {
      ewarn("scan_includes was unable to process " + r.origin)
      continue
    }
    perAsmDeps[r.origin] = r.out
    for _, f := range r.out { allDeps[f] = struct{}{} }
  }

  depList := make([]string, 0, len(allDeps))
  for d := range allDeps { depList = append(depList, d) }
  fmt.Fprintf(os.Stderr, "...%d dependencies total\n", len(depList))
  return perAsmDeps, depList, nil
}

func scanFile(scanTool, f string) ([]string, error) {
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  cmd := exec.CommandContext(ctx, scanTool, f)
  out, e := cmd.CombinedOutput()
  if e != nil { return []string{}, e }
  return strings.Fields(string(out)), nil
}

var fileExistence = make(map[string]struct{})

func fileExists(i string) bool {
  /* Here, `i` is assumed to be absolute to prevent ambiguities. A `warn` is
     emitted to warn me when it isn't. */
  if !filepath.IsAbs(i) { ewarn(fmt.Sprintf("%s is not absolute", i)) }
  _, ok := fileExistence[i]
  return ok
  /* The initial version, which checks on-demand. The Linux kernel cache may
     help, but wouldn't it be better if we handle the cache ourselves thus
     benefitting everyone? */
  /*
  _, e := os.Stat(i)
  if os.IsNotExist(e) { return false }
  return true
  */
}

/* refineRule nullifies a Rule
   if it has been detected to either have been already registered as a target, or
   actually exists within the source folder.

   A nil *Stmt means that this dependency has yet to be resolved, whereas a &Stmt{}
   means that there are explicitly no rules for this dependency. */
func refineRule(r Stmt, targets map[string]*Stmt, sourceRoot string) (*Stmt, []string) {
  allInputs := append(r.inputs, r.implicitInputs...)
  for _, p := range r.inputs {
    if !fileExists(filepath.Join(sourceRoot, p)) && !contains(targets, p) {
      return nil, allInputs
    }
  }
  return &r, allInputs
}

/* Again, Go 1.10. Which means `slices` haven't been invented yet. */
func contains(haystack map[string]*Stmt, needle string) bool {
  _, ok := haystack[needle]
  return ok
}

func eexit(msg string) {
  fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg)
  os.Exit(1)
}

func ewarn(msg string) {
  fmt.Fprintf(os.Stderr, "WARN: %s\n", msg)
}
