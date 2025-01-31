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
	_ = x[PPUHandleVisualLine-8]
	_ = x[PPUHandleGlitchedLine0-9]
	_ = x[PPUEndFrame-10]
	_ = x[PPULine153Continue-11]
	_ = x[PPULine153End-12]
	_ = x[PPUStartVBlank-13]
	_ = x[PPUContinueVBlank-14]
	_ = x[PPUOAMInterrupt-15]
	_ = x[DMAStartTransfer-16]
	_ = x[DMAEndTransfer-17]
	_ = x[DMATransfer-18]
	_ = x[TimerTIMAReload-19]
	_ = x[TimerTIMAFinishReload-20]
	_ = x[TimerTIMAIncrement-21]
	_ = x[SerialBitTransfer-22]
	_ = x[SerialBitInterrupt-23]
	_ = x[CameraShoot-24]
}

const _EventType_name = "APUFrameSequencerAPUFrameSequencer2APUChannel1APUChannel2APUChannel3APUSampleEIPendingEIHaltDelayPPUHandleVisualLinePPUHandleGlitchedLine0PPUEndFramePPULine153ContinuePPULine153EndPPUStartVBlankPPUContinueVBlankPPUOAMInterruptDMAStartTransferDMAEndTransferDMATransferTimerTIMAReloadTimerTIMAFinishReloadTimerTIMAIncrementSerialBitTransferSerialBitInterruptCameraShoot"

var _EventType_index = [...]uint16{0, 17, 35, 46, 57, 68, 77, 86, 97, 116, 138, 149, 167, 180, 194, 211, 226, 242, 256, 267, 282, 303, 321, 338, 356, 367}

func (i EventType) String() string {
	if i >= EventType(len(_EventType_index)-1) {
		return "EventType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _EventType_name[_EventType_index[i]:_EventType_index[i+1]]
}
