// Code generated by "stringer -type=CartridgeType,CGBFlag -output=cartridge_string.go"; DO NOT EDIT.

package io

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ROM-0]
	_ = x[MBC1-1]
	_ = x[MBC1RAM-2]
	_ = x[MBC1RAMBATT-3]
	_ = x[MBC2-5]
	_ = x[MBC2BATT-6]
	_ = x[MMM01-11]
	_ = x[MMM01RAM-12]
	_ = x[MMM01RAMBATT-13]
	_ = x[MBC3TIMERBATT-15]
	_ = x[MBC3TIMERRAMBATT-16]
	_ = x[MBC3-17]
	_ = x[MBC3RAM-18]
	_ = x[MBC3RAMBATT-19]
	_ = x[MBC5-25]
	_ = x[MBC5RAM-26]
	_ = x[MBC5RAMBATT-27]
	_ = x[MBC5RUMBLE-28]
	_ = x[MBC5RUMBLERAM-29]
	_ = x[MBC5RUMBLERAMBATT-30]
	_ = x[MBC6-32]
	_ = x[MBC7-34]
	_ = x[POCKETCAMERA-252]
	_ = x[BANDAITAMA5-253]
	_ = x[HUDSONHUC3-254]
	_ = x[HUDSONHUC1-255]
	_ = x[MBC1M-256]
	_ = x[M161-257]
	_ = x[WISDOMTREE-258]
}

const (
	_CartridgeType_name_0 = "ROMMBC1MBC1RAMMBC1RAMBATT"
	_CartridgeType_name_1 = "MBC2MBC2BATT"
	_CartridgeType_name_2 = "MMM01MMM01RAMMMM01RAMBATT"
	_CartridgeType_name_3 = "MBC3TIMERBATTMBC3TIMERRAMBATTMBC3MBC3RAMMBC3RAMBATT"
	_CartridgeType_name_4 = "MBC5MBC5RAMMBC5RAMBATTMBC5RUMBLEMBC5RUMBLERAMMBC5RUMBLERAMBATT"
	_CartridgeType_name_5 = "MBC6"
	_CartridgeType_name_6 = "MBC7"
	_CartridgeType_name_7 = "POCKETCAMERABANDAITAMA5HUDSONHUC3HUDSONHUC1MBC1MM161WISDOMTREE"
)

var (
	_CartridgeType_index_0 = [...]uint8{0, 3, 7, 14, 25}
	_CartridgeType_index_1 = [...]uint8{0, 4, 12}
	_CartridgeType_index_2 = [...]uint8{0, 5, 13, 25}
	_CartridgeType_index_3 = [...]uint8{0, 13, 29, 33, 40, 51}
	_CartridgeType_index_4 = [...]uint8{0, 4, 11, 22, 32, 45, 62}
	_CartridgeType_index_7 = [...]uint8{0, 12, 23, 33, 43, 48, 52, 62}
)

func (i CartridgeType) String() string {
	switch {
	case i <= 3:
		return _CartridgeType_name_0[_CartridgeType_index_0[i]:_CartridgeType_index_0[i+1]]
	case 5 <= i && i <= 6:
		i -= 5
		return _CartridgeType_name_1[_CartridgeType_index_1[i]:_CartridgeType_index_1[i+1]]
	case 11 <= i && i <= 13:
		i -= 11
		return _CartridgeType_name_2[_CartridgeType_index_2[i]:_CartridgeType_index_2[i+1]]
	case 15 <= i && i <= 19:
		i -= 15
		return _CartridgeType_name_3[_CartridgeType_index_3[i]:_CartridgeType_index_3[i+1]]
	case 25 <= i && i <= 30:
		i -= 25
		return _CartridgeType_name_4[_CartridgeType_index_4[i]:_CartridgeType_index_4[i+1]]
	case i == 32:
		return _CartridgeType_name_5
	case i == 34:
		return _CartridgeType_name_6
	case 252 <= i && i <= 258:
		i -= 252
		return _CartridgeType_name_7[_CartridgeType_index_7[i]:_CartridgeType_index_7[i+1]]
	default:
		return "CartridgeType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[CGBUnset-0]
	_ = x[CGBEnhanced-1]
	_ = x[CGBOnly-2]
}

const _CGBFlag_name = "CGBUnsetCGBEnhancedCGBOnly"

var _CGBFlag_index = [...]uint8{0, 8, 19, 26}

func (i CGBFlag) String() string {
	if i >= CGBFlag(len(_CGBFlag_index)-1) {
		return "CGBFlag(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _CGBFlag_name[_CGBFlag_index[i]:_CGBFlag_index[i+1]]
}
