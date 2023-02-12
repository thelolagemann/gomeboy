# Table of Contents
* [acid2](#acid2)
  * [acid2](#acid2)
* [mooneye](#mooneye)
  * [bits](#bits)
  * [instr](#instr)
  * [interrupts](#interrupts)
  * [oam_dma](#oam_dma)
  * [ppu](#ppu)
  * [serial](#serial)
  * [timer](#timer)
  * [misc](#misc)
* [samesuite](#samesuite)
  * [apu](#apu)
  * [dma](#dma)
  * [interrupt](#interrupt)
  * [ppu](#ppu)
  * [sgb](#sgb)
# acid2
## acid2
| Test | Passing |
| ---- | ------- |
| dmg-acid2 | ✅ |
| cgb-acid2 | ❌ |
| cgb-acid-hell | ❌ |
# mooneye
## bits
| Test | Passing |
| ---- | ------- |
| mem_oam.gb | ✅ |
| reg_f.gb | ✅ |
| unused_hwio-GS.gb | ❌ |
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
| intr_1_2_timing-GS.gb | ❌ |
| intr_2_0_timing.gb | ✅ |
| intr_2_mode0_timing.gb | ❌ |
| intr_2_mode0_timing_sprites.gb | ❌ |
| intr_2_mode3_timing.gb | ❌ |
| intr_2_oam_ok_timing.gb | ❌ |
| lcdon_timing-GS.gb | ❌ |
| lcdon_write_timing-GS.gb | ❌ |
| stat_irq_blocking.gb | ❌ |
| stat_lyc_onoff.gb | ❌ |
| vblank_stat_intr-GS.gb | ❌ |
## serial
| Test | Passing |
| ---- | ------- |
| boot_sclk_align-dmgABCmgb.gb | ❌ |
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
## misc
| Test | Passing |
| ---- | ------- |
| add_sp_e_timing.gb | ✅ |
| boot_div-S.gb | ❌ |
| boot_div-dmg0.gb | ❌ |
| boot_div-dmgABCmgb.gb | ❌ |
| boot_div2-S.gb | ❌ |
| boot_hwio-S.gb | ❌ |
| boot_hwio-dmg0.gb | ❌ |
| boot_hwio-dmgABCmgb.gb | ❌ |
| boot_regs-dmg0.gb | ❌ |
| boot_regs-dmgABC.gb | ✅ |
| boot_regs-mgb.gb | ❌ |
| boot_regs-sgb.gb | ❌ |
| boot_regs-sgb2.gb | ❌ |
| call_cc_timing.gb | ✅ |
| call_cc_timing2.gb | ✅ |
| call_timing.gb | ✅ |
| call_timing2.gb | ✅ |
| di_timing-GS.gb | ❌ |
| div_timing.gb | ❌ |
| ei_sequence.gb | ✅ |
| ei_timing.gb | ✅ |
| halt_ime0_ei.gb | ✅ |
| halt_ime0_nointr_timing.gb | ❌ |
| halt_ime1_timing.gb | ✅ |
| halt_ime1_timing2-GS.gb | ❌ |
| if_ie_registers.gb | ✅ |
| intr_timing.gb | ❌ |
| jp_cc_timing.gb | ✅ |
| jp_timing.gb | ✅ |
| ld_hl_sp_e_timing.gb | ✅ |
| oam_dma_restart.gb | ✅ |
| oam_dma_start.gb | ✅ |
| oam_dma_timing.gb | ✅ |
| pop_timing.gb | ❌ |
| push_timing.gb | ✅ |
| rapid_di_ei.gb | ✅ |
| ret_cc_timing.gb | ✅ |
| ret_timing.gb | ✅ |
| reti_intr_timing.gb | ✅ |
| reti_timing.gb | ✅ |
| rst_timing.gb | ✅ |
# samesuite
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
| gdma_addr_mask.gb | ❌ |
| hdma_lcd_off.gb | ❌ |
| hdma_mode0.gb | ❌ |
## interrupt
| Test | Passing |
| ---- | ------- |
| ei_delay_halt.gb | ❌ |
## ppu
| Test | Passing |
| ---- | ------- |
| blocking_bgpi_increase.gb | ❌ |
## sgb
| Test | Passing |
| ---- | ------- |
| command_mlt_req.gb | ❌ |
| command_mlt_req_1_incrementing.gb | ❌ |
