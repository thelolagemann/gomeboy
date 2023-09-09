package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

var (
	mealybugTests = []ROMTest{
		newImageTest("m2_win_en_toggle"),
		newImageTest("m2_win_en_toggle", asModel(types.CGBABC)),
		newImageTest("m3_bgp_change"),
		newImageTest("m3_bgp_change", asModel(types.CGBABC)),
		newImageTest("m3_bgp_change_sprites"),
		newImageTest("m3_bgp_change_sprites", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_bg_en_change"),
		newImageTest("m3_lcdc_bg_en_change", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_bg_en_change2", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_bg_map_change"),
		newImageTest("m3_lcdc_bg_map_change", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_obj_en_change"),
		newImageTest("m3_lcdc_obj_en_change", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_obj_en_change_variant"),
		newImageTest("m3_lcdc_obj_en_change_variant", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_obj_size_change"),
		newImageTest("m3_lcdc_obj_size_change", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_obj_size_change_scx"),
		newImageTest("m3_lcdc_obj_size_change_scx", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_tile_sel_change"),
		newImageTest("m3_lcdc_tile_sel_change", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_tile_sel_change2", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_tile_sel_win_change"),
		newImageTest("m3_lcdc_tile_sel_win_change", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_tile_sel_win_change2", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_win_en_change_multiple"),
		newImageTest("m3_lcdc_win_en_change_multiple", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_win_en_change_multiple_wx"),
		newImageTest("m3_lcdc_win_en_change_multiple_wx", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_win_map_change"),
		newImageTest("m3_lcdc_win_map_change", asModel(types.CGBABC)),
		newImageTest("m3_lcdc_win_map_change2", asModel(types.CGBABC)),
		newImageTest("m3_obp0_change"),
		newImageTest("m3_obp0_change", asModel(types.CGBABC)),
		newImageTest("m3_scx_high_5_bits"),
		newImageTest("m3_scx_high_5_bits", asModel(types.CGBABC)),
		newImageTest("m3_scx_high_5_bits_change2", asModel(types.CGBABC)),
		newImageTest("m3_scx_low_3_bits"),
		newImageTest("m3_scx_low_3_bits", asModel(types.CGBABC)),
		newImageTest("m3_scy_change"),
		newImageTest("m3_scy_change", asModel(types.CGBABC)),
		newImageTest("m3_scy_change2", asModel(types.CGBABC)),
		newImageTest("m3_window_timing"),
		newImageTest("m3_window_timing", asModel(types.CGBABC)),
		newImageTest("m3_window_timing_wx_0"),
		newImageTest("m3_window_timing_wx_0", asModel(types.CGBABC)),
		newImageTest("m3_wx_4_change"),
		newImageTest("m3_wx_4_change_sprites"),
		newImageTest("m3_wx_4_change_sprites", asModel(types.CGBABC)),
		newImageTest("m3_wx_5_change"),
		newImageTest("m3_wx_6_change"),
	}
)

func Test_Mealybug(t *testing.T) {
	testROMs(t, mealybugTests...)
}

func testMealybug(table *TestTable) {
	tS := table.NewTestSuite("mealybug-tearoom-tests")

	ppu := tS.NewTestCollection("ppu")
	ppu.AddTests(mealybugTests...)
}
