// Code generated by "stringer -type=EventType -output=event_string.go"; DO NOT EDIT.

package scheduler

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[APUFrameSequencer-0]
	_ = x[APUFrameSequencer2-1]
	_ = x[APUChannel1-2]
	_ = x[APUChannel2-3]
	_ = x[APUChannel3-4]
	_ = x[APUSample-5]
	_ = x[EIPending-6]
	_ = x[EIHaltDelay-7]
	_ = x[PPUStartHBlank-8]
	_ = x[PPUEndHBlank-9]
	_ = x[PPUHBlankInterrupt-10]
	_ = x[PPUBeginFIFO-11]
	_ = x[PPUFIFOTransfer-12]
	_ = x[PPUStartOAMSearch-13]
	_ = x[PPUEndFrame-14]
	_ = x[PPUContinueOAMSearch-15]
	_ = x[PPUPrepareEndOAMSearch-16]
	_ = x[PPUEndOAMSearch-17]
	_ = x[PPULine153Continue-18]
	_ = x[PPULine153End-19]
	_ = x[PPUStartVBlank-20]
	_ = x[PPUContinueVBlank-21]
	_ = x[PPUEndVRAMTransfer-22]
	_ = x[PPUStartGlitchedLine0-23]
	_ = x[PPUMiddleGlitchedLine0-24]
	_ = x[PPUContinueGlitchedLine0-25]
	_ = x[PPUEndGlitchedLine0-26]
	_ = x[PPUOAMInterrupt-27]
	_ = x[DMAStartTransfer-28]
	_ = x[DMAEndTransfer-29]
	_ = x[DMATransfer-30]
	_ = x[TimerTIMAReload-31]
	_ = x[TimerTIMAFinishReload-32]
	_ = x[TimerTIMAIncrement-33]
	_ = x[SerialBitTransfer-34]
	_ = x[SerialBitInterrupt-35]
	_ = x[CameraShoot-36]
}

const _EventType_name = "APUFrameSequencerAPUFrameSequencer2APUChannel1APUChannel2APUChannel3APUSampleEIPendingEIHaltDelayPPUStartHBlankPPUHBlankPPUHBlankInterruptPPUBeginFIFOPPUFIFOTransferPPUStartOAMSearchPPUEndFramePPUContinueOAMSearchPPUPrepareEndOAMSearchPPUEndOAMSearchPPULine153ContinuePPULine153EndPPUStartVBlankPPUContinueVBlankPPUVRAMTransferPPUStartGlitchedLine0PPUMiddleGlitchedLine0PPUContinueGlitchedLine0PPUEndGlitchedLine0PPUOAMInterruptDMAStartTransferDMAEndTransferDMATransferTimerTIMAReloadTimerTIMAFinishReloadTimerTIMAIncrementSerialBitTransferSerialBitInterruptCameraShoot"

var _EventType_index = [...]uint16{0, 17, 35, 46, 57, 68, 77, 86, 97, 111, 120, 138, 150, 165, 182, 193, 213, 235, 250, 268, 281, 295, 312, 327, 348, 370, 394, 413, 428, 444, 458, 469, 484, 505, 523, 540, 558, 569}

func (i EventType) String() string {
	if i >= EventType(len(_EventType_index)-1) {
		return "EventType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _EventType_name[_EventType_index[i]:_EventType_index[i+1]]
}
