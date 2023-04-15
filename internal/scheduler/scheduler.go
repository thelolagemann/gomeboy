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

	events      [256]*Event // only one event of each type can be scheduled at a time
	nextEventAt uint64      // the cycle at which the next event should be executed

	doubleSpeed bool // whether the scheduler is running at double speed (TODO: implement)

	debugLogging bool
}

func (s *Scheduler) EnableDebugLogging() {
	s.debugLogging = true
}

func (s *Scheduler) DisableDebugLogging() {
	s.debugLogging = false
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		divTimer: 0x5437, // TODO make configurable
		cycles:   0,
		events:   [256]*Event{},
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
}

func (s *Scheduler) Start() {
	// s.root = s.events[APUSample]
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
	s.nextEventAt = s.doEvents()
}

func (s *Scheduler) SysClock() uint16 {
	fmt.Printf("0x%04X\n", uint16((s.cycles-s.divTimer)&0xFFFF))
	fmt.Printf("0x%0X - 0x%0X = 0x%0X\n", s.cycles, s.divTimer, s.cycles-s.divTimer)
	return uint16((s.cycles - s.divTimer) & 0xFFFF)
}

func (s *Scheduler) SysClockReset() {
	s.divTimer = s.cycles
}

// doEvents executes all events scheduled in the list up to the given
// cycle. It returns the cycle at which the next event should be executed.
func (s *Scheduler) doEvents() uint64 {
	until := s.cycles // avoid the cost of accessing the field in each iteration
	nextEvent := s.nextEventAt

	for {
		// we need to copy the event to a local variable
		// as the handler may schedule a new event, which
		// could modify the event in the list
		event := s.root

		// set the next event to be executed
		s.root = event.next

		if event.eventType >= PPUHBlank && event.eventType <= PPUOAMInterrupt && s.debugLogging {
			// fmt.Printf("executing event %s at cycle %d\n", eventTypeNames[event.eventType], s.cycles)
		}
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

func (s *Scheduler) DoEventNow(event EventType) {
	s.events[event].handler()
}

// ScheduleEvent schedules an event to be executed at the given cycle.
func (s *Scheduler) ScheduleEvent(eventType EventType, cycle uint64) {
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
