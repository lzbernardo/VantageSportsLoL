package baseview

const (
	BlueTeam int64 = 100
	RedTeam  int64 = 200
)

//
// Entities
//

type ActorType string

const (
	ActorHero      ActorType = "hero"
	ActorMinion    ActorType = "minion" // includes jungle minions
	ActorMonster   ActorType = "monster"
	ActorTurret    ActorType = "turret"
	ActorInhibitor ActorType = "inhibitor"
	ActorWard      ActorType = "ward"
)

//
// Buildings
//

const (
	TurretBlueTopInner   = 101
	TurretBlueTopOuter   = 102
	TurretBlueUpperNexus = 103
	TurretBlueLowerNexus = 104
	TurretBlueMidBase    = 105
	TurretBlueMidInner   = 106
	TurretBlueMidOuter   = 107
	TurretBlueTopBase    = 108
	TurretBlueBotBase    = 109
	TurretBlueBotInner   = 110
	TurretBlueBotOuter   = 111
	TurretBlueFountain   = 112

	InhibitorBlueTop = 130
	InhibitorBlueMid = 131
	InhibitorBlueBot = 132

	TurretRedTopInner   = 201
	TurretRedTopOuter   = 202
	TurretRedLowerNexus = 203
	TurretRedUpperNexus = 204
	TurretRedMidBase    = 205
	TurretRedMidInner   = 206
	TurretRedMidOuter   = 207
	TurretRedTopBase    = 208
	TurretRedBotBase    = 209
	TurretRedBotInner   = 210
	TurretRedBotOuter   = 211
	TurretRedFountain   = 212

	InhibitorRedTop = 230
	InhibitorRedMid = 231
	InhibitorRedBot = 232
)

// Approximate Turret Locations (can't find an actual source for this)
var TurretPositions map[int64]Position = map[int64]Position{
	101: {X: 1364.939, Y: 6837.4, Z: 52.83815},
	102: {X: 994, Y: 10434, Z: 52.8381},
	103: {X: 1784.683, Y: 2325.73, Z: 95.74805},
	104: {X: 2219.66, Y: 1777.356, Z: 95.74808},
	105: {X: 3649.316, Y: 3670.658, Z: 95.74804},
	106: {X: 5038.548, Y: 4957.532, Z: 50.23169},
	107: {X: 5902, Y: 6336, Z: 51.79154},
	108: {X: 1198, Y: 4278, Z: 95.74805},
	109: {X: 4274, Y: 1258, Z: 95.74808},
	110: {X: 6844.213, Y: 1544.085, Z: 49.4502},
	111: {X: 10420.48, Y: 1048, Z: 51.35432},
	201: {X: 7940, Y: 13376, Z: 52.8381},
	202: {X: 4320, Y: 13854, Z: 52.8381},
	203: {X: 13078, Y: 12584, Z: 91.42981},
	204: {X: 12564, Y: 13084, Z: 91.42981},
	205: {X: 11110.61, Y: 11183.44, Z: 91.42981},
	206: {X: 9012, Y: 10066, Z: 52.3063},
	207: {X: 8900, Y: 8576, Z: 54.04472},
	208: {X: 10472.67, Y: 13598.79, Z: 91.79472},
	209: {X: 13604, Y: 10578, Z: 91.42978},
	210: {X: 13346, Y: 8340, Z: 52.3063},
	211: {X: 13854.49, Y: 4426.352, Z: 52.80367},
}

//
// Monsters
//

const (
	MonsterRiftHerald       int64 = 300 // SRU_RiftHerald...
	MonsterBaron            int64 = 301 // SRU_Baron...
	MonsterDragonElemental  int64 = 302 // SRU_Dragon{Air,Earth,Fire,Water}...
	MonsterDragonElder      int64 = 303 // SRU_DragonElder...
	MonsterBlueSentinel     int64 = 304 // SRU_Blue...
	MonsterBlueSentinelMini int64 = 305 // SRU_BlueMini...
	MonsterGromp            int64 = 306 // SRU_Gromp...
	MonsterKrug             int64 = 307 // SRU_Krug...
	MonsterKrugMini         int64 = 308 // SRU_KrugMini...
	MonsterMurkwolf         int64 = 309 // SRU_Murk...
	MonsterMurkwolfMini     int64 = 310 // SRU_MurkMini...
	MonsterRazor            int64 = 311 // SRU_Razor...
	MonsterRazorMini        int64 = 312 // SRU_RazorMini...
	MonsterRedBramb         int64 = 313 // SRU_Red...
	MonsterRedBrambMini     int64 = 314 // SRU_RedMini...
	MonsterCrab             int64 = 315 // Sru_Crab...
)

func IsEpicMonster(id int64) bool {
	switch id {
	case MonsterRiftHerald, MonsterBaron, MonsterDragonElemental, MonsterDragonElder:
		return true
	}
	return false
}

func IsJungleMinion(id int64) bool {
	switch id {
	case MonsterBlueSentinel, MonsterBlueSentinelMini, MonsterGromp,
		MonsterKrug, MonsterKrugMini, MonsterMurkwolf, MonsterMurkwolfMini,
		MonsterRazor, MonsterRazorMini, MonsterRedBramb, MonsterRedBrambMini,
		MonsterCrab:
		return true
	}
	return false
}

//
// Roles
//

type RolePosition string

const (
	RoleTop     RolePosition = "top"
	RoleMid     RolePosition = "mid"
	RoleAdc     RolePosition = "adc"
	RoleJungle  RolePosition = "jng"
	RoleSupport RolePosition = "sup"
)
