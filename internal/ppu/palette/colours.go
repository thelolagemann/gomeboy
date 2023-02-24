package palette

const (
	White            = 0xFFFFFF
	Black            = 0x000000
	Grullo           = 0xADAD84
	Ming             = 0x42737B
	PhilippineOrange = 0xFF7300
	BrownTraditional = 0x944200
	BlueJeans        = 0x5ABDFF
	Yellow           = 0xFFFF00
	MaizeCrayola     = 0xFFC542
	Vodka            = 0xB5B5FF
	Conditioner      = 0xFFFFCE
)

// TODO palette dump parse
// - iterate over all roms in roms folder
// - load rom with boot ROM enabled
// - upon loading, save palette to file
// - determine palette hash
// - if hash already exists, skip
// - if hash does not exist, save palette to file
// - after all roms have been processed, find all unique colours
// - create hex code for each colour, and determine name of colour
