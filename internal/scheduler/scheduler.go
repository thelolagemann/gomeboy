package scheduler

import "C"
import (
	"fmt"
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

	eventHandlers [256]func() // set to 256 (uint8 max) avoids bounds check on eventHandlers[eventType]()
	events        [256]*Event // only one event of each type can be scheduled at a time
	nextEventAt   uint64
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		cycles: 0,
		events: [256]*Event{},
	}

	// initialize the events with the number of event types
	// to avoid the cost of allocating a new event for each
	// scheduled event
	for i := 0; i < eventTypes; i++ {
		s.events[i] = &Event{}
	}

	return s
}

func (s *Scheduler) Cycle() uint64 {
	return s.cycles
}

// RegisterEvent registers a function of the EventType to be called when
// the event is scheduled for execution. This is to avoid the cost of
// having to allocate a function for each event, which would frequently
// invoke the garbage collector, despite the functions always performing
// the same task.
func (s *Scheduler) RegisterEvent(eventType EventType, fn func()) {
	s.eventHandlers[eventType] = fn
}

// Tick advances the scheduler by the given number of cycles. This will
// execute all scheduled events up to the current cycle. If an event is
// scheduled for the current cycle, it will be executed and removed from
// the list. If an event is scheduled for a cycle in the future, it will
// be executed when the scheduler is ticked with the cycle at which it
// should be executed.
func (s *Scheduler) Tick(c uint64) {
	// increment the cycle counter
	s.cycles += c

	// skip if there are no events scheduled
	if s.nextEventAt > s.cycles {
		return
	}

	// execute all scheduled events up to the current cycle
	for nextEvent := s.nextEventAt; nextEvent <= s.cycles; nextEvent = s.root.cycle {
		event := s.root

		s.root = event.next

		// execute the event
		s.eventHandlers[event.eventType]()
	}

	// update the next event to be executed
	s.nextEventAt = s.root.cycle
}

// ScheduleEvent schedules an event to be executed at the given cycle.
func (s *Scheduler) ScheduleEvent(eventType EventType, cycle uint64) {

	// when the event is scheduled, it is scheduled for the current cycle + the cycle
	// at which it should be executed
	atCycle := s.cycles + cycle

	var prev *Event
	this := s.events[eventType]
	this.cycle = atCycle
	this.eventType = eventType

	this.next = nil

	if s.root == nil {
		s.root = this
		return
	} else if atCycle < s.nextEventAt {
		// the event should be executed before the current event
		// so we can just prepend it
		this.next = s.root
		s.root = this
		s.nextEventAt = atCycle
		return
	}

	event := s.root
	for {
		if atCycle < event.cycle {
			// the event should be executed before the current event
			// so we need to insert it before the current event

			if prev == nil {
				// the event should be executed before the current event
				// and there is no previous event, so we can just prepend it
				this.next = event
				s.root = this
				s.nextEventAt = atCycle
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
	if s.root == nil {
		return
	}

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

func (s *Scheduler) DoEvent() uint64 {
	event := s.root

	s.root = event.next
	s.eventHandlers[event.eventType]()

	return s.root.cycle
}

// Skip invokes the scheduler to execute the next event, by setting the
// current cycle to the cycle at which the next event is scheduled to be
// executed. This is useful when the CPU is halted, and the scheduler
// should be invoked to execute until the CPU is un-halted by an interrupt.
func (s *Scheduler) Skip() {
	s.cycles = s.nextEventAt
	s.nextEventAt = s.DoEvent()
}

func (s *Scheduler) String() string {
	result := ""
	event := s.root
	for event != nil {
		result += fmt.Sprintf("%d:%d->", event.eventType, event.cycle)
		event = event.next
	}
	return result
}
