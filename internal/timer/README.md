# Timer

The timer package provides an implementation of the Game Boy's timer. It is responsible for maintaining the state of the
timer in sync with the CPU's clock speed, and for triggering the timer interrupt when the timer overflows.

### Hardware Registers

The timer package provides the following hardware registers:

| Address | Name | Description      |
|---------|------|------------------|
| 0xFF04  | DIV  | Divider Register |
| 0xFF05  | TIMA | Timer Counter    |
| 0xFF06  | TMA  | Timer Modulo     |
| 0xFF07  | TAC  | Timer Control    |

#### Divider Register (0xFF04)

The divider register is a 16-bit register that is incremented at a fixed rate of 16384 Hz. Writing any value to this 
register resets it to 0. Although the register is 16-bits wide, only the lower 8 bits are accessible.

#### Timer Counter (0xFF05)

The timer counter is an 8-bit register that is incremented at a rate specified by the timer control register. When the
timer counter overflows (i.e. increments from 0xFF to 0x00), it is reset to the value stored in the timer modulo
register, and a timer interrupt is requested.

#### Timer Modulo (0xFF06)

The timer modulo is an 8-bit register that stores the value that the timer counter is reset to when it overflows.

#### Timer Control (0xFF07)

The timer control is an 8-bit register that controls the timer counter. Bit 2 of this register enables and disables the
timer, while bits 1 and 0 control the timer's clock frequency. The following table shows the possible values of the
timer control register, and the corresponding timer frequencies:

| Value | Frequency |
|-------|-----------|
| 0x00  | 4096 Hz   |
| 0x01  | 262144 Hz |
| 0x02  | 65536 Hz  |
| 0x03  | 16384 Hz  |

## How it works

The timer consists of two main components: the divider register `DIV`, and the timer counter `TIMA`. The divider 
register is incremented at a fixed rate of 16384 Hz, and the timer counter is incremented at a rate specified by the 
timer control register `TAC`. When the timer counter overflows, it is reset to the value stored in the timer modulo
register `TMA`, and a timer interrupt is requested. 

During each CPU cycle, the timer checks if it's enabled, and if so, increments the timer counter. If the timer counter
overflows, it is reset to the value stored in the timer modulo register, and a timer interrupt is requested. The divider
register always increments during each CPU cycle, regardless of whether the timer is enabled or not.


## How it's implemented
