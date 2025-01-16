package server

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/google/go-cmp/cmp"
	"github.com/heroiclabs/nakama/v3/server/evr"
	"go.uber.org/zap"
)

func createTestDiscordGoSession(t *testing.T, logger *zap.Logger) *discordgo.Session {
	return &discordgo.Session{}
}

func createTestProfileRegistry(t *testing.T, logger *zap.Logger) (*ProfileCache, error) {
	runtimeLogger := NewRuntimeGoLogger(logger)

	db := NewDB(t)
	nk := NewRuntimeGoNakamaModule(logger, db, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	profileRegistry := NewProfileRegistry(nk, db, runtimeLogger, metrics, nil)

	return profileRegistry, nil
}

func TestProfileRegistry(t *testing.T) {
	consoleLogger := loggerForTest(t)

	profileRegistry, err := createTestProfileRegistry(t, consoleLogger)
	if err != nil {
		t.Fatalf("error creating test match registry: %v", err)
	}
	_ = profileRegistry
}

func TestConvertWalletToCosmetics(t *testing.T) {
	tests := []struct {
		name     string
		wallet   map[string]int64
		expected map[string]map[string]bool
	}{
		{
			name:     "Empty wallet",
			wallet:   map[string]int64{},
			expected: map[string]map[string]bool{},
		},
		{
			name: "Single cosmetic item",
			wallet: map[string]int64{
				"cosmetic:arena:rwd_tag_s1_vrml_s1": 1,
			},
			expected: map[string]map[string]bool{
				"arena": {
					"rwd_tag_s1_vrml_s1": true,
				},
			},
		},
		{
			name: "Multiple cosmetic items",
			wallet: map[string]int64{
				"cosmetic:arena:rwd_tag_s1_vrml_s1": 1,
				"cosmetic:arena:rwd_tag_s1_vrml_s2": 1,
			},
			expected: map[string]map[string]bool{
				"arena": {
					"rwd_tag_s1_vrml_s1": true,
					"rwd_tag_s1_vrml_s2": true,
				},
			},
		},
		{
			name: "Non-cosmetic items",
			wallet: map[string]int64{
				"noncosmetic:item1":                 1,
				"cosmetic:arena:rwd_tag_s1_vrml_s2": 1,
			},
			expected: map[string]map[string]bool{
				"arena": {
					"rwd_tag_s1_vrml_s2": true,
				},
			},
		},
		{
			name: "Cosmetic item with zero quantity",
			wallet: map[string]int64{
				"cosmetic:arena:rwd_tag_s1_vrml_s1": 0,
			},
			expected: map[string]map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unlocks := walletToCosmetics(tt.wallet, nil)
			if cmp.Diff(unlocks, tt.expected) != "" {
				wantGotDiff(t, tt.expected, unlocks)
			}
		})
	}
}

func wantGotDiff(t *testing.T, want, got interface{}) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestServerProfileGeneration(t *testing.T) {

	wantMap := map[string]any{}

	err := json.Unmarshal([]byte(serverProfile), &wantMap)
	if err != nil {
		t.Fatalf("error unmarshalling server profile: %v", err)
	}

	want, err := json.MarshalIndent(wantMap, "", "    ")
	if err != nil {
		t.Fatalf("error marshalling server profile: %v", err)
	}

	var profile evr.ServerProfile
	if err := json.Unmarshal(want, &profile); err != nil {
		t.Fatalf("error unmarshalling profile: %v", err)
	}

	got, err := json.MarshalIndent(profile, "", "    ")
	if err != nil {
		t.Fatalf("error marshalling profile: %v", err)
	}

	//output the differences
	gotMap := map[string]any{}
	if err := json.Unmarshal(got, &gotMap); err != nil {
		t.Fatalf("error unmarshalling got profile: %v", err)
	}

	for key, prop := range wantMap {
		if value, ok := gotMap[key]; !ok {
			t.Errorf("missing key: %v", key)
		} else if !reflect.DeepEqual(prop, value) {
			t.Errorf("(-/+) want/got: \n%s", cmp.Diff(prop, value))
		}
	}
}

var serverProfile = `
{
	"displayname": "SomePerson",
	"xplatformid": "OVR-ORG-123412341234",
	"_version": 5,
	"publisher_lock": "rad15_live",
	"purchasedcombat": 0,
	"lobbyversion": 1680630467,
	"stats": {
		"arena": {
			"Level": {
				"cnt": 1,
				"op": "add",
				"val": 1
			}
		},
		"combat": {
			"Level": {
				"cnt": 1,
				"op": "add",
				"val": 1
			}
		}
	},
	"unlocks": {
		"arena": {
			"decal_combat_flamingo_a": true,
			"decal_combat_logo_a": true,
			"decal_default": true,
			"decal_sheldon_a": true,
			"emote_blink_smiley_a": true,
			"emote_default": true,
			"emote_dizzy_eyes_a": true,
			"loadout_number": true,
			"pattern_default": true,
			"pattern_lightning_a": true,
			"rwd_banner_s1_default": true,
			"rwd_booster_default": true,
			"rwd_bracer_default": true,
			"rwd_chassis_body_s11_a": true,
			"rwd_decalback_default": true,
			"rwd_decalborder_default": true,
			"rwd_medal_default": true,
			"rwd_tag_default": true,
			"rwd_tag_s1_a_secondary": true,
			"rwd_title_title_default": true,
			"tint_blue_a_default": true,
			"tint_neutral_a_default": true,
			"tint_neutral_a_s10_default": true,
			"tint_orange_a_default": true,
			"rwd_goal_fx_default": true,
			"emissive_default": true,
			"pattern_triangles_a": true,
			"rwd_tag_s1_i_secondary": true,
			"pattern_honeycomb_triple_a": true,
			"rwd_title_title_e": true,
			"rwd_tag_s1_o_secondary": true,
			"tint_orange_j_default": true,
			"tint_neutral_l_default": true,
			"tint_orange_e_default": true,
			"pattern_angles_a": true,
			"decal_ray_gun_a": true,
			"rwd_tag_s1_e_secondary": true,
			"rwd_pattern_salt_a": true,
			"pattern_squiggles_a": true,
			"tint_orange_i_default": true,
			"pattern_weave_a": true,
			"tint_blue_k_default": true,
			"emote_winky_tongue_a": true,
			"emote_stinky_poop_a": true,
			"decal_rose_a": true,
			"decal_music_note_a": true,
			"tint_blue_h_default": true,
			"decal_bomb_a": true,
			"emote_sick_face_a": true,
			"decal_alien_head_a": true,
			"tint_neutral_e_default": true,
			"rwd_emissive_0011": true,
			"decal_combat_pig_a": true,
			"decal_disc_a": true,
			"tint_orange_d_default": true,
			"heraldry_default": true,
			"tint_blue_b_default": true,
			"tint_neutral_d_default": true,
			"rwd_emissive_0014": true,
			"tint_chassis_default": true,
			"decal_dinosaur_a": true,
			"rwd_tag_s1_h_secondary": true,
			"decal_spider_a": true,
			"decal_combat_meteor_a": true,
			"rwd_pip_0007": true,
			"pattern_inset_cubes_a": true,
			"tint_orange_f_default": true,
			"pattern_bananas_a": true,
			"rwd_banner_s1_basic": true,
			"rwd_banner_s1_bold_stripe": true,
			"rwd_emissive_0012": true,
			"rwd_pattern_rage_wolf_a": true,
			"rwd_tag_s1_m_secondary": true,
			"decal_combat_pulsar_a": true,
			"tint_neutral_k_default": true,
			"emote_sleepy_zzz_a": true,
			"tint_neutral_n_default": true,
			"pattern_digital_camo_a": true,
			"emote_kissy_lips_a": true,
			"tint_orange_k_default": true,
			"rwd_pip_0010": true,
			"tint_neutral_g_default": true,
			"rwd_tag_s1_k_secondary": true,
			"tint_orange_g_default": true,
			"rwd_pip_0001": true,
			"decal_combat_rage_bear_a": true,
			"rwd_tag_s1_b_secondary": true,
			"decal_lightning_bolt_a": true,
			"decal_koi_fish_a": true,
			"rwd_banner_s1_chevrons": true,
			"emote_exclamation_point_a": true,
			"emote_angry_face_a": true,
			"pattern_gears_a": true,
			"tint_blue_j_default": true,
			"rwd_emissive_0010": true,
			"rwd_title_title_d": true,
			"rwd_goal_fx_0002": true,
			"decal_crosshair_a": true,
			"pattern_scales_a": true,
			"decal_bullseye_a": true,
			"emote_crying_face_a": true,
			"decal_combat_demon_a": true,
			"tint_blue_f_default": true,
			"rwd_tag_s1_j_secondary": true,
			"emote_hourglass_a": true,
			"rwd_medal_s1_arena_silver": true,
			"emote_broken_heart_a": true,
			"tint_blue_c_default": true,
			"pattern_tiger_a": true,
			"tint_neutral_f_default": true,
			"tint_blue_i_default": true,
			"tint_orange_b_default": true,
			"rwd_pip_0014": true,
			"emote_wifi_symbol_a": true,
			"emote_clock_a": true,
			"decal_rage_wolf_a": true,
			"rwd_booster_s11_s1_a_retro": true,
			"emote_dead_face_a": true,
			"pattern_leopard_a": true,
			"pattern_diamond_plate_a": true,
			"emote_heart_eyes_a": true,
			"decal_eagle_a": true,
			"pattern_hawaiian_a": true,
			"rwd_pip_0011": true,
			"emote_tear_drop_a": true,
			"rwd_pip_0005": true,
			"emote_moustache_a": true,
			"decal_salt_shaker_a": true,
			"rwd_pattern_pizza_a": true,
			"pattern_pineapple_a": true,
			"emote_pizza_dance": true,
			"rwd_pip_0006": true,
			"pattern_streaks_a": true,
			"decal_fireball_a": true,
			"rwd_tag_s1_v_secondary": true,
			"emote_eye_roll_a": true,
			"decal_combat_pizza_a": true,
			"rwd_pattern_cupcake_a": true,
			"rwd_emissive_0007": true,
			"decal_radioactive_a": true,
			"rwd_emissive_0025": true,
			"pattern_strings_a": true,
			"emote_star_eyes_a": true,
			"pattern_arrowheads_a": true,
			"rwd_booster_vintage_a": true,
			"decal_saturn_a": true,
			"pattern_paws_a": true,
			"decal_swords_a": true,
			"rwd_emissive_0004": true,
			"rwd_pattern_hamburger_a": true,
			"pattern_treads_a": true,
			"decal_rocket_a": true,
			"rwd_bracer_vintage_a": true,
			"rwd_tag_s1_c_secondary": true,
			"tint_blue_d_default": true,
			"decal_combat_skull_crossbones_a": true,
			"decal_radioactive_bio_a": true,
			"rwd_pip_0008": true,
			"rwd_pip_0009": true,
			"rwd_pip_0013": true,
			"pattern_dumbbells_a": true,
			"rwd_goal_fx_0008": true,
			"rwd_pip_0015": true,
			"rwd_emissive_0006": true,
			"rwd_emissive_0001": true,
			"rwd_pattern_skull_a": true,
			"rwd_pattern_alien_a": true,
			"decal_combat_trex_skull_a": true,
			"pattern_cats_a": true,
			"pattern_dots_a": true,
			"rwd_emissive_0008": true,
			"rwd_emissive_0009": true,
			"emote_money_bag_a": true,
			"rwd_emissive_0002": true,
			"rwd_chassis_s11_retro_a": true,
			"rwd_emissive_0003": true,
			"decal_combat_flying_saucer_a": true,
			"rwd_emissive_0005": true,
			"rwd_emissive_0013": true,
			"rwd_medal_s1_arena_gold": true,
			"rwd_banner_s1_tritip": true,
			"decal_combat_medic_a": true,
			"decal_combat_comet_a": true,
			"decal_combat_puppy_a": true,
			"rwd_booster_s11_s1_a_fire": true,
			"emote_reticle_a": true,
			"decal_combat_octopus_a": true,
			"tint_blue_e_default": true,
			"rwd_banner_s1_squish": true,
			"decal_hamburger_a": true,
			"emote_skull_crossbones_a": true,
			"emote_gg_a": true,
			"pattern_cubes_a": true,
			"pattern_swirl_a": true,
			"decal_bear_paw_a": true,
			"pattern_stars_a": true,
			"tint_neutral_j_default": true,
			"emote_dollar_eyes_a": true,
			"rwd_chassis_s8b_a": true,
			"emote_loading_a": true,
			"rwd_chassis_s11_flame_a": true,
			"decal_combat_military_badge_a": true,
			"decal_cat_a": true,
			"pattern_tablecloth_a": true,
			"rwd_banner_s1_hourglass": true,
			"tint_blue_g_default": true,
			"rwd_pattern_trex_skull_a": true,
			"rwd_tag_s1_f_secondary": true,
			"decal_combat_ice_cream_a": true,
			"pattern_diamonds_a": true,
			"tint_neutral_c_default": true,
			"tint_neutral_i_default": true,
			"rwd_goal_fx_0005": true,
			"decal_profile_wolf_a": true,
			"rwd_goal_fx_0010": true,
			"rwd_goal_fx_0011": true,
			"rwd_pattern_rocket_a": true,
			"emote_lightbulb_a": true,
			"rwd_title_title_c": true,
			"rwd_title_title_a": true,
			"pattern_chevron_a": true,
			"tint_orange_h_default": true,
			"decal_combat_nova_a": true,
			"decal_combat_lion_a": true,
			"emote_question_mark_a": true,
			"rwd_tag_s1_d_secondary": true,
			"tint_neutral_h_default": true,
			"decal_cupcake_a": true,
			"decal_skull_a": true,
			"emote_flying_hearts_a": true,
			"decal_crown_a": true,
			"decal_combat_scratch_a": true,
			"rwd_medal_s1_arena_bronze": true,
			"tint_neutral_b_default": true,
			"emote_star_sparkles_a": true,
			"tint_orange_c_default": true,
			"emote_smirk_face_a": true,
			"rwd_chassis_mako_s1_a": true,
			"rwd_emote_battery_s1_a": true,
			"rwd_decal_pepper_a": true,
			"rwd_banner_s1_digi": true,
			"rwd_bracer_mako_s1_a": true,
			"rwd_tag_s1_t_secondary": true,
			"rwd_tint_s1_c_default": true,
			"rwd_title_s1_a": true,
			"rwd_xp_boost_individual_s01_01": true,
			"rwd_booster_mako_s1_a": true,
			"rwd_emote_coffee_s1_a": true,
			"rwd_xp_boost_group_s01_01": true,
			"rwd_decal_gg_a": true,
			"rwd_medal_s1_echo_pass_bronze": true,
			"rwd_currency_s01_01": true,
			"rwd_xp_boost_individual_s01_02": true,
			"rwd_banner_s1_flames": true,
			"rwd_pattern_s1_b": true,
			"rwd_xp_boost_group_s01_02": true,
			"rwd_booster_arcade_s1_a": true,
			"rwd_title_s1_b": true,
			"rwd_xp_boost_individual_s01_03": true,
			"rwd_emote_meteor_s1_a": true,
			"rwd_tag_s1_q_secondary": true,
			"rwd_currency_s01_02": true,
			"rwd_xp_boost_group_s01_03": true,
			"rwd_medal_s1_echo_pass_silver": true,
			"rwd_xp_boost_individual_s01_04": true,
			"rwd_decal_cherry_blossom_a": true,
			"rwd_bracer_arcade_s1_a": true,
			"rwd_banner_s1_trex": true,
			"rwd_xp_boost_group_s01_04": true,
			"rwd_tint_s1_d_default": true,
			"rwd_pattern_s1_c": true,
			"rwd_currency_s01_03": true,
			"rwd_decal_ramen_a": true,
			"rwd_banner_s1_tattered": true,
			"rwd_xp_boost_individual_s01_05": true,
			"rwd_bracer_arcade_var_s1_a": true,
			"rwd_booster_trex_s1_a": true,
			"rwd_xp_boost_group_s01_05": true,
			"rwd_tag_s1_g_secondary": true,
			"rwd_pattern_s1_d": true,
			"rwd_currency_s01_04": true,
			"rwd_bracer_trex_s1_a": true,
			"rwd_banner_s1_wings": true,
			"rwd_title_s1_c": true,
			"rwd_medal_s1_echo_pass_gold": true,
			"rwd_booster_arcade_var_s1_a": true,
			"rwd_chassis_trex_s1_a": true,
			"rwd_chassis_automaton_s2_a": true,
			"emote_shifty_eyes_s2_a": true,
			"rwd_tint_s1_a_default": true,
			"rwd_banner_s2_deco": true,
			"rwd_bracer_automaton_s2_a": true,
			"rwd_decal_scarab_s2_a": true,
			"rwd_pattern_s1_a": true,
			"rwd_title_s2_a": true,
			"rwd_xp_boost_individual_s02_01": true,
			"rwd_booster_automaton_s2_a": true,
			"emote_sound_wave_s2_a": true,
			"rwd_xp_boost_group_s02_01": true,
			"rwd_tint_s2_c_default": true,
			"rwd_medal_s2_echo_pass_bronze": true,
			"rwd_currency_s02_01": true,
			"rwd_xp_boost_individual_s02_02": true,
			"rwd_banner_s2_gears": true,
			"rwd_pattern_s2_b": true,
			"rwd_xp_boost_group_s02_02": true,
			"rwd_bracer_ladybug_s2_a": true,
			"rwd_title_s2_b": true,
			"rwd_xp_boost_individual_s02_03": true,
			"rwd_tint_s2_b_default": true,
			"rwd_tag_s2_b_secondary": true,
			"rwd_currency_s02_02": true,
			"rwd_xp_boost_group_s02_03": true,
			"rwd_banner_s2_pyramids": true,
			"rwd_xp_boost_individual_s02_04": true,
			"emote_uwu_s2_a": true,
			"rwd_medal_s2_echo_pass_silver": true,
			"rwd_booster_ladybug_s2_a": true,
			"rwd_xp_boost_group_s02_04": true,
			"rwd_tag_s2_g_secondary": true,
			"rwd_decal_gears_s2_a": true,
			"rwd_currency_s02_03": true,
			"rwd_pattern_s2_c": true,
			"rwd_banner_s2_ladybug": true,
			"rwd_xp_boost_individual_s02_05": true,
			"rwd_bracer_bee_s2_a": true,
			"rwd_booster_anubis_s2_a": true,
			"rwd_xp_boost_group_s02_05": true,
			"rwd_title_s2_c": true,
			"rwd_tag_s2_h_secondary": true,
			"rwd_currency_s02_04": true,
			"rwd_bracer_anubis_s2_a": true,
			"rwd_decal_axolotl_s2_a": true,
			"rwd_medal_s2_echo_pass_gold": true,
			"rwd_banner_s2_squares": true,
			"rwd_booster_bee_s2_a": true,
			"rwd_chassis_anubis_s2_a": true,
			"rwd_chassis_spartan_a": true,
			"rwd_tint_s3_tint_a": true,
			"rwd_emote_lightning_a": true,
			"rwd_banner_triangles_a": true,
			"rwd_bracer_spartan_a": true,
			"rwd_pattern_circuit_board_a": true,
			"rwd_title_guardian_a": true,
			"rwd_tag_diamonds_a": true,
			"rwd_xp_boost_individual_s03_01": true,
			"rwd_booster_spartan_a": true,
			"rwd_decal_narwhal_a": true,
			"rwd_xp_boost_group_s03_01": true,
			"rwd_tint_s3_tint_b": true,
			"rwd_medal_s3_echo_pass_bronze_a": true,
			"rwd_currency_s03_01": true,
			"rwd_xp_boost_individual_s03_02": true,
			"rwd_bracer_lazurlite_a": true,
			"rwd_emote_battle_cry_a": true,
			"rwd_xp_boost_group_s03_02": true,
			"rwd_banner_spartan_shield_a": true,
			"rwd_pattern_spear_shield_a": true,
			"rwd_xp_boost_individual_s03_03": true,
			"rwd_booster_lazurlite_a": true,
			"rwd_tint_s3_tint_c": true,
			"rwd_currency_s03_02": true,
			"rwd_title_shield_bearer_a": true,
			"rwd_xp_boost_group_s03_03": true,
			"rwd_decal_spartan_a": true,
			"rwd_bracer_aurum_a": true,
			"rwd_tag_spear_a": true,
			"rwd_xp_boost_individual_s03_04": true,
			"rwd_medal_s3_echo_pass_silver_a": true,
			"rwd_xp_boost_group_s03_04": true,
			"rwd_booster_aurum_a": true,
			"rwd_emote_samurai_mask_a": true,
			"rwd_tint_s3_tint_d": true,
			"rwd_banner_sashimono_a": true,
			"rwd_currency_s03_03": true,
			"rwd_pattern_seigaiha_a": true,
			"rwd_xp_boost_individual_s03_05": true,
			"rwd_title_ronin_a": true,
			"rwd_bracer_samurai_a": true,
			"rwd_xp_boost_group_s03_05": true,
			"rwd_decal_oni_a": true,
			"rwd_booster_samurai_a": true,
			"rwd_tint_s3_tint_e": true,
			"rwd_medal_s3_echo_pass_gold_a": true,
			"rwd_tag_tori_a": true,
			"rwd_currency_s03_04": true,
			"rwd_chassis_samurai_a": true,
			"rwd_chassis_streetwear_a": true,
			"rwd_banner_0000": true,
			"rwd_emote_0000": true,
			"rwd_tag_0000": true,
			"rwd_bracer_streetwear_a": true,
			"rwd_tint_0000": true,
			"rwd_title_0000": true,
			"rwd_pattern_0000": true,
			"rwd_xp_boost_individual_s04_01": true,
			"rwd_booster_streetwear_a": true,
			"rwd_decal_0000": true,
			"rwd_xp_boost_group_s04_01": true,
			"rwd_banner_0001": true,
			"rwd_medal_0000": true,
			"rwd_currency_s04_01": true,
			"rwd_xp_boost_individual_s04_02": true,
			"rwd_bracer_rover_a": true,
			"rwd_tag_0001": true,
			"rwd_xp_boost_group_s04_02": true,
			"rwd_emote_0001": true,
			"rwd_tint_0001": true,
			"rwd_xp_boost_individual_s04_03": true,
			"rwd_booster_rover_a": true,
			"rwd_pattern_0001": true,
			"rwd_currency_s04_02": true,
			"rwd_title_0001": true,
			"rwd_xp_boost_group_s04_03": true,
			"rwd_banner_0002": true,
			"rwd_bracer_rover_a_deco": true,
			"rwd_decal_0001": true,
			"rwd_xp_boost_individual_s04_04": true,
			"rwd_medal_0001": true,
			"rwd_xp_boost_group_s04_04": true,
			"rwd_booster_rover_a_deco": true,
			"rwd_tag_s2_c": true,
			"rwd_pattern_0002": true,
			"rwd_emote_0002": true,
			"rwd_currency_s04_03": true,
			"rwd_tint_0002": true,
			"rwd_xp_boost_individual_s04_05": true,
			"rwd_title_0002": true,
			"rwd_bracer_funk_a": true,
			"rwd_xp_boost_group_s04_05": true,
			"rwd_banner_0003": true,
			"rwd_booster_funk_a": true,
			"rwd_decal_0002": true,
			"rwd_medal_0002": true,
			"rwd_tag_0003": true,
			"rwd_currency_s04_04": true,
			"rwd_chassis_funk_a": true,
			"rwd_currency_s05_01": true,
			"rwd_currency_s05_02": true,
			"rwd_currency_s05_03": true,
			"rwd_currency_s05_04": true,
			"rwd_xp_boost_individual_s05_01": true,
			"rwd_xp_boost_individual_s05_02": true,
			"rwd_xp_boost_individual_s05_03": true,
			"rwd_xp_boost_individual_s05_04": true,
			"rwd_xp_boost_individual_s05_05": true,
			"rwd_xp_boost_group_s05_01": true,
			"rwd_xp_boost_group_s05_02": true,
			"rwd_xp_boost_group_s05_03": true,
			"rwd_xp_boost_group_s05_04": true,
			"rwd_xp_boost_group_s05_05": true,
			"rwd_chassis_junkyard_a": true,
			"rwd_banner_0008": true,
			"rwd_emote_0005": true,
			"rwd_tag_0012": true,
			"rwd_bracer_junkyard_a": true,
			"rwd_tint_0007": true,
			"rwd_title_0004": true,
			"rwd_pattern_0005": true,
			"rwd_booster_junkyard_a": true,
			"rwd_decal_0005": true,
			"rwd_banner_0009": true,
			"rwd_medal_0003": true,
			"rwd_bracer_nuclear_a": true,
			"rwd_tag_0013": true,
			"rwd_emote_0006": true,
			"rwd_tint_0008": true,
			"rwd_booster_nuclear_a": true,
			"rwd_pattern_0006": true,
			"rwd_title_0005": true,
			"rwd_banner_0010": true,
			"rwd_bracer_nuclear_a_hydro": true,
			"rwd_decal_0006": true,
			"rwd_medal_0004": true,
			"rwd_booster_nuclear_a_hydro": true,
			"rwd_tag_0014": true,
			"rwd_emote_0007": true,
			"rwd_tint_0009": true,
			"rwd_title_0006": true,
			"rwd_bracer_wasteland_a": true,
			"rwd_banner_0011": true,
			"rwd_booster_wasteland_a": true,
			"rwd_decal_0007": true,
			"rwd_medal_0005": true,
			"rwd_tag_0015": true,
			"rwd_chassis_wasteland_a": true,
			"rwd_pattern_s2_a": true,
			"rwd_currency_s06_01": true,
			"rwd_currency_s06_02": true,
			"rwd_currency_s06_03": true,
			"rwd_currency_s06_04": true,
			"rwd_xp_boost_individual_s06_01": true,
			"rwd_xp_boost_individual_s06_02": true,
			"rwd_xp_boost_individual_s06_03": true,
			"rwd_xp_boost_individual_s06_04": true,
			"rwd_xp_boost_individual_s06_05": true,
			"rwd_xp_boost_group_s06_01": true,
			"rwd_xp_boost_group_s06_02": true,
			"rwd_xp_boost_group_s06_03": true,
			"rwd_xp_boost_group_s06_04": true,
			"rwd_xp_boost_group_s06_05": true,
			"rwd_banner_0015": true,
			"rwd_banner_0014": true,
			"rwd_banner_0016": true,
			"rwd_tag_0018": true,
			"rwd_tag_0019": true,
			"rwd_tag_0020": true,
			"rwd_banner_0017": true,
			"rwd_tag_0021": true,
			"rwd_decal_0010": true,
			"rwd_decal_0011": true,
			"rwd_decal_0012": true,
			"rwd_medal_0009": true,
			"rwd_medal_0010": true,
			"rwd_medal_0011": true,
			"rwd_tint_0013": true,
			"rwd_tint_0014": true,
			"rwd_tint_0015": true,
			"rwd_pattern_0009": true,
			"rwd_pattern_0010": true,
			"rwd_emote_0010": true,
			"rwd_emote_0011": true,
			"rwd_emote_0012": true,
			"rwd_title_0007": true,
			"rwd_title_0008": true,
			"rwd_title_0009": true,
			"rwd_bracer_shark_a": true,
			"rwd_booster_shark_a": true,
			"rwd_chassis_shark_a": true,
			"rwd_pattern_0008": true,
			"rwd_booster_covenant_a": true,
			"rwd_bracer_covenant_a": true,
			"rwd_bracer_covenant_a_flame": true,
			"rwd_booster_covenant_a_flame": true,
			"rwd_chassis_scuba_a": true,
			"rwd_bracer_scuba_a": true,
			"rwd_booster_scuba_a": true,
			"rwd_currency_s07_01": true,
			"rwd_currency_s07_02": true,
			"rwd_currency_s07_03": true,
			"rwd_currency_s07_04": true,
			"rwd_xp_boost_individual_s07_01": true,
			"rwd_xp_boost_individual_s07_02": true,
			"rwd_xp_boost_individual_s07_03": true,
			"rwd_xp_boost_individual_s07_04": true,
			"rwd_xp_boost_individual_s07_05": true,
			"rwd_xp_boost_group_s07_01": true,
			"rwd_xp_boost_group_s07_02": true,
			"rwd_xp_boost_group_s07_03": true,
			"rwd_xp_boost_group_s07_04": true,
			"rwd_xp_boost_group_s07_05": true,
			"rwd_banner_0023": true,
			"rwd_title_0010": true,
			"rwd_decal_0015": true,
			"rwd_pattern_0016": true,
			"rwd_medal_0012": true,
			"rwd_banner_0024": true,
			"rwd_medal_0015": true,
			"rwd_tint_0020": true,
			"rwd_booster_fume_a": true,
			"rwd_title_0011": true,
			"rwd_tint_0021": true,
			"rwd_bracer_fume_a": true,
			"rwd_medal_0016": true,
			"rwd_emote_0016": true,
			"rwd_medal_0014": true,
			"rwd_decal_0016": true,
			"rwd_tag_0028": true,
			"rwd_tag_0029": true,
			"rwd_pattern_0017": true,
			"rwd_tint_0022": true,
			"rwd_tag_0030": true,
			"rwd_bracer_noble_a": true,
			"rwd_medal_0013": true,
			"rwd_title_0012": true,
			"rwd_booster_noble_a": true,
			"rwd_banner_0025": true,
			"rwd_emote_0017": true,
			"rwd_chassis_noble_a": true,
			"rwd_title_0013": true,
			"rwd_tint_0023": true,
			"rwd_tag_0031": true,
			"rwd_decal_0018": true,
			"rwd_pattern_0018": true,
			"rwd_banner_0021": true,
			"rwd_emote_0019": true,
			"rwd_title_0014": true,
			"rwd_emote_0018": true,
			"rwd_tint_0024": true,
			"rwd_decal_0019": true,
			"rwd_tag_0027": true,
			"rwd_decal_0020": true,
			"rwd_pattern_0019": true,
			"rwd_banner_0022": true,
			"rwd_emote_0020": true,
			"rwd_title_0015": true,
			"rwd_tint_0025": true,
			"rwd_chassis_plagueknight_a": true,
			"rwd_bracer_plagueknight_a": true,
			"rwd_booster_plagueknight_a": true,
			"rwd_pip_0017": true,
			"rwd_pip_0018": true,
			"rwd_pip_0019": true,
			"rwd_pip_0020": true,
			"rwd_pip_0021": true,
			"rwd_pip_0022": true,
			"rwd_emissive_0023": true,
			"rwd_emissive_0024": true,
			"rwd_emissive_0026": true,
			"rwd_emissive_0028": true,
			"rwd_emissive_0029": true,
			"rwd_bracer_fume_a_daydream": true,
			"rwd_booster_fume_a_daydream": true,
			"rwd_goal_fx_0012": true,
			"rwd_goal_fx_0004": true,
			"rwd_goal_fx_0013": true,
			"rwd_goal_fx_0015": true,
			"rwd_goal_fx_0003": true,
			"rwd_decal_0017": true,
			"rwd_emissive_0016": true,
			"decal_kronos_a": true,
			"decal_one_year_a": true,
			"emote_one_a": true,
			"tint_neutral_xmas_a_default": true,
			"tint_neutral_xmas_b_default": true,
			"tint_neutral_xmas_c_default": true,
			"tint_neutral_xmas_d_default": true,
			"tint_neutral_xmas_e_default": true,
			"pattern_xmas_lights_a": true,
			"pattern_xmas_snowflakes_a": true,
			"pattern_xmas_mistletoe_a": true,
			"pattern_xmas_flourish_a": true,
			"pattern_xmas_knit_a": true,
			"pattern_xmas_knit_flowers_a": true,
			"decal_present_a": true,
			"decal_bow_a": true,
			"decal_gingerbread_a": true,
			"decal_penguin_a": true,
			"decal_snowman_a": true,
			"decal_wreath_a": true,
			"decal_snowflake_a": true,
			"decal_reindeer_a": true,
			"emote_snowman_a": true,
			"emote_fire_a": true,
			"emote_present_a": true,
			"emote_gingerbread_man_a": true,
			"tint_neutral_spooky_a_default": true,
			"tint_neutral_spooky_b_default": true,
			"tint_neutral_spooky_c_default": true,
			"tint_neutral_spooky_d_default": true,
			"tint_neutral_spooky_e_default": true,
			"pattern_spooky_stitches_a": true,
			"pattern_spooky_cobweb_a": true,
			"pattern_spooky_bandages_a": true,
			"pattern_spooky_pumpkins_a": true,
			"pattern_spooky_bats_a": true,
			"pattern_spooky_skulls_a": true,
			"decal_halloween_bat_a": true,
			"decal_halloween_cat_a": true,
			"decal_fangs_a": true,
			"decal_halloween_ghost_a": true,
			"decal_halloween_pumpkin_a": true,
			"decal_halloween_skull_a": true,
			"decal_halloween_zombie_a": true,
			"decal_halloween_scythe_a": true,
			"emote_pumpkin_face_a": true,
			"emote_scared_a": true,
			"emote_rip_a": true,
			"emote_bats_a": true,
			"rwd_banner_lone_echo_2_a": true,
			"rwd_tag_lone_echo_2_a": true,
			"rwd_decal_lone_echo_2_a": true,
			"rwd_medal_lone_echo_2_a": true,
			"rwd_booster_herosuit_a": true,
			"rwd_chassis_herosuit_a": true,
			"tint_neutral_summer_a_default": true,
			"pattern_summer_hawaiian_a": true,
			"decal_summer_pirate_a": true,
			"decal_summer_shark_a": true,
			"decal_summer_whale_a": true,
			"decal_summer_submarine_a": true,
			"decal_halloween_cauldron_a": true,
			"decal_anniversary_cupcake_a": true,
			"decal_combat_anniversary_a": true,
			"decal_santa_cubesat_a": true,
			"decal_quest_launch_a": true,
			"emote_dancing_octopus_a": true,
			"emote_spider_a": true,
			"emote_combat_anniversary_a": true,
			"emote_snow_globe_a": true,
			"emote_ding": true,
			"rwd_medal_s1_quest_launch": true,
			"rwd_booster_anubis_a_horus": true,
			"rwd_bracer_anubis_a_horus": true,
			"rwd_booster_shark_a_tropical": true,
			"rwd_bracer_shark_a_tropical": true,
			"rwd_chassis_anubis_a_horus": true,
			"rwd_chassis_shark_a_tropical": true,
			"rwd_chassis_spartan_a_hero": true,
			"rwd_bracer_spartan_a_hero": true,
			"rwd_booster_spartan_a_hero": true,
			"rwd_bracer_snacktime_a": true,
			"rwd_booster_snacktime_a": true,
			"rwd_bracer_heartbreak_a": true,
			"rwd_booster_heartbreak_a": true,
			"rwd_bracer_vroom_a": true,
			"rwd_booster_vroom_a": true,
			"rwd_chassis_ninja_a": true,
			"rwd_bracer_ninja_a": true,
			"rwd_booster_ninja_a": true,
			"rwd_pattern_0021": true,
			"rwd_emissive_0022": true,
			"rwd_pip_0025": true,
			"rwd_emote_0022": true,
			"rwd_tint_0029": true,
			"rwd_goal_fx_0007": true,
			"rwd_decal_0022": true,
			"rwd_banner_0028": true,
			"rwd_emote_0023": true,
			"rwd_emissive_0032": true,
			"rwd_pip_0024": true,
			"rwd_tag_0039": true,
			"tint_neutral_m_default": true,
			"decal_oculus_a": true,
			"emote_deal_glasses_a": true,
			"rwd_chassis_body_s10_a": true,
			"rwd_booster_s10": true,
			"rwd_title_title_b": true,
			"rwd_medal_s1_combat_bronze": true,
			"rwd_medal_s1_combat_silver": true,
			"rwd_medal_s1_combat_gold": true,
			"rwd_chassis_sporty_a": true,
			"rwd_banner_0004": true,
			"rwd_tag_0004": true,
			"rwd_tint_0003": true,
			"rwd_title_0003": true,
			"rwd_currency_starter_pack_01": true,
			"rwd_pattern_0003": true,
			"rwd_banner_0007": true,
			"rwd_tag_0007": true,
			"rwd_bracer_reptile_a": true,
			"rwd_emote_0003": true,
			"rwd_tint_0006": true,
			"rwd_booster_reptile_a": true,
			"rwd_decal_0003": true,
			"rwd_banner_0005": true,
			"rwd_bracer_retro_a": true,
			"rwd_emote_0004": true,
			"rwd_tag_0005": true,
			"rwd_booster_retro_a": true,
			"rwd_tint_0004": true,
			"rwd_pattern_0004": true,
			"rwd_bracer_avian_a": true,
			"rwd_tag_0006": true,
			"rwd_decal_0004": true,
			"rwd_booster_avian_a": true,
			"rwd_banner_0006": true,
			"rwd_tint_0005": true,
			"rwd_chassis_frost_a": true,
			"rwd_bracer_frost_a": true,
			"rwd_booster_frost_a": true,
			"rwd_booster_speedform_a": true,
			"rwd_tag_0016": true,
			"rwd_emote_0008": true,
			"rwd_bracer_speedform_a": true,
			"rwd_tint_0010": true,
			"rwd_booster_mech_a": true,
			"rwd_emote_0009": true,
			"rwd_banner_0012": true,
			"rwd_bracer_mech_a": true,
			"rwd_tint_0011": true,
			"rwd_decal_0008": true,
			"rwd_booster_organic_a": true,
			"rwd_banner_0013": true,
			"rwd_tag_0017": true,
			"rwd_bracer_organic_a": true,
			"rwd_tint_0012": true,
			"rwd_decal_0009": true,
			"rwd_chassis_exo_a": true,
			"rwd_bracer_exo_a": true,
			"rwd_booster_exo_a": true,
			"rwd_pattern_0007": true,
			"rwd_banner_0018": true,
			"rwd_tag_0022": true,
			"rwd_chassis_wolf_a": true,
			"rwd_bracer_wolf_a": true,
			"rwd_booster_wolf_a": true,
			"rwd_pattern_0011": true,
			"rwd_bracer_fragment_a": true,
			"rwd_tint_0016": true,
			"rwd_emote_0013": true,
			"rwd_booster_fragment_a": true,
			"rwd_tag_0023": true,
			"rwd_decal_0013": true,
			"rwd_bracer_baroque_a": true,
			"rwd_emote_0014": true,
			"rwd_banner_0019": true,
			"rwd_booster_baroque_a": true,
			"rwd_tint_0017": true,
			"rwd_pattern_0012": true,
			"rwd_booster_lava_a": true,
			"rwd_banner_0020": true,
			"rwd_tint_0018": true,
			"rwd_bracer_lava_a": true,
			"rwd_tag_0024": true,
			"rwd_decal_0014": true,
			"rwd_tag_0026": true,
			"rwd_tag_0025": true,
			"rwd_emote_0015": true,
			"rwd_tag_s1_r_secondary": true,
			"rwd_tag_0033": true,
			"rwd_pattern_0020": true,
			"rwd_emote_0021": true,
			"rwd_tint_0028": true,
			"rwd_bracer_halloween_a": true,
			"rwd_booster_halloween_a": true,
			"rwd_bracer_flamingo_a": true,
			"rwd_booster_flamingo_a": true,
			"rwd_bracer_paladin_a": true,
			"rwd_booster_paladin_a": true,
			"rwd_chassis_overgrown_a": true,
			"rwd_booster_overgrown_a": true,
			"rwd_bracer_overgrown_a": true,
			"rwd_tint_s1_b_default": true,
			"rwd_emote_0024": true,
			"rwd_goal_fx_0014": true,
			"rwd_goal_fx_0001": true,
			"rwd_pip_0023": true,
			"rwd_emissive_0030": true,
			"rwd_chassis_samurai_a_oni": true,
			"rwd_bracer_samurai_a_oni": true,
			"rwd_booster_samurai_a_oni": true,
			"rwd_pip_0004": true,
			"rwd_emissive_0021": true,
			"rwd_emissive_0039": true,
			"rwd_banner_0026": true,
			"rwd_tag_0038": true,
			"rwd_banner_0027": true,
			"rwd_pip_0003": true,
			"rwd_bracer_trex_a_skelerex": true,
			"rwd_booster_trex_a_skelerex": true,
			"rwd_chassis_trex_a_skelerex": true,
			"rwd_decal_0021": true,
			"rwd_goal_fx_0006": true,
			"rwd_banner_0029": true,
			"rwd_tag_0034": true,
			"rwd_pip_0002": true,
			"rwd_pip_0016": true,
			"rwd_pip_0012": true,
			"rwd_emissive_0015": true,
			"rwd_emissive_0018": true,
			"rwd_emissive_0019": true,
			"rwd_emissive_0020": true,
			"rwd_banner_s2_lines": true,
			"rwd_tint_s2_a_default": true,
			"rwd_title_0016": true,
			"rwd_title_0017": true,
			"rwd_title_0018": true,
			"rwd_title_0019": true
		},
		"combat": {
			"rwd_booster_s10": true,
			"rwd_chassis_body_s10_a": true,
			"rwd_medal_s1_combat_gold": true,
			"rwd_title_title_b": true,
			"rwd_medal_s1_combat_bronze": true,
			"rwd_medal_s1_combat_silver": true
		}
	},
	"loadout": {
		"instances": {
			"unified": {
				"slots": {
					"decal": "decal_dinosaur_a",
					"decal_body": "decal_dinosaur_a",
					"emote": "emote_default",
					"secondemote": "emote_default",
					"tint": "tint_blue_e_default",
					"tint_body": "tint_blue_e_default",
					"tint_alignment_a": "tint_blue_a_default",
					"tint_alignment_b": "tint_orange_a_default",
					"pattern": "pattern_honeycomb_triple_a",
					"pattern_body": "pattern_honeycomb_triple_a",
					"pip": "rwd_decalback_default",
					"chassis": "rwd_chassis_s8b_a",
					"bracer": "rwd_bracer_vintage_a",
					"booster": "rwd_booster_vintage_a",
					"title": "rwd_title_title_c",
					"tag": "rwd_tag_s1_k_secondary",
					"banner": "rwd_banner_s1_bold_stripe",
					"medal": "rwd_medal_s1_combat_silver",
					"goal_fx": "rwd_goal_fx_0010",
					"emissive": "rwd_emissive_0023"
				}
			}
		},
		"number": 1
	}
}`
