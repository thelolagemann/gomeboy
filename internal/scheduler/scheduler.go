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
	cycles   uint64
	divTimer uint64 // the cycle at which the DIV register was last reset
	root     *Event

	events      [256]*Event // only one event of each type can be scheduled at a time
	nextEventAt uint64      // the cycle at which the next event should be executed

	doubleSpeed bool // whether the scheduler is running at double speed (TODO: implement)
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		divTimer: 0x5433, // TODO make configurable
		cycles:   0,
		events:   [256]*Event{},
		root: &Event{
			cycle: math.MaxUint64,
			handler: func() {
				fmt.Println("scheduler: no event handler found")
			},
		},
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
	s.events[eventType].handler = fn
	s.events[eventType].eventType = eventType
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

	// if the next event is scheduled for a cycle in the future,
	// then we can return early and avoid iterating over the list
	// of events
	if s.nextEventAt > s.cycles {
		return
	}
	//fmt.Println(s.String())

	// update the next event to be executed
	s.nextEventAt = s.doEvents(s.nextEventAt)
}

func (s *Scheduler) SysClock() uint16 {
	return uint16((s.cycles - s.divTimer) & 0xFFFF)
}

func (s *Scheduler) SysClockReset() {
	s.divTimer = s.cycles
}

// doEvents executes all events scheduled in the list up to the given
// cycle. It returns the cycle at which the next event should be executed.
func (s *Scheduler) doEvents(nextEvent uint64) uint64 {
	for nextEvent <= s.cycles {
		// we need to copy the event to a local variable
		// as the handler may schedule a new event, which
		// could modify the event in the list
		event := s.root

		// set the next event to be executed
		s.root = event.next

		// execute the event
		event.handler()

		// set the cycle to the next event to be executed
		nextEvent = s.root.cycle
	}

	return nextEvent
}

// ScheduleEvent schedules an event to be executed at the given cycle.
func (s *Scheduler) ScheduleEvent(eventType EventType, cycle uint64) {
	// when the event is scheduled, it is scheduled for the current cycle + the cycle
	// at which it should be executed
	atCycle := s.cycles + cycle

	var prev *Event
	this := s.events[eventType]
	this.cycle = atCycle

	if atCycle < s.nextEventAt {
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

		// the event should be executed after the current event
		if event.next == nil && event.cycle <= atCycle {
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

// TODO pass in event type to avoid BCE in the loop
// TODO
func (s *Scheduler) DoEvent() uint64 {
	event := s.root

	s.root = event.next
	event.handler()

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

func (s *Scheduler) Until(increment EventType) uint64 {
	event := s.root
	for event != nil {
		if event.eventType == increment {
			return event.cycle - s.cycles
		}
		event = event.next
	}
	return 0
}
