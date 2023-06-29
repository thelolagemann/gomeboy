# gomeboy

![Go](https://img.shields.io/github/go-mod/go-version/thelolagemann/go-gameboy/main)

GomeBoy is my attempt at creating a fairly accurate and reasonably performant Game Boy emulator written with golang. It 
is still currently in the very early stages of development, but it is already capable of running quite a few games with
varying degrees of success.

---

## Screenshots

### DMG Games

![Tetris](assets/images/tetris.png)
![Super Mario Land](assets/images/super-mario-land2.png)
![Pokemon Red](assets/images/pokemon-red.png)

### DMG Games running on CGB hardware

![Tetris](assets/images/tetris-cgb.png)
![Super Mario Land](assets/images/super-mario-land2-cgb.png)
![Pokemon Red](assets/images/pokemon-red-cgb.png)

### CGB Games

![Tetris DX](assets/images/tetris-dx.png)
![Mario Tennis](assets/images/mario-tennis.png)
![Pokemon Crystal](assets/images/pokemon-crystal.png)

### Peripherals (Printer, Link Cable)

![Printer](assets/images/printer.gif)
![Link Cable](assets/images/link-cable.png)

---

## Features


- GameBoy (DMG) and GameBoy Color (CGB) support
- SRAM and RTC support
- Run DMG games with CGB colorization palettes (without using a boot ROM)
- Automated testing against a large number of test ROMs
- Peripherals
	- Cartridge Mappers
      - MBC1	
      - MBC2
      - MBC3
      - MBC5
      - ROM
  - Cheat Carts
    - Game Genie
    - GameShark
  - Serial
    - Printer
    - Link Cable
    - Local Multiplayer
- Platform-agnostic (runs on Windows, Linux, and Mac)

---

# Automated Test Results


![progress](https://progress-bar.dev/89/?scale=100&title=passing%20162,%20failing%2019&width=500)

# Test Results
| Test Suite | Pass Rate | Tests Passed | Tests Failed | Tests Total |
| --- | --- | --- | --- | --- |
| acid2 | 75% | 3 | 1 | 4 |
| bully | 0% | 0 | 1 | 1 |
| blarrg | 100% | 43 | 0 | 43 |
| little-things-gb | 100% | 4 | 0 | 4 |
| mooneye | 92% | 106 | 8 | 114 |
| samesuite | 46% | 6 | 7 | 13 |
| strikethrough | 0% | 0 | 2 | 2 |

<sup>Visit the [tests](tests/README.md) directory for more information.</sup>

---

# TODO

- [ ] build instructions
- [ ] github actions
- [ ] improve error handling and logging
- [ ] expose more emulator options to the user