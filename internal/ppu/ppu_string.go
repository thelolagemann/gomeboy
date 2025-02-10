// Code generated by "stringer -type=GlitchedLineState,LineState,OffscreenLineState,FetcherState -output=ppu_string.go"; DO NOT EDIT.

package ppu

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[StartGlitchedLine-0]
	_ = x[GlitchedLineOAMWBlock-1]
	_ = x[GlitchedLineEndOAM-2]
	_ = x[GlitchedLineStartPixelTransfer-3]
}

const _GlitchedLineState_name = "StartGlitchedLineGlitchedLineOAMWBlockGlitchedLineEndOAMGlitchedLineStartPixelTransfer"

var _GlitchedLineState_index = [...]uint8{0, 17, 38, 56, 86}

func (i GlitchedLineState) String() string {
	if i < 0 || i >= GlitchedLineState(len(_GlitchedLineState_index)-1) {
		return "GlitchedLineState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _GlitchedLineState_name[_GlitchedLineState_index[i]:_GlitchedLineState_index[i+1]]
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[StartOAMScan-0]
	_ = x[ReleaseOAMBus-1]
	_ = x[StartPixelTransfer-2]
	_ = x[PixelTransferDummy-3]
	_ = x[PixelTransferSCXDiscard-4]
	_ = x[PixelTransferLX0-5]
	_ = x[PixelTransferLX8-6]
	_ = x[EnterHBlank-7]
	_ = x[HBlankUpdateLY-8]
	_ = x[HBlankUpdateOAM-9]
	_ = x[HBlankUpdateVisibleLY-10]
	_ = x[HBlankEnd-11]
}

const _LineState_name = "StartOAMScanReleaseOAMBusStartPixelTransferPixelTransferDummyPixelTransferSCXDiscardPixelTransferLX0PixelTransferLX8EnterHBlankHBlankUpdateLYHBlankUpdateOAMHBlankUpdateVisibleLYHBlankEnd"

var _LineState_index = [...]uint8{0, 12, 25, 43, 61, 84, 100, 116, 127, 141, 156, 177, 186}

func (i LineState) String() string {
	if i < 0 || i >= LineState(len(_LineState_index)-1) {
		return "LineState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _LineState_name[_LineState_index[i]:_LineState_index[i+1]]
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[StartVBlank-0]
	_ = x[VBlankUpdateLY-1]
	_ = x[VBlankUpdateLYC-2]
	_ = x[VBlankHandleInt-3]
	_ = x[StartVBlankLastLine-4]
	_ = x[Line153LYUpdate-5]
	_ = x[Line153LY0-6]
	_ = x[Line153LYC-7]
	_ = x[Line153LYC0-8]
	_ = x[EndFrame-9]
}

const _OffscreenLineState_name = "StartVBlankVBlankUpdateLYVBlankUpdateLYCVBlankHandleIntStartVBlankLastLineLine153LYUpdateLine153LY0Line153LYCLine153LYC0EndFrame"

var _OffscreenLineState_index = [...]uint8{0, 11, 25, 40, 55, 74, 89, 99, 109, 120, 128}

func (i OffscreenLineState) String() string {
	if i < 0 || i >= OffscreenLineState(len(_OffscreenLineState_index)-1) {
		return "OffscreenLineState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _OffscreenLineState_name[_OffscreenLineState_index[i]:_OffscreenLineState_index[i+1]]
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[BGWinActivating-0]
	_ = x[BGGetTileNoT1-1]
	_ = x[BGGetTileNoT2-2]
	_ = x[BGGetTileDataLowT1-3]
	_ = x[BGGetTileDataLowT2-4]
	_ = x[BGGetTileDataHighT1-5]
	_ = x[BGWinGetTileDataHighT2-6]
	_ = x[BGWinPushPixels-7]
}

const _FetcherState_name = "BGWinActivatingBGGetTileNoT1BGGetTileNoT2BGGetTileDataLowT1BGGetTileDataLowT2BGGetTileDataHighT1BGWinGetTileDataHighT2BGWinPushPixels"

var _FetcherState_index = [...]uint8{0, 15, 28, 41, 59, 77, 96, 118, 133}

func (i FetcherState) String() string {
	if i < 0 || i >= FetcherState(len(_FetcherState_index)-1) {
		return "FetcherState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _FetcherState_name[_FetcherState_index[i]:_FetcherState_index[i+1]]
}
