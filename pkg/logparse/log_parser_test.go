package logparse

import (
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path"
	"strings"
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	require.Equal(t, time.Date(2021, 2, 21, 6, 22, 23, 0, time.UTC),
		parseDateTime("02/21/2021", "06:22:23"))
}

func TestParseAltBasic(t *testing.T) {
	p := golib.FindFile(path.Join("test_data", "log_1.log"), "gbans")
	f, e := ioutil.ReadFile(p)
	if e != nil {
		t.Fatalf("Failed to open test file: %s", p)
	}
	results := make(map[int]Results)
	for i, line := range strings.Split(string(f), "\n") {
		v := Parse(line)
		results[i] = v
	}
	expected := map[MsgType]int{
		SayTeam: 6,
		Say:     18,
	}
	for mt, expectedCount := range expected {
		found := 0
		for _, result := range results {
			if result.MsgType == mt {
				found++
			}
		}
		require.Equal(t, expectedCount, found, "Invalid count for type: %v %d/%d", mt, found, expectedCount)
	}
}

func TestParseAltSupMedStats(t *testing.T) {
	p := golib.FindFile(path.Join("test_data", "log_sup_med_1.log"), "gbans")
	f, e := ioutil.ReadFile(p)
	if e != nil {
		t.Fatalf("Failed to open test file: %s", p)
	}
	results := make(map[int]Results)
	for i, line := range strings.Split(string(f), "\n") {
		v := Parse(line)
		results[i] = v
	}
	expected := map[MsgType]int{
		UnhandledMsg: 0,
		UnknownMsg:   0,

		Say:                 31,
		SayTeam:             10,
		Killed:              292,
		KillAssist:          132,
		Suicide:             4,
		ShotFired:           2636,
		ShotHit:             996,
		Damage:              2235,
		Domination:          6,
		Revenge:             3,
		Pickup:              441,
		EmptyUber:           0,
		MedicDeath:          19,
		MedicDeathEx:        0,
		LostUberAdv:         0,
		ChargeReady:         29,
		ChargeDeployed:      29,
		ChargeEnded:         0,
		Healed:              796,
		Extinguished:        9,
		BuiltObject:         61,
		CarryObject:         5,
		KilledObject:        46,
		DetonatedObject:     1,
		DropObject:          5,
		FirstHealAfterSpawn: 0,
		CaptureBlocked:      11,
		PointCaptured:       10,
		JoinedTeam:          32,
		ChangeClass:         58,
		SpawnedAs:           368,
		MilkAttack:          12,

		WRoundOvertime:  2,
		WRoundStart:     2,
		WRoundWin:       2,
		WRoundLen:       2,
		WTeamScore:      4,
		WTeamFinalScore: 2,
		WGameOver:       1,
		WPaused:         0,
		WResumed:        0,

		LogStart:     1,
		LogStop:      1,
		CVAR:         6,
		RCON:         34,
		Connected:    19,
		Disconnected: 13,
		Validated:    19,
		Entered:      34,
	}
	var unknown []Results
	var unhandled []Results
	for mt, expectedCount := range expected {
		found := 0
		for _, result := range results {
			if result.MsgType == mt {
				found++
				if result.MsgType == UnhandledMsg {
					unhandled = append(unhandled, result)
				} else if result.MsgType == UnknownMsg {
					unknown = append(unknown, result)
				}
			}
		}
		if mt != UnhandledMsg && mt != UnknownMsg {
			require.Equal(t, expectedCount, found, "Invalid count for type: %v %d/%d", mt, found, expectedCount)
		}
	}
	require.Equal(t, 1, len(unknown))
	require.Equal(t, 3, len(unhandled))
}

func TestKvToMap(t *testing.T) {
	const s1 = `file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (attacker_position "-1189 2513 -423") (version "6300758"`
	m := map[string]string{}
	kvToMap(s1, m)
	require.EqualValues(t, map[string]string{
		"file":              "logs/L0221034.log",
		"game":              "/home/tf2server/serverfiles/tf",
		"version":           "6300758",
		"attacker_position": "-1189 2513 -423",
	}, m)
}

func TestParse(t *testing.T) {
	var pa = func(s string, msgType MsgType) map[string]string {
		v := Parse(s)
		require.Equal(t, msgType, v.MsgType, "Invalid msg type (%d -> %d) %s", msgType, v.MsgType, s)
		return v.Values
	}
	var value1 UnhandledMsgEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:22:23: asdf`, UnhandledMsg), &value1))
	require.Equal(t, UnhandledMsgEvt{}, value1)

	var value2 LogStartEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:22:23: Log file started (file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (version "6300758")`,
		LogStart), &value2))
	require.Equal(t, LogStartEvt{
		File: "logs/L0221034.log", Game: "/home/tf2server/serverfiles/tf", Version: "6300758"}, value2)

	var value3 CVAREvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:22:23: server_cvar: "sm_nextmap" "pl_frontier_final"`, CVAR), &value3))
	require.Equal(t, CVAREvt{CVAR: "sm_nextmap", Value: "pl_frontier_final"}, value3)

	var value4 RCONEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:22:24: rcon from "23.239.22.163:42004": command "status"`, RCON), &value4))
	require.EqualValues(t, RCONEvt{Cmd: "status"}, value4)

	var value5 EnteredEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:22:31: "Hacksaw<12><[U:1:68745073]><>" entered the game`, Entered), &value5))
	require.EqualValues(t, EmptyEvt{}, value5)

	var value6 JoinedTeamEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:22:35: "Hacksaw<12><[U:1:68745073]><Unassigned>" joined team "Red"`, JoinedTeam), &value6))
	require.EqualValues(t, JoinedTeamEvt{
		Team: RED,
		SourcePlayer: SourcePlayer{
			Name: "Hacksaw", PID: 12, SID: steamid.SID3ToSID64("[U:1:68745073]"), Team: 1,
		}}, value6)

	var value7 ChangeClassEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:22:36: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "scout"`, ChangeClass), &value7))
	require.EqualValues(t, ChangeClassEvt{Class: Scout}, value7)

	var value8 SuicideEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:23:04: "Dzefersons14<8><[U:1:1080653073]><Blue>" committed suicide with "world" (attacker_position "-1189 2513 -423")`, Suicide), &value8))
	require.EqualValues(t, SuicideEvt{
		SourcePlayer:     SourcePlayer{Name: "Dzefersons14", PID: 8, SID: 0x110000140697511, Team: 2},
		AttackerPosition: Pos{X: -1189, Y: 2513, Z: -423}}, value8)

	var value9 WRoundStartEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:23:11: World triggered "Round_Start"`, WRoundStart), &value9))
	require.EqualValues(t, EmptyEvt{}, value9)

	var value10 MedicDeathEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:23:44: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "0")`, MedicDeath), &value10))
	require.EqualValues(t, MedicDeathEvt{
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8,
			SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
		},
		Healing: 135,
		Uber:    0}, value10)

	var value11 KilledEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:23:44: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`, Killed), &value11))
	require.EqualValues(t, KilledEvt{
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8,
			SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
		},
		APos:       Pos{X: 217, Y: -54, Z: -302},
		VPos:       Pos{X: 203, Y: -2, Z: -319},
		CustomKill: "backstab"}, value11)

	var value12 KillAssistEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:23:44: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "-476 154 -254") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`, KillAssist), &value12))
	require.EqualValues(t, KillAssistEvt{
		SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: 0x11000010418f771, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8,
			SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
		},
		AssisterPosition: Pos{X: -476, Y: 154, Z: -254},
		AttackerPosition: Pos{X: 217, Y: -54, Z: -302},
		VictimPosition:   Pos{X: 203, Y: -2, Z: -319}}, value12)

	var value13 PointCapturedEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:24:14: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313")`, PointCaptured), &value13))
	require.EqualValues(t, PointCapturedEvt{
		Team: RED, CP: 0, CPName: "#koth_viaduct_cap", NumCappers: 1,
		Body: `(player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313")`}, value13)

	var value14 ConnectedEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:24:22: "amogus gaming<13><[U:1:1089803558]><>" connected, address "139.47.95.130:47949"`, Connected), &value14))
	require.EqualValues(t, ConnectedEvt{
		SourcePlayer: SourcePlayer{Name: "amogus gaming", PID: 13, SID: 0x110000140f51526, Team: 0},
		Address:      "139.47.95.130:47949"}, value14)

	var value15 EmptyEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:24:23: "amogus gaming<13><[U:1:1089803558]><>" STEAM USERID Validated`, Validated), &value15))
	require.EqualValues(t, EmptyEvt{}, value15)

	var value16 KilledObjectEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:26:33: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "2 -579 -255")`, KilledObject), &value16))
	require.EqualValues(t, KilledObjectEvt{
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{
			Name2: "idk",
			PID2:  9,
			SID2:  steamid.SID3ToSID64("[U:1:1170132017]"),
			Team2: BLU,
		},
		Object: "OBJ_SENTRYGUN",
		Weapon: "obj_attachment_sapper",
		APos:   Pos{2, -579, -255}}, value16)

	var value17 CarryObjectEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:30:45: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1074 -2279 -423")`, CarryObject), &value17))
	require.EqualValues(t, CarryObjectEvt{
		SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: 2},
		Object:       "OBJ_SENTRYGUN", AttackerPosition: Pos{1074, -2279, -423}}, value17)

	var value18 DropObjectEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:32:00: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "339 -419 -255")`, DropObject), &value18))
	require.EqualValues(t, DropObjectEvt{
		SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: 2},
		Object:       "OBJ_SENTRYGUN", AttackerPosition: Pos{339, -419, -255}}, value18)

	var value19 BuiltObjectEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:32:30: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "880 -152 -255")`, BuiltObject), &value19))
	require.EqualValues(t, BuiltObjectEvt{
		SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: 2}, Object: "OBJ_SENTRYGUN",
		AttackerPosition: Pos{880, -152, -255}}, value19)

	var value20 WRoundWinEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:29:49: World triggered "Round_Win" (winner "Red")`, WRoundWin), &value20))
	require.EqualValues(t, WRoundWinEvt{Winner: RED}, value20)

	var value21 WRoundLenEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:29:49: World triggered "Round_Length" (seconds "398.10")`, WRoundLen), &value21))
	require.EqualValues(t, WRoundLenEvt{Length: 398.10}, value21)

	var value22 WTeamScoreEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:29:49: Team "Red" current score "1" with "2" players`, WTeamScore), &value22))
	require.EqualValues(t, WTeamScoreEvt{Team: RED, Score: 1, Players: 2}, value22)

	var value23 SayEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:29:57: "Hacksaw<12><[U:1:68745073]><Red>" say "gg"`, Say), &value23))
	require.EqualValues(t, SayEvt{SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: 0x11000010418f771, Team: 1}, Msg: "gg"}, value23)

	var value24 SayTeamEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:29:59: "Desmos Calculator<10><[U:1:1132396177]><Red>" say_team "gg"`, SayTeam), &value24))
	require.EqualValues(t, SayTeamEvt{SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1}, Msg: "gg"}, value24)

	var value25 DominationEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:33:41: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "domination" against "Dzefersons14<8><[U:1:1080653073]><Blue>"`, Domination), &value25))
	require.EqualValues(t, DominationEvt{
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{
			Name2: "Dzefersons14", PID2: 8, SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
		}}, value25)

	var value26 DisconnectedEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:33:43: "Cybermorphic<15><[U:1:901503117]><Unassigned>" disconnected (reason "Disconnect by user.")`, Disconnected), &value26))
	require.EqualValues(t, DisconnectedEvt{SourcePlayer: SourcePlayer{Name: "Cybermorphic", PID: 15, SID: 0x110000135bbd88d, Team: 0},
		Reason: "Disconnect by user."}, value26)

	var value27 RevengeEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:35:37: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "revenge" against "Desmos Calculator<10><[U:1:1132396177]><Red>"`, Revenge), &value27))
	require.EqualValues(t, RevengeEvt{
		SourcePlayer: SourcePlayer{Name: "Dzefersons14", PID: 8, SID: 0x110000140697511, Team: 2},
		TargetPlayer: TargetPlayer{
			Name2: "Desmos Calculator", PID2: 10,
			SID2: steamid.SID3ToSID64("[U:1:1132396177]"), Team2: RED}}, value27)

	var value28 WRoundOvertimeEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:37:20: World triggered "Round_Overtime"`, WRoundOvertime), &value28))
	require.EqualValues(t, WRoundOvertimeEvt{}, value28)

	var value29 CaptureBlockedEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:40:19: "potato<16><[U:1:385661040]><Red>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-163 324 -272")`, CaptureBlocked), &value29))
	require.EqualValues(t, CaptureBlockedEvt{
		SourcePlayer: SourcePlayer{Name: "potato", PID: 16, SID: 0x110000116fcb870, Team: 1},
		CP:           0,
		CPName:       "#koth_viaduct_cap",
		Pos:          Pos{-163, 324, -272}}, value29)

	var value30 WGameOverEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:42:13: World triggered "Game_Over" reason "Reached Win Limit"`, WGameOver), &value30))
	require.EqualValues(t, WGameOverEvt{Reason: "Reached Win Limit"}, value30)

	var value31 WTeamFinalScoreEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:42:13: Team "Red" final score "2" with "3" players`, WTeamFinalScore), &value31))
	require.EqualValues(t, WTeamFinalScoreEvt{Score: 2, Players: 3}, value31)

	var value32 UnhandledMsgEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:42:13: Team "RED" triggered "Intermission_Win_Limit"`, UnhandledMsg), &value32))
	require.EqualValues(t, UnhandledMsgEvt{}, value32)

	var value33 UnhandledMsgEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:42:33: [META] Loaded 0 plugins (1 already loaded)`, UnhandledMsg), &value33))
	require.EqualValues(t, UnhandledMsgEvt{}, value33)

	var value34 LogStopEvt
	require.NoError(t, Unmarshal(pa(`L 02/21/2021 - 06:42:33: Log file closed.`, LogStop), &value34))
	require.EqualValues(t, LogStopEvt{}, value34)

	var value35 WPausedEvt
	require.NoError(t, Unmarshal(pa(`L 10/27/2019 - 23:53:58: World triggered "Game_Paused"`, WPaused), &value35))
	require.EqualValues(t, WPausedEvt{}, value35)

	var value36 WResumedEvt
	require.NoError(t, Unmarshal(pa(`L 10/27/2019 - 23:53:38: World triggered "Game_Unpaused"`, WResumed), &value36))
	require.EqualValues(t, WResumedEvt{}, value36)

	var value37 FirstHealAfterSpawnEvt
	require.NoError(t, Unmarshal(pa(`L 10/25/2019 - 12:19:46: "SCOTTY T<27><[U:1:97282856]><Blue>" triggered "first_heal_after_spawn" (time "1.6")`, FirstHealAfterSpawn), &value37))
	require.EqualValues(t, FirstHealAfterSpawnEvt{SourcePlayer: SourcePlayer{Name: "SCOTTY T", PID: 27, SID: 0x110000105cc6b28, Team: 2}, HealTime: 1.6}, value37)

	var value38 ChargeReadyEvt
	require.NoError(t, Unmarshal(pa(`L 07/11/2019 - 00:11:04: "wonder<7><[U:1:34284979]><Red>" triggered "chargeready"`, ChargeReady), &value38))
	require.EqualValues(t, ChargeReadyEvt{SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: 1}}, value38)

	var value39 ChargeDeployedEvt
	require.NoError(t, Unmarshal(pa(`L 07/11/2019 - 00:11:11: "wonder<7><[U:1:34284979]><Red>" triggered "chargedeployed" (medigun "medigun")`, ChargeDeployed), &value39))
	require.EqualValues(t, ChargeDeployedEvt{SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: 1}, Medigun: Uber}, value39)

	var value40 ChargeEndedEvt
	require.NoError(t, Unmarshal(pa(`L 07/11/2019 - 00:11:18: "wonder<7><[U:1:34284979]><Red>" triggered "chargeended" (duration "7.5")`, ChargeEnded), &value40))
	require.EqualValues(t, ChargeEndedEvt{SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: 1}, Duration: 7.5}, value40)

	var value41 MedicDeathExEvt
	require.NoError(t, Unmarshal(pa(`L 07/10/2019 - 23:47:52: "wonder<7><[U:1:34284979]><Red>" triggered "medic_death_ex" (uberpct "32")`, MedicDeathEx), &value41))
	require.Equal(t, MedicDeathExEvt{UberPct: 32}, value41)

	var value42 LostUberAdvantageEvt
	require.NoError(t, Unmarshal(pa(`L 07/10/2019 - 23:47:32: "SEND HELP<16><[U:1:84528002]><Blue>" triggered "lost_uber_advantage" (time "44")`, LostUberAdv), &value42))
	require.Equal(t, LostUberAdvantageEvt{SourcePlayer: SourcePlayer{Name: "SEND HELP", PID: 16, SID: 0x11000010509cb82, Team: 2}, AdvTime: 44}, value42)

	var value43 EmptyUberEvt
	require.NoError(t, Unmarshal(pa(`L 07/10/2019 - 23:26:43: "Kwq<9><[U:1:96748980]><Blue>" triggered "empty_uber"`, EmptyUber), &value43))
	require.Equal(t, EmptyUberEvt{SourcePlayer: SourcePlayer{Name: "Kwq", PID: 9, SID: 0x110000105c445b4, Team: 2}}, value43)

	var value44 PickupEvt
	require.NoError(t, Unmarshal(pa(`L 07/10/2019 - 23:47:34: "g о а т z<13><[U:1:41435165]><Red>" picked up item "ammopack_small"`, Pickup), &value44))
	require.EqualValues(t, PickupEvt{SourcePlayer: SourcePlayer{Name: "g о а т z", PID: 13, SID: 0x11000010278401d, Team: 1}, Item: AmmoSmall}, value44)

	var value45 ShotFiredEvt
	require.NoError(t, Unmarshal(pa(`L 07/10/2019 - 23:28:02: "rad<6><[U:1:57823119]><Red>" triggered "shot_fired" (weapon "syringegun_medic")`, ShotFired), &value45))
	require.EqualValues(t, ShotFiredEvt{SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: 1}, Weapon: SyringeGun}, value45)

	var value46 ShotHitEvt
	require.NoError(t, Unmarshal(pa(`L 07/10/2019 - 23:28:02: "z/<14><[U:1:66656848]><Blue>" triggered "shot_hit" (weapon "blackbox")`, ShotHit), &value46))
	require.EqualValues(t, ShotHitEvt{SourcePlayer: SourcePlayer{Name: "z/", PID: 14, SID: 0x110000103f91a50, Team: 2}, Weapon: Blackbox}, value46)

	var value47 DamageEvt
	require.NoError(t, Unmarshal(pa(`L 07/10/2019 - 23:28:01: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "11") (weapon "syringegun_medic")`, Damage), &value47))
	require.EqualValues(t, DamageEvt{
		SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: RED},
		TargetPlayer: TargetPlayer{Name2: "z/", PID2: 14, SID2: steamid.SID3ToSID64("[U:1:66656848]"), Team2: BLU},
		Weapon:       SyringeGun, Damage: 11}, value47)

	var value48 DamageEvt
	require.NoError(t, Unmarshal(pa(`L 07/10/2019 - 23:29:54: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "88") (realdamage "32") (weapon "ubersaw") (healing "110")`, Damage), &value48))
	require.EqualValues(t, DamageEvt{
		SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: RED},
		TargetPlayer: TargetPlayer{Name2: "z/", PID2: 14, SID2: steamid.SID3ToSID64("[U:1:66656848]"), Team2: BLU},
		Damage:       88, RealDamage: 32, Weapon: Ubersaw, Healing: 110}, value48)

	var value49 MilkAttackEvt
	require.NoError(t, Unmarshal(pa(`L 05/21/2021 - 20:39:58: "Five<636><[U:1:66374745]><Blue>" triggered "milk_attack" against "drex<634><[U:1:59956152]><Red>" with "tf_weapon_jar" (attacker_position "552 -526 -195") (victim_position "524 -100 -184")`, MilkAttack), &value49))
	require.EqualValues(t, MilkAttackEvt{
		SourcePlayer: SourcePlayer{Name: "Five", PID: 636, SID: steamid.SID3ToSID64("[U:1:66374745]"), Team: BLU},
		TargetPlayer: TargetPlayer{Name2: "drex", PID2: 634, SID2: steamid.SID3ToSID64("[U:1:59956152]"), Team2: RED},
		APos:         Pos{X: 552, Y: -526, Z: -195}, VPos: Pos{X: 524, Y: -100, Z: -184},
	}, value49)
}

func TestParseKVs(t *testing.T) {
	m := map[string]string{}
	require.True(t, parseKVs(`(damage "88") (realdamage "32") (weapon "ubersaw") (attacker_position "-1189 2513 -423") (healing "110")`, m))
	require.Equal(t, map[string]string{"damage": "88", "realdamage": "32", "weapon": "ubersaw", "healing": "110", "attacker_position": "-1189 2513 -423"}, m)
}
