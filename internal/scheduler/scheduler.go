package scheduler

import "C"
import (
	"fmt"
	"math"
)

// Scheduler is a simple event scheduler that can be used to schedule events
// to be executed at a specific cycle.
//
// The scheduler is a linked list of events, sorted by the cycle at which
// they should be executed. When an event is scheduled, it is inserted into
// the list in the correct position, and when the scheduler is ticked, the
// next event is executed and removed from the list, if the event is scheduled
// for the current cycle.
type Scheduler struct {
	cycles uint64
	root   *Event

	eventHandlers [16]func() // 7 is the number of event types
	events        [16]*Event // only one event of each type can be scheduled at a time
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		cycles: 0,
		events: [16]*Event{},
	}

	for i := 0; i < 16; i++ {
		s.events[i] = &Event{}
	}

	return s
}

func (s *Scheduler) Cycle() uint64 {
	return s.cycles
}

func (s *Scheduler) Tick(c uint64) {
	s.cycles += c
}

func (s *Scheduler) Next() uint64 {
	if s.root == nil {
		return math.MaxUint64
	}

	return s.root.cycle
}

// RegisterEvent registers a function of the EventType to be called when
// the event is scheduled for execution. This is to avoid the cost of
// having to allocate a function for each event, which would frequently
// invoke the garbage collector, despite the functions always performing
// the same task.
func (s *Scheduler) RegisterEvent(eventType EventType, fn func()) {
	s.eventHandlers[eventType] = fn
}

func (s *Scheduler) ScheduleEvent(eventType EventType, cycle uint64) {

	// when the event is scheduled, it is scheduled for the current cycle + the cycle
	// at which it should be executed
	atCycle := s.cycles + cycle

	var prev *Event
	this := s.events[eventType]
	this.cycle = atCycle
	this.eventType = eventType
	this.fn = s.eventHandlers[eventType]
	this.next = nil

	event := s.root
	for {
		if event == nil {
			// no scheduled events, so we can just add the event to the root
			s.root = this
			break
		}

		if atCycle < event.cycle {
			// the event should be executed before the current event
			// so we need to insert it before the current event

			if prev == nil {
				// the event should be executed before the current event
				// and there is no previous event, so we can just prepend it
				this.next = event
				s.root = this
				break
			} else if prev.cycle <= atCycle {
				// the event should be executed between the previous event
				// and the current event, so we can just insert it
				this.next = event

				prev.next = this

				break
			}
		}

		if event.next == nil && event.cycle <= atCycle {
			// the event should be executed after the current event
			event.next = this
			break
		}

		prev = event
		event = event.next
	}

}

func (s *Scheduler) DescheduleEvent(eventType EventType) {
	var prev *Event
	event := s.root

	for event != nil {
		if event.eventType == eventType {
			if prev == nil {
				s.root = event.next
				break
			} else {
				prev.next = event.next
				break
			}
		}
		prev = event
		event = event.next
	}
}

func (s *Scheduler) DoEvent() {
	event := s.root
	if event == nil {
		return
	}

	s.root = event.next
	event.fn()

}

func (s *Scheduler) String() string {
	result := ""
	event := s.root
	for event != nil {
		result += fmt.Sprintf("%s:%d->", event.eventType, event.cycle)
		event = event.next
	}
	return result
}
