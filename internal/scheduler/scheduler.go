package scheduler

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

	sampler     func()
	events      [eventTypes]*Event // only one event of each type can be scheduled at a time
	nextEventAt uint64             // the cycle at which the next event should be executed

	doubleSpeed bool // whether the scheduler is running at double speed (TODO: implement)

}

func (s *Scheduler) OverrideDiv(div uint16) {
	s.divTimer = s.cycles - uint64(div)
}

func NewScheduler() *Scheduler {
	s := &Scheduler{ // 0x = DMG magic value,
		cycles: 0,
		events: [eventTypes]*Event{},
		root: &Event{
			next:  nil,
			cycle: math.MaxUint64,
			handler: func() {

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

	if eventType == APUSample {
		s.sampler = fn
	}
}

// Tick advances the scheduler by the given number of cycles. This will
// execute all scheduled events up to the current cycle. If an event is
// scheduled for the current cycle, it will be executed and removed from
// the list. If an event is scheduled for a cycle in the future, it will
// be executed when the scheduler is ticked with the cycle at which it
// should be executed.
func (s *Scheduler) Tick(c uint64) {
	// what cycle are we advancing to
	cycleToAdvance := s.cycles + c

	// if the next event occurs in the future, simply increment
	// the cycle count and return
	if s.nextEventAt > cycleToAdvance {
		s.cycles = cycleToAdvance
		return
	}

	// get the cycle at which the next event occurs
	nextEvent := s.nextEventAt

	// otherwise perform accurate loop, jumping from each event
	// (mostly to pass blarggs wave read/write/trigger - maybe make
	// this optional in the future?)
	for nextEvent <= cycleToAdvance {
		s.cycles = nextEvent

		nextEvent = s.doEvents(s.cycles)
	}

	// update cycles and next event
	s.cycles = cycleToAdvance
	s.nextEventAt = nextEvent
}

// SysClock returns the internal divider clock of the Game Boy.
func (s *Scheduler) SysClock() uint16 {
	return uint16((s.cycles - s.divTimer) & 0xFFFF)
}

// SysClockReset resets the internal divider clock. TODO notify
// timer of reset.
func (s *Scheduler) SysClockReset() {
	s.divTimer = s.cycles
}

// doEvents executes all events scheduled in the list up to the given
// cycle. It returns the cycle at which the next event should be executed.
func (s *Scheduler) doEvents(until uint64) uint64 {
	nextEvent := s.nextEventAt

	for {
		// we need to copy the event to a local variable
		// as the handler may schedule a new event, which
		// could modify the event in the list
		event := s.root

		// set the next event to be executed
		s.root = event.next

		// execute the event
		event.handler()

		nextEvent = s.root.cycle

		// check if there are events to execute
		if nextEvent > until {
			break
		}
	}

	return nextEvent
}

// ScheduleEvent schedules an event to be executed at the given cycle.
func (s *Scheduler) ScheduleEvent(eventType EventType, cycle uint64) {
	if s.doubleSpeed && eventType <= PPUOAMInterrupt {
		cycle = cycle * 2
	}
	// when the event is scheduled, it is scheduled for the current cycle + the cycle
	// at which it should be executed
	atCycle := s.cycles + cycle

	// get the event to insert from the event pool and set the cycle
	// this is to avoid the cost of allocating a new event for each
	// scheduled event
	eventToInsert := s.events[eventType]
	eventToInsert.cycle = atCycle

	// if the event is scheduled for a cycle before the next scheduled
	// event, then we can just prepend it to the list by setting it as
	// the root and updating the next event to be the current root
	if atCycle <= s.nextEventAt {
		eventToInsert.next = s.root
		s.root = eventToInsert
		s.nextEventAt = atCycle

		// early return to avoid iterating over the list
		return
	}

	// iterate over the list of events to find the correct position
	// to insert the event
	var currentRoot = s.root // start at the root
	var nextCycle = currentRoot.cycle
	var didLoop = currentRoot.cycle <= atCycle
	var prev *Event
eventLoop:
	// typically you would use a for loop here, but in this case,
	// the performance critical nature of the scheduler means
	// that we need to avoid as much overhead as possible
	if nextCycle <= atCycle {
		// when the event is scheduled for a cycle after the current
		// root, then we need to insert the event after the current
		// root, so we need to continue iterating over the list
		// until we find the correct position
		prev = currentRoot
		currentRoot = currentRoot.next
		nextCycle = currentRoot.cycle
		goto eventLoop
	}

	// TODO use positions instead of nil checks

	if !didLoop {
		eventToInsert.next = currentRoot
		s.root = eventToInsert
		s.nextEventAt = atCycle
	} else {
		prev.next = eventToInsert
		prev.next.next = currentRoot
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

// ChangeSpeed informs the scheduler that the speed of the CPU has either gone
// from normal speed to double speed, or from double speed to normal speed.
// This is useful for the scheduler to know when to schedule events for the
// CPU, as events are scheduled for the CPU at a different rate when the
// CPU is running at double speed.
func (s *Scheduler) ChangeSpeed(speed bool) {
	if !s.doubleSpeed && speed {
		eventsProcessed := [eventTypes]bool{} // filthy hack to avoid processing the same event twice
		// we are going from normal speed to double speed
		// so we need to halve the event cycles for events
		// affected by the speed change

		// we need to iterate over the linked list of events
		// and halve the cycle for each event if the event
		// is affected by the speed change (APU, PPU, Serial)
		event := s.root
		for event != nil {
			if event.eventType <= PPUOAMInterrupt {
				if eventsProcessed[event.eventType] {
					event = event.next
					continue
				}
				eventsProcessed[event.eventType] = true
				// first we need to get the cycle at which the event
				// would be executed at normal speed
				cycleToExecute := event.cycle

				// then we need to calculate in how many cycles that
				// is from the current cycle
				cyclesFromNow := cycleToExecute - s.cycles

				// then we need to halve the cycles from now
				cyclesFromNow = cyclesFromNow / 2

				// then we need to add the cycles from now to the
				// current cycle to get the cycle at which the event
				// will be executed at double speed

				// we need to reschedule the event at the new cycle (not just change the cycle, as that would mess up the linked list)
				s.DescheduleEvent(event.eventType)
				s.ScheduleEvent(event.eventType, cyclesFromNow)
			}
			event = event.next
		}
	} else if s.doubleSpeed && !speed {
		// we are going from double speed to normal speed
		// so we need to double the event cycles for events
		// affected by the speed change

		// we need to iterate over the linked list of events
		// and double the cycle for each event if the event
		// is affected by the speed change (APU, PPU, Serial)
		event := s.root
		for event != nil {
			if event.eventType <= PPUOAMInterrupt {
				// first we need to get the cycle at which the event
				// would be executed at double speed
				cycleToExecute := event.cycle

				// then we need to calculate in how many cycles that
				// is from the current cycle
				cyclesFromNow := cycleToExecute - s.cycles

				// then we need to double the cycles from now
				cyclesFromNow = cyclesFromNow * 2

				// then we need to add the cycles from now to the
				// current cycle to get the cycle at which the event
				// will be executed at normal speed

				// we need to change the cycle of the event (not just reschedule it, as that would mess up the linked list)
				s.DescheduleEvent(event.eventType)
				s.ScheduleEvent(event.eventType, cyclesFromNow)
			}
			event = event.next
		}
	}

	s.doubleSpeed = speed
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
		//fmt.Println(event)
		result += fmt.Sprintf("%s:%d->", event.eventType, event.cycle)
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

func (s *Scheduler) DoubleSpeed() bool {
	return s.doubleSpeed
}
