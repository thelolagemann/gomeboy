# Interrupts

The interrupts package provides a service for managing interrupts in a Gameboy emulator. It includes constants for the 
addresses of different interrupts (VBlank, LCD, Timer, Serial, Joypad), as well as corresponding bits in the Interrupt 
Flag and Interrupt Enable registers.

### Hardware Registers

The interrupts service has two hardware registers mapped to memory addresses:

| Address | Name | Description                |
|---------|------|----------------------------|
| 0xFF0F  | IF   | Interrupt Flag Register.   |
| 0xFFFF  | IE   | Interrupt Enable Register. |

And they are laid out as follows:

| Bit | Interrupt | State                    |
|-----|-----------|--------------------------|
| 0   | VBlank    | 1 = Request/Enable (R/W) |
| 1   | LCD       | 1 = Request/Enable (R/W) |
| 2   | Timer     | 1 = Request/Enable (R/W) |
| 3   | Serial    | 1 = Request/Enable (R/W) |
| 4   | Joypad    | 1 = Request/Enable (R/W) |
| 5   | Not Used  | 1                        |
| 6   | Not Used  | 1                        |
| 7   | Not Used  | 1                        |

## How it works

Interrupts are requested by setting the corresponding flag in the Interrupt Flag Register. The CPU checks the Interrupt
Master Enable `(IME)` flag to see if interrupts are enabled, if they are, it will then check Interrupt Enable Register 
`(IE)` and the Interrupt Flag Register `(IF)` to determine which interrupt to service. If an interrupt is enabled and 
its corresponding flag is set, the CPU will jump to the corresponding interrupt vector and clear the flag. Interrupts
are checked in the following order:

1. VBlank
2. LCD
3. Timer
4. Serial
5. Joypad

## How it's implemented

Interrupts are requested by calling the `Request` method on the service, which will set the corresponding flag in the
Interrupt Flag Register. When appropriate, the CPU will call the `Vector()` method to determine if an interrupt should be
serviced. If an interrupt is to be serviced, the `Vector()` method will clear the corresponding flag in the Interrupt Flag
Register and return the vector of the interrupt. 