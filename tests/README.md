# Automated test results
![progress](https://progress-bar.dev/90/?scale=100&title=passing%20164,%20failing%2017&width=500)

#### This document was automatically generated from commit c9895552
<hr/>
GomeBoy is automatically tested against the following test suites:

* **[Blargg's test roms](https://github.com/retrio/gb-test-roms)**  
  <sup>by [Shay Green (a.k.a. Blargg)](http://www.slack.net/~ant/) </sup>
* **[Bully](https://github.com/Hacktix/BullyGB)**
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
| bully | 0% | 0 | 1 | 1 |
| blarrg | 100% | 43 | 0 | 43 |
| little-things-gb | 100% | 4 | 0 | 4 |
| mooneye | 94% | 108 | 6 | 114 |
| samesuite | 46% | 6 | 7 | 13 |
| strikethrough | 0% | 0 | 2 | 2 |

Explore the individual tests for each suite using the table of contents below.

## Table of Contents
* [acid2](#acid2)
  * [dmg-acid2](#dmg-acid2)
  * [cgb-acid2](#cgb-acid2)
* [bully](#bully)
  * [bully](#bully)
* [blarrg](#blarrg)
  * [cgb_sound](#cgb_sound)
  * [cpu_instrs](#cpu_instrs)
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
  * [dma](#dma)
  * [interrupt](#interrupt)
  * [ppu](#ppu)
  * [sgb](#sgb)
* [strikethrough](#strikethrough)
  * [strikethrough](#strikethrough)

# acid2
![progress](https://progress-bar.dev/75/?scale=100&title=passing%203,%20failing%201&width=500)
## dmg-acid2
| Test | Passing |
| ---- | ------- |
| dmg-acid2 | ✅ |
| dmg-acid2-cgb | ✅ |
## cgb-acid2
| Test | Passing |
| ---- | ------- |
| cgb-acid2 | ✅ |
| cgb-acid-hell | ❌ |
# bully
![progress](https://progress-bar.dev/0/?scale=100&title=passing%200,%20failing%201&width=500)

| Test | Passing |
| ---- | ------- |
| bully | ❌ |
# blarrg
![progress](https://progress-bar.dev/100/?scale=100&title=passing%2043,%20failing%200&width=500)
## cgb_sound
| Test | Passing |
| ---- | ------- |
| cgb_sound/01-registers | ✅ |
| cgb_sound/02-len ctr | ✅ |
| cgb_sound/03-trigger | ✅ |
| cgb_sound/04-sweep | ✅ |
| cgb_sound/05-sweep details | ✅ |
| cgb_sound/06-overflow on trigger | ✅ |
| cgb_sound/07-len sweep period sync | ✅ |
| cgb_sound/08-len ctr during power | ✅ |
| cgb_sound/09-wave read while on | ✅ |
| cgb_sound/10-wave trigger while on | ✅ |
| cgb_sound/11-regs after power | ✅ |
| cgb_sound/12-wave | ✅ |
## cpu_instrs
| Test | Passing |
| ---- | ------- |
| 01-special.gb | ✅ |
| 02-interrupts.gb | ✅ |
| 03-op sp,hl.gb | ✅ |
| 04-op r,imm.gb | ✅ |
| 05-op rp.gb | ✅ |
| 06-ld r,r.gb | ✅ |
| 07-jr,jp,call,ret,rst.gb | ✅ |
| 08-misc instrs.gb | ✅ |
| 09-op r,r.gb | ✅ |
| 10-bit ops.gb | ✅ |
| 11-op a,(hl).gb | ✅ |
## dmg_sound
| Test | Passing |
| ---- | ------- |
| dmg_sound/01-registers | ✅ |
| dmg_sound/02-len ctr | ✅ |
| dmg_sound/03-trigger | ✅ |
| dmg_sound/04-sweep | ✅ |
| dmg_sound/05-sweep details | ✅ |
| dmg_sound/06-overflow on trigger | ✅ |
| dmg_sound/07-len sweep period sync | ✅ |
| dmg_sound/08-len ctr during power | ✅ |
| dmg_sound/09-wave read while on | ✅ |
| dmg_sound/10-wave trigger while on | ✅ |
| dmg_sound/11-regs after power | ✅ |
| dmg_sound/12-wave write while on | ✅ |
## halt_bug
| Test | Passing |
| ---- | ------- |
| halt_bug | ✅ |
| halt_bug-cgb | ✅ |
## instr_timing
| Test | Passing |
| ---- | ------- |
| instr_timing | ✅ |
## interrupt_time
| Test | Passing |
| ---- | ------- |
| interrupt_time | ✅ |
| interrupt_time-cgb | ✅ |
## mem_timing
| Test | Passing |
| ---- | ------- |
| 01-read_timing.gb | ✅ |
| 02-write_timing.gb | ✅ |
| 03-modify_timing.gb | ✅ |
# little-things-gb
![progress](https://progress-bar.dev/100/?scale=100&title=passing%204,%20failing%200&width=500)
## firstwhite
| Test | Passing |
| ---- | ------- |
| firstwhite | ✅ |
| firstwhite-cgb | ✅ |
## tellinglys
| Test | Passing |
| ---- | ------- |
| tellinglys | ✅ |
| tellinglys-cgb | ✅ |
# mooneye
![progress](https://progress-bar.dev/94/?scale=100&title=passing%20108,%20failing%206&width=500)
## acceptance
| Test | Passing |
| ---- | ------- |
| add_sp_e_timing.gb | ✅ |
| boot_div-S.gb | ✅ |
| boot_div-dmg0.gb | ✅ |
| boot_div-dmgABCmgb.gb | ✅ |
| boot_div2-S.gb | ✅ |
| boot_hwio-S.gb | ❌ |
| boot_hwio-dmg0.gb | ❌ |
| boot_hwio-dmgABCmgb.gb | ✅ |
| boot_regs-dmg0.gb | ✅ |
| boot_regs-dmgABC.gb | ✅ |
| boot_regs-mgb.gb | ✅ |
| boot_regs-sgb.gb | ✅ |
| boot_regs-sgb2.gb | ✅ |
| call_cc_timing.gb | ✅ |
| call_cc_timing2.gb | ✅ |
| call_timing.gb | ✅ |
| call_timing2.gb | ✅ |
| di_timing-GS.gb | ✅ |
| div_timing.gb | ✅ |
| ei_sequence.gb | ✅ |
| ei_timing.gb | ✅ |
| halt_ime0_ei.gb | ✅ |
| halt_ime0_nointr_timing.gb | ✅ |
| halt_ime1_timing.gb | ✅ |
| halt_ime1_timing2-GS.gb | ✅ |
| if_ie_registers.gb | ✅ |
| intr_timing.gb | ✅ |
| jp_cc_timing.gb | ✅ |
| jp_timing.gb | ✅ |
| ld_hl_sp_e_timing.gb | ✅ |
| oam_dma_restart.gb | ✅ |
| oam_dma_start.gb | ✅ |
| oam_dma_timing.gb | ✅ |
| pop_timing.gb | ✅ |
| push_timing.gb | ✅ |
| rapid_di_ei.gb | ✅ |
| ret_cc_timing.gb | ✅ |
| ret_timing.gb | ✅ |
| reti_intr_timing.gb | ✅ |
| reti_timing.gb | ✅ |
| rst_timing.gb | ✅ |
## bits
| Test | Passing |
| ---- | ------- |
| mem_oam.gb | ✅ |
| reg_f.gb | ✅ |
| unused_hwio-GS.gb | ✅ |
## instr
| Test | Passing |
| ---- | ------- |
| daa.gb | ✅ |
## interrupts
| Test | Passing |
| ---- | ------- |
| ie_push.gb | ✅ |
## oam_dma
| Test | Passing |
| ---- | ------- |
| basic.gb | ✅ |
| reg_read.gb | ✅ |
| sources-GS.gb | ✅ |
## ppu
| Test | Passing |
| ---- | ------- |
| hblank_ly_scx_timing-GS.gb | ✅ |
| intr_1_2_timing-GS.gb | ✅ |
| intr_2_0_timing.gb | ✅ |
| intr_2_mode0_timing.gb | ✅ |
| intr_2_mode0_timing_sprites.gb | ❌ |
| intr_2_mode3_timing.gb | ✅ |
| intr_2_oam_ok_timing.gb | ✅ |
| lcdon_timing-GS.gb | ✅ |
| lcdon_write_timing-GS.gb | ✅ |
| stat_irq_blocking.gb | ✅ |
| stat_lyc_onoff.gb | ✅ |
| vblank_stat_intr-GS.gb | ✅ |
## serial
| Test | Passing |
| ---- | ------- |
| boot_sclk_align-dmgABCmgb.gb | ✅ |
## timer
| Test | Passing |
| ---- | ------- |
| div_write.gb | ✅ |
| rapid_toggle.gb | ✅ |
| tim00.gb | ✅ |
| tim00_div_trigger.gb | ✅ |
| tim01.gb | ✅ |
| tim01_div_trigger.gb | ✅ |
| tim10.gb | ✅ |
| tim10_div_trigger.gb | ✅ |
| tim11.gb | ✅ |
| tim11_div_trigger.gb | ✅ |
| tima_reload.gb | ✅ |
| tima_write_reloading.gb | ✅ |
| tma_write_reloading.gb | ✅ |
## emulator-only
| Test | Passing |
| ---- | ------- |
## mbc1
| Test | Passing |
| ---- | ------- |
| bits_bank1.gb | ✅ |
| bits_bank2.gb | ✅ |
| bits_mode.gb | ✅ |
| bits_ramg.gb | ✅ |
| multicart_rom_8Mb.gb | ✅ |
| ram_256kb.gb | ✅ |
| ram_64kb.gb | ✅ |
| rom_16Mb.gb | ✅ |
| rom_1Mb.gb | ✅ |
| rom_2Mb.gb | ✅ |
| rom_4Mb.gb | ✅ |
| rom_512kb.gb | ✅ |
| rom_8Mb.gb | ✅ |
## mbc2
| Test | Passing |
| ---- | ------- |
| bits_ramg.gb | ✅ |
| bits_romb.gb | ✅ |
| bits_unused.gb | ✅ |
| ram.gb | ✅ |
| rom_1Mb.gb | ✅ |
| rom_2Mb.gb | ✅ |
| rom_512kb.gb | ✅ |
## mbc5
| Test | Passing |
| ---- | ------- |
| rom_16Mb.gb | ✅ |
| rom_1Mb.gb | ✅ |
| rom_2Mb.gb | ✅ |
| rom_32Mb.gb | ✅ |
| rom_4Mb.gb | ✅ |
| rom_512kb.gb | ✅ |
| rom_64Mb.gb | ✅ |
| rom_8Mb.gb | ✅ |
## madness
| Test | Passing |
| ---- | ------- |
| mgb_oam_dma_halt_sprites | ❌ |
## misc
| Test | Passing |
| ---- | ------- |
| boot_div-A.gb | ✅ |
| boot_div-cgb0.gb | ✅ |
| boot_div-cgbABCDE.gb | ✅ |
| boot_hwio-C.gb | ❌ |
| boot_regs-A.gb | ✅ |
| boot_regs-cgb.gb | ✅ |
## bits
| Test | Passing |
| ---- | ------- |
| unused_hwio-C.gb | ✅ |
## ppu
| Test | Passing |
| ---- | ------- |
| vblank_stat_intr-C.gb | ✅ |
## manual-only
| Test | Passing |
| ---- | ------- |
| sprite_priority | ✅ |
| sprite_priority | ❌ |
# samesuite
![progress](https://progress-bar.dev/46/?scale=100&title=passing%206,%20failing%207&width=500)
## apu
| Test | Passing |
| ---- | ------- |
| div_trigger_volume_10.gb | ❌ |
| div_write_trigger.gb | ❌ |
| div_write_trigger_10.gb | ❌ |
| div_write_trigger_volume.gb | ❌ |
| div_write_trigger_volume_10.gb | ❌ |
## dma
| Test | Passing |
| ---- | ------- |
| gbc_dma_cont.gb | ✅ |
| gdma_addr_mask.gb | ✅ |
| hdma_lcd_off.gb | ✅ |
| hdma_mode0.gb | ✅ |
## interrupt
| Test | Passing |
| ---- | ------- |
| ei_delay_halt.gb | ✅ |
## ppu
| Test | Passing |
| ---- | ------- |
| blocking_bgpi_increase.gb | ✅ |
## sgb
| Test | Passing |
| ---- | ------- |
| command_mlt_req.gb | ❌ |
| command_mlt_req_1_incrementing.gb | ❌ |
# strikethrough
![progress](https://progress-bar.dev/0/?scale=100&title=passing%200,%20failing%202&width=500)

| Test | Passing |
| ---- | ------- |
| strikethrough | ❌ |
| strikethrough | ❌ |
