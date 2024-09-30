# Automated test results
![progress](https://progress-bar.xyz/90/?scale=100&title=passing%20227,%20failing%2025&width=500)

#### This document was automatically generated from commit 334025ab
<hr/>
GomeBoy is automatically tested against the following test suites:

* **[Blargg's test roms](https://github.com/retrio/gb-test-roms)**  
  <sup>by [Shay Green (a.k.a. Blargg)](http://www.slack.net/~ant/) </sup>
* **[Bully](https://github.com/Hacktix/BullyGB)**, 
  **[scribbltests](https://github.com/Hacktix/scribbltests)** 
  and **[Strikethrough](https://github.com/Hacktix/strikethrough.gb)**  
  <sup>by [Hacktix](https://github.com/Hacktix) </sup>
* **[cgb-acid-hell](https://github.com/mattcurrie/cgb-acid-hell)**,
  **[cgb-acid2](https://github.com/mattcurrie/cgb-acid2)** and
  **[dmg-acid2](https://github.com/mattcurrie/dmg-acid2)**  
  <sup>by [Matt Currie](https://github.com/mattcurrie) </sup>
* **[(parts of) little-things-gb](https://github.com/pinobatch/little-things-gb)**  
  <sup>by [Damian Yerrick](https://github.com/pinobatch) </sup>
* **[Mooneye Test Suite](https://github.com/Gekkio/mooneye-test-suite)**  
  <sup>by [Joonas Javanainen](https://github.com/Gekkio) </sup>
* **[SameSuite](https://github.com/LIJI32/SameSuite)**  
  <sup>by [Lior Halphon](https://github.com/LIJI32) </sup>

Different test suites use different pass/fail criteria. Some may write output to the serial port such as
[Blargg's test roms](https://github.com/retrio/gb-test-roms), others may write to the CPU registers, such as 
[Mooneye Test Suite](https://github.com/Gekkio/mooneye-test-suite) and [SameSuite](https://github.com/LIJI32/SameSuite).
If the test suite does not provide a way to automatically determine a pass/fail criteria, then the emulator's output
is compared against a reference image from a known good emulator.
<hr/>


# Test Results
| Test Suite | Pass Rate | Tests Passed | Tests Failed | Tests Total |
| --- | --- | --- | --- | --- |
| acid2 | 75% | 3 | 1 | 4 |
| bully | 50% | 1 | 1 | 2 |
| blarrg | 100% | 43 | 0 | 43 |
| little-things-gb | 100% | 4 | 0 | 4 |
| mooneye | 98% | 112 | 2 | 114 |
| samesuite | 75% | 59 | 19 | 78 |
| scribbltests | 100% | 5 | 0 | 5 |
| strikethrough | 0% | 0 | 2 | 2 |

Explore the individual tests for each suite using the table of contents below.

## Table of Contents
* [acid2](#acid2)
  * [dmg-acid2](#dmg-acid2)
  * [cgb-acid2](#cgb-acid2)
* [bully](#bully)
  * [bully](#bully)
* [blarrg](#blarrg)
  * [cpu_instrs](#cpu_instrs)
  * [cgb_sound](#cgb_sound)
  * [dmg_sound](#dmg_sound)
  * [halt_bug](#halt_bug)
  * [instr_timing](#instr_timing)
  * [interrupt_time](#interrupt_time)
  * [mem_timing](#mem_timing)
* [little-things-gb](#little-things-gb)
  * [firstwhite](#firstwhite)
  * [tellinglys](#tellinglys)
* [mooneye](#mooneye)
  * [acceptance](#acceptance)
    * [bits](#bits)
    * [instr](#instr)
    * [interrupts](#interrupts)
    * [oam_dma](#oam_dma)
    * [ppu](#ppu)
    * [serial](#serial)
    * [timer](#timer)
  * [emulator-only](#emulator-only)
    * [mbc1](#mbc1)
    * [mbc2](#mbc2)
    * [mbc5](#mbc5)
  * [madness](#madness)
  * [misc](#misc)
    * [bits](#bits)
    * [ppu](#ppu)
  * [manual-only](#manual-only)
* [samesuite](#samesuite)
  * [apu](#apu)
  * [apu/channel_1](#apu/channel_1)
  * [apu/channel_2](#apu/channel_2)
  * [apu/channel_3](#apu/channel_3)
  * [apu/channel_4](#apu/channel_4)
  * [dma](#dma)
  * [interrupt](#interrupt)
  * [ppu](#ppu)
  * [sgb](#sgb)
* [scribbltests](#scribbltests)
  * [scribbltests](#scribbltests)
* [strikethrough](#strikethrough)
  * [strikethrough](#strikethrough)

# acid2
![progress](https://progress-bar.xyz/75/?scale=100&title=passing%203,%20failing%201&width=500)
## dmg-acid2
| Test | Passing |
| ---- | ------- |
| dmg-acid2 (DMG) | ✅ |
| dmg-acid2 (CGB) | ✅ |
## cgb-acid2
| Test | Passing |
| ---- | ------- |
| cgb-acid2 | ✅ |
| cgb-acid-hell | ❌ |
# bully
![progress](https://progress-bar.xyz/50/?scale=100&title=passing%201,%20failing%201&width=500)

| Test | Passing |
| ---- | ------- |
| bully (DMG) | ✅ |
| bully (CGB) | ❌ |
# blarrg
![progress](https://progress-bar.xyz/100/?scale=100&title=passing%2043,%20failing%200&width=500)
## cpu_instrs
| Test | Passing |
| ---- | ------- |
| 01-special | ✅ |
| 02-interrupts | ✅ |
| 03-op sp,hl | ✅ |
| 04-op r,imm | ✅ |
| 05-op rp | ✅ |
| 06-ld r,r | ✅ |
| 07-jr,jp,call,ret,rst | ✅ |
| 08-misc instrs | ✅ |
| 09-op r,r | ✅ |
| 10-bit ops | ✅ |
| 11-op a,(hl) | ✅ |
## cgb_sound
| Test | Passing |
| ---- | ------- |
| 01-registers | ✅ |
| 02-len ctr | ✅ |
| 03-trigger | ✅ |
| 04-sweep | ✅ |
| 05-sweep details | ✅ |
| 06-overflow on trigger | ✅ |
| 07-len sweep period sync | ✅ |
| 08-len ctr during power | ✅ |
| 09-wave read while on | ✅ |
| 10-wave trigger while on | ✅ |
| 11-regs after power | ✅ |
| 12-wave | ✅ |
## dmg_sound
| Test | Passing |
| ---- | ------- |
| 01-registers | ✅ |
| 02-len ctr | ✅ |
| 03-trigger | ✅ |
| 04-sweep | ✅ |
| 05-sweep details | ✅ |
| 06-overflow on trigger | ✅ |
| 07-len sweep period sync | ✅ |
| 08-len ctr during power | ✅ |
| 09-wave read while on | ✅ |
| 10-wave trigger while on | ✅ |
| 11-regs after power | ✅ |
| 12-wave write while on | ✅ |
## halt_bug
| Test | Passing |
| ---- | ------- |
| halt_bug (DMG) | ✅ |
| halt_bug (CGB) | ✅ |
## instr_timing
| Test | Passing |
| ---- | ------- |
| instr_timing | ✅ |
## interrupt_time
| Test | Passing |
| ---- | ------- |
| interrupt_time (DMG) | ✅ |
| interrupt_time (CGB) | ✅ |
## mem_timing
| Test | Passing |
| ---- | ------- |
| 01-read_timing | ✅ |
| 02-write_timing | ✅ |
| 03-modify_timing | ✅ |
# little-things-gb
![progress](https://progress-bar.xyz/100/?scale=100&title=passing%204,%20failing%200&width=500)
## firstwhite
| Test | Passing |
| ---- | ------- |
| firstwhite (DMG) | ✅ |
| firstwhite (CGB) | ✅ |
## tellinglys
| Test | Passing |
| ---- | ------- |
| tellinglys (DMG) | ✅ |
| tellinglys (CGB) | ✅ |
# mooneye
![progress](https://progress-bar.xyz/98/?scale=100&title=passing%20112,%20failing%202&width=500)
## acceptance
| Test | Passing |
| ---- | ------- |
| add_sp_e_timing | ✅ |
| boot_div-S | ✅ |
| boot_div-dmg0 | ✅ |
| boot_div-dmgABCmgb | ✅ |
| boot_div2-S | ✅ |
| boot_hwio-S | ✅ |
| boot_hwio-dmg0 | ✅ |
| boot_hwio-dmgABCmgb | ✅ |
| boot_regs-dmg0 | ✅ |
| boot_regs-dmgABC | ✅ |
| boot_regs-mgb | ✅ |
| boot_regs-sgb | ✅ |
| boot_regs-sgb2 | ✅ |
| call_cc_timing | ✅ |
| call_cc_timing2 | ✅ |
| call_timing | ✅ |
| call_timing2 | ✅ |
| di_timing-GS | ✅ |
| div_timing | ✅ |
| ei_sequence | ✅ |
| ei_timing | ✅ |
| halt_ime0_ei | ✅ |
| halt_ime0_nointr_timing | ✅ |
| halt_ime1_timing | ✅ |
| halt_ime1_timing2-GS | ✅ |
| if_ie_registers | ✅ |
| intr_timing | ✅ |
| jp_cc_timing | ✅ |
| jp_timing | ✅ |
| ld_hl_sp_e_timing | ✅ |
| oam_dma_restart | ✅ |
| oam_dma_start | ✅ |
| oam_dma_timing | ✅ |
| pop_timing | ✅ |
| push_timing | ✅ |
| rapid_di_ei | ✅ |
| ret_cc_timing | ✅ |
| ret_timing | ✅ |
| reti_intr_timing | ✅ |
| reti_timing | ✅ |
| rst_timing | ✅ |
## bits
| Test | Passing |
| ---- | ------- |
| mem_oam | ✅ |
| reg_f | ✅ |
| unused_hwio-GS | ✅ |
## instr
| Test | Passing |
| ---- | ------- |
| daa | ✅ |
## interrupts
| Test | Passing |
| ---- | ------- |
| ie_push | ✅ |
## oam_dma
| Test | Passing |
| ---- | ------- |
| basic | ✅ |
| reg_read | ✅ |
| sources-GS | ✅ |
## ppu
| Test | Passing |
| ---- | ------- |
| hblank_ly_scx_timing-GS | ✅ |
| intr_1_2_timing-GS | ✅ |
| intr_2_0_timing | ✅ |
| intr_2_mode0_timing | ✅ |
| intr_2_mode0_timing_sprites | ❌ |
| intr_2_mode3_timing | ✅ |
| intr_2_oam_ok_timing | ✅ |
| lcdon_timing-GS | ✅ |
| lcdon_write_timing-GS | ✅ |
| stat_irq_blocking | ✅ |
| stat_lyc_onoff | ✅ |
| vblank_stat_intr-GS | ✅ |
## serial
| Test | Passing |
| ---- | ------- |
| boot_sclk_align-dmgABCmgb | ✅ |
## timer
| Test | Passing |
| ---- | ------- |
| div_write | ✅ |
| rapid_toggle | ✅ |
| tim00 | ✅ |
| tim00_div_trigger | ✅ |
| tim01 | ✅ |
| tim01_div_trigger | ✅ |
| tim10 | ✅ |
| tim10_div_trigger | ✅ |
| tim11 | ✅ |
| tim11_div_trigger | ✅ |
| tima_reload | ✅ |
| tima_write_reloading | ✅ |
| tma_write_reloading | ✅ |
## emulator-only
| Test | Passing |
| ---- | ------- |
## mbc1
| Test | Passing |
| ---- | ------- |
| bits_bank1 | ✅ |
| bits_bank2 | ✅ |
| bits_mode | ✅ |
| bits_ramg | ✅ |
| multicart_rom_8Mb | ✅ |
| ram_256kb | ✅ |
| ram_64kb | ✅ |
| rom_16Mb | ✅ |
| rom_1Mb | ✅ |
| rom_2Mb | ✅ |
| rom_4Mb | ✅ |
| rom_512kb | ✅ |
| rom_8Mb | ✅ |
## mbc2
| Test | Passing |
| ---- | ------- |
| bits_ramg | ✅ |
| bits_romb | ✅ |
| bits_unused | ✅ |
| ram | ✅ |
| rom_1Mb | ✅ |
| rom_2Mb | ✅ |
| rom_512kb | ✅ |
## mbc5
| Test | Passing |
| ---- | ------- |
| rom_16Mb | ✅ |
| rom_1Mb | ✅ |
| rom_2Mb | ✅ |
| rom_32Mb | ✅ |
| rom_4Mb | ✅ |
| rom_512kb | ✅ |
| rom_64Mb | ✅ |
| rom_8Mb | ✅ |
## madness
| Test | Passing |
| ---- | ------- |
| mgb_oam_dma_halt_sprites | ❌ |
## misc
| Test | Passing |
| ---- | ------- |
| boot_div-A | ✅ |
| boot_div-cgb0 | ✅ |
| boot_div-cgbABCDE | ✅ |
| boot_hwio-C | ✅ |
| boot_regs-A | ✅ |
| boot_regs-cgb | ✅ |
## bits
| Test | Passing |
| ---- | ------- |
| unused_hwio-C | ✅ |
## ppu
| Test | Passing |
| ---- | ------- |
| vblank_stat_intr-C | ✅ |
## manual-only
| Test | Passing |
| ---- | ------- |
| sprite_priority (DMG) | ✅ |
| sprite_priority (CGB) | ✅ |
# samesuite
![progress](https://progress-bar.xyz/75/?scale=100&title=passing%2059,%20failing%2019&width=500)
## apu
| Test | Passing |
| ---- | ------- |
| div_trigger_volume_10 | ✅ |
| div_write_trigger | ✅ |
| div_write_trigger_10 | ✅ |
| div_write_trigger_volume | ✅ |
| div_write_trigger_volume_10 | ✅ |
## apu/channel_1
| Test | Passing |
| ---- | ------- |
| channel_1_align | ✅ |
| channel_1_align_cpu | ✅ |
| channel_1_delay | ✅ |
| channel_1_duty | ✅ |
| channel_1_duty_delay | ✅ |
| channel_1_extra_length_clocking-cgb0B | ❌ |
| channel_1_freq_change | ✅ |
| channel_1_freq_change_timing-A | ❌ |
| channel_1_freq_change_timing-cgb0BC | ❌ |
| channel_1_freq_change_timing-cgbDE | ❌ |
| channel_1_nrx2_glitch | ✅ |
| channel_1_nrx2_speed_change | ✅ |
| channel_1_restart | ✅ |
| channel_1_restart_nrx2_glitch | ✅ |
| channel_1_stop_div | ✅ |
| channel_1_stop_restart | ✅ |
| channel_1_sweep | ❌ |
| channel_1_sweep_restart | ❌ |
| channel_1_sweep_restart_2 | ❌ |
| channel_1_volume | ✅ |
| channel_1_volume_div | ✅ |
## apu/channel_2
| Test | Passing |
| ---- | ------- |
| channel_2_align | ✅ |
| channel_2_align_cpu | ✅ |
| channel_2_delay | ✅ |
| channel_2_duty | ✅ |
| channel_2_duty_delay | ✅ |
| channel_2_extra_length_clocking-cgb0B | ❌ |
| channel_2_freq_change | ✅ |
| channel_2_nrx2_glitch | ✅ |
| channel_2_nrx2_speed_change | ✅ |
| channel_2_restart | ✅ |
| channel_2_restart_nrx2_glitch | ✅ |
| channel_2_stop_div | ✅ |
| channel_2_stop_restart | ✅ |
| channel_2_volume | ✅ |
| channel_2_volume_div | ✅ |
## apu/channel_3
| Test | Passing |
| ---- | ------- |
| channel_3_and_glitch | ✅ |
| channel_3_delay | ✅ |
| channel_3_extra_length_clocking-cgb0 | ❌ |
| channel_3_extra_length_clocking-cgbB | ❌ |
| channel_3_first_sample | ✅ |
| channel_3_freq_change_delay | ❌ |
| channel_3_restart_delay | ❌ |
| channel_3_restart_during_delay | ✅ |
| channel_3_restart_stop_delay | ✅ |
| channel_3_shift_delay | ✅ |
| channel_3_shift_skip_delay | ✅ |
| channel_3_stop_delay | ✅ |
| channel_3_stop_div | ✅ |
| channel_3_wave_ram_dac_on_rw | ✅ |
| channel_3_wave_ram_locked_write | ✅ |
| channel_3_wave_ram_sync | ✅ |
## apu/channel_4
| Test | Passing |
| ---- | ------- |
| channel_4_align | ✅ |
| channel_4_delay | ❌ |
| channel_4_equivalent_frequencies | ❌ |
| channel_4_extra_length_clocking-cgb0B | ❌ |
| channel_4_freq_change | ❌ |
| channel_4_frequency_alignment | ❌ |
| channel_4_lfsr | ✅ |
| channel_4_lfsr15 | ✅ |
| channel_4_lfsr_15_7 | ✅ |
| channel_4_lfsr_7_15 | ✅ |
| channel_4_lfsr_restart | ✅ |
| channel_4_lfsr_restart_fast | ✅ |
| channel_4_volume_div | ✅ |
## dma
| Test | Passing |
| ---- | ------- |
| gbc_dma_cont | ✅ |
| gdma_addr_mask | ✅ |
| hdma_lcd_off | ✅ |
| hdma_mode0 | ✅ |
## interrupt
| Test | Passing |
| ---- | ------- |
| ei_delay_halt | ✅ |
## ppu
| Test | Passing |
| ---- | ------- |
| blocking_bgpi_increase | ✅ |
## sgb
| Test | Passing |
| ---- | ------- |
| command_mlt_req | ❌ |
| command_mlt_req_1_incrementing | ❌ |
# scribbltests
![progress](https://progress-bar.xyz/100/?scale=100&title=passing%205,%20failing%200&width=500)

| Test | Passing |
| ---- | ------- |
| lycscx | ✅ |
| lycscy | ✅ |
| palettely | ✅ |
| scxly | ✅ |
| statcount-auto | ✅ |
# strikethrough
![progress](https://progress-bar.xyz/0/?scale=100&title=passing%200,%20failing%202&width=500)

| Test | Passing |
| ---- | ------- |
| strikethrough (DMG) | ❌ |
| strikethrough (CGB) | ❌ |
