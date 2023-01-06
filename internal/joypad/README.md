# Joypad  

The joypad package provides an implementation of the Game Boy joypad. It is 
responsible for selecting the correct button input based on the Joypad Register
and the current state of the buttons.

## How it works

The joypad is arranged in a 2x4 matrix. Writing to bits 4 and 5 of the joypad 
register selects the row to read from, and reading from bits 0 to 3 returns the
state of the buttons in that row. Bits 6 and 7 are unused, and always return 1.

For example, writing `0b1110_1111` to the joypad register and reading from the 
joypad register will return the state of the arrow buttons (up, down, left, right).
So if the up button is pressed, the joypad register will return `0b1110_1011`. 

Similarly, writing `0b1101_1111` and reading from the joypad register will return
the state of the A, B, select and start buttons. If the A button is pressed, the
joypad register will return `0b1101_1110`.

The joypad register is mapped to memory address `0xFF00`, and is laid out as
follows:

| Bit | Button     | State                                 |
|-----|------------|---------------------------------------|
| 0   | Right/A    | 0 = Pressed (Read Only)               |
| 1   | Left/B     | 0 = Pressed (Read Only)               |
| 2   | Up/Select  | 0 = Pressed (Read Only)               |
| 3   | Down/Start | 0 = Pressed (Read Only)               |
| 4   | Row Select | 0 = Selection Directions (Write Only) |
| 5   | Row Select | 0 = Select Actions (Write Only)       |
| 6   | Not Used   | 1 = Not Used (Read Only)              |
| 7   | Not Used   | 1 = Not Used (Read Only)              |

## How it's implemented

The joypad is implemented as a struct that implements the `mmu.IOBus` interface. 
This allows the MMU to dispatch read and write calls to the joypad, without
knowing anything about the implementation details of the joypad. 

The MMU should only ever dispatch read and write calls to the joypad register, 
so the joypad can safely assume that it will only ever receive calls to `Read` and `Write`
with the address `0xFF00`. Otherwise, a panic will occur to indicate that the joypad 
is being used incorrectly.

When the joypad receives a read call, it will return the register ORed with the
current state of the buttons in the selected row. When the joypad receives a write
call, it will update the register with the new value. Only bits 4 and 5 are used
to select the row, so the other bits are ignored.