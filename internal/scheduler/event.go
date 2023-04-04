package scheduler

type EventType int

const (
	APUFrameSequencer EventType = iota
	APUChannel1
	APUChannel2
	APUChannel3
	APUChannel3WaveRAM
	APUChannel4
	APUSample
)

type Event struct {
	cycle     uint64
	eventType EventType
	fn        func()
	next      *Event
}

func (e *Event) Reset() {
	e.cycle = 0
	e.eventType = 0
	e.fn = nil
}
