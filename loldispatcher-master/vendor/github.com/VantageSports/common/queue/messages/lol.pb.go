// Code generated by protoc-gen-go.
// source: lol.proto
// DO NOT EDIT!

/*
Package messages is a generated protocol buffer package.

It is generated from these files:
	lol.proto
	nba.proto
	common.proto

It has these top-level messages:
	LolAdvancedStatsIngest
	LolBasicStatsIngest
	LolEloDataProcess
	LolMatchDownload
	LolReplayDataExtract
	LolSummonerCrawl
	LolVideo
	LolGoalsUpdate
	GameFileMigration
	MigrateChances
	OCRTextExtraction
	OCRDimensions
	ValidateChances
	VideoJoin
	VideoSplit
	VideoSegment
	Email
	FilesExistCondition
	TranscodeVideo
*/
package messages

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// LolAdvancedStatsIngest describes a single match and summoner to ingest advanced stats for.
type LolAdvancedStatsIngest struct {
	BaseviewPath string `protobuf:"bytes,1,opt,name=baseview_path,json=baseviewPath" json:"baseview_path,omitempty"`
	BaseviewType string `protobuf:"bytes,2,opt,name=baseview_type,json=baseviewType" json:"baseview_type,omitempty"`
	MatchId      int64  `protobuf:"varint,3,opt,name=match_id,json=matchId" json:"match_id,omitempty"`
	PlatformId   string `protobuf:"bytes,4,opt,name=platform_id,json=platformId" json:"platform_id,omitempty"`
	SummonerId   int64  `protobuf:"varint,5,opt,name=summoner_id,json=summonerId" json:"summoner_id,omitempty"`
	// iff true, advanced stats will be generated whether they already exist or
	// not.
	Override bool `protobuf:"varint,6,opt,name=override" json:"override,omitempty"`
}

func (m *LolAdvancedStatsIngest) Reset()                    { *m = LolAdvancedStatsIngest{} }
func (m *LolAdvancedStatsIngest) String() string            { return proto.CompactTextString(m) }
func (*LolAdvancedStatsIngest) ProtoMessage()               {}
func (*LolAdvancedStatsIngest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *LolAdvancedStatsIngest) GetBaseviewPath() string {
	if m != nil {
		return m.BaseviewPath
	}
	return ""
}

func (m *LolAdvancedStatsIngest) GetBaseviewType() string {
	if m != nil {
		return m.BaseviewType
	}
	return ""
}

func (m *LolAdvancedStatsIngest) GetMatchId() int64 {
	if m != nil {
		return m.MatchId
	}
	return 0
}

func (m *LolAdvancedStatsIngest) GetPlatformId() string {
	if m != nil {
		return m.PlatformId
	}
	return ""
}

func (m *LolAdvancedStatsIngest) GetSummonerId() int64 {
	if m != nil {
		return m.SummonerId
	}
	return 0
}

func (m *LolAdvancedStatsIngest) GetOverride() bool {
	if m != nil {
		return m.Override
	}
	return false
}

// LolBasicStatsIngest describes a single match and summoner to ingest basic stats for.
type LolBasicStatsIngest struct {
	MatchDetailsPath string `protobuf:"bytes,1,opt,name=match_details_path,json=matchDetailsPath" json:"match_details_path,omitempty"`
	MatchId          int64  `protobuf:"varint,2,opt,name=match_id,json=matchId" json:"match_id,omitempty"`
	PlatformId       string `protobuf:"bytes,3,opt,name=platform_id,json=platformId" json:"platform_id,omitempty"`
	SummonerId       int64  `protobuf:"varint,4,opt,name=summoner_id,json=summonerId" json:"summoner_id,omitempty"`
	// iff true, basic stats will be generated whether they already exist or
	// not.
	Override bool `protobuf:"varint,5,opt,name=override" json:"override,omitempty"`
}

func (m *LolBasicStatsIngest) Reset()                    { *m = LolBasicStatsIngest{} }
func (m *LolBasicStatsIngest) String() string            { return proto.CompactTextString(m) }
func (*LolBasicStatsIngest) ProtoMessage()               {}
func (*LolBasicStatsIngest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *LolBasicStatsIngest) GetMatchDetailsPath() string {
	if m != nil {
		return m.MatchDetailsPath
	}
	return ""
}

func (m *LolBasicStatsIngest) GetMatchId() int64 {
	if m != nil {
		return m.MatchId
	}
	return 0
}

func (m *LolBasicStatsIngest) GetPlatformId() string {
	if m != nil {
		return m.PlatformId
	}
	return ""
}

func (m *LolBasicStatsIngest) GetSummonerId() int64 {
	if m != nil {
		return m.SummonerId
	}
	return 0
}

func (m *LolBasicStatsIngest) GetOverride() bool {
	if m != nil {
		return m.Override
	}
	return false
}

// LolEloDataProcess describes a single match to extract a baseview file from elo data.
type LolEloDataProcess struct {
	EloDataPath      string  `protobuf:"bytes,1,opt,name=elo_data_path,json=eloDataPath" json:"elo_data_path,omitempty"`
	MatchDetailsPath string  `protobuf:"bytes,2,opt,name=match_details_path,json=matchDetailsPath" json:"match_details_path,omitempty"`
	MatchId          int64   `protobuf:"varint,3,opt,name=match_id,json=matchId" json:"match_id,omitempty"`
	PlatformId       string  `protobuf:"bytes,4,opt,name=platform_id,json=platformId" json:"platform_id,omitempty"`
	SummonerIds      []int64 `protobuf:"varint,5,rep,packed,name=summoner_ids,json=summonerIds" json:"summoner_ids,omitempty"`
	// iff true, baseview will be generated whether it already exists or not.
	Override bool `protobuf:"varint,6,opt,name=override" json:"override,omitempty"`
}

func (m *LolEloDataProcess) Reset()                    { *m = LolEloDataProcess{} }
func (m *LolEloDataProcess) String() string            { return proto.CompactTextString(m) }
func (*LolEloDataProcess) ProtoMessage()               {}
func (*LolEloDataProcess) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *LolEloDataProcess) GetEloDataPath() string {
	if m != nil {
		return m.EloDataPath
	}
	return ""
}

func (m *LolEloDataProcess) GetMatchDetailsPath() string {
	if m != nil {
		return m.MatchDetailsPath
	}
	return ""
}

func (m *LolEloDataProcess) GetMatchId() int64 {
	if m != nil {
		return m.MatchId
	}
	return 0
}

func (m *LolEloDataProcess) GetPlatformId() string {
	if m != nil {
		return m.PlatformId
	}
	return ""
}

func (m *LolEloDataProcess) GetSummonerIds() []int64 {
	if m != nil {
		return m.SummonerIds
	}
	return nil
}

func (m *LolEloDataProcess) GetOverride() bool {
	if m != nil {
		return m.Override
	}
	return false
}

// LolMatchDownload describes matches to download and is used by the dispatcher runner to determine
// what to do with downloaded matches.
type LolMatchDownload struct {
	Key                 string  `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	MatchId             int64   `protobuf:"varint,2,opt,name=match_id,json=matchId" json:"match_id,omitempty"`
	PlatformId          string  `protobuf:"bytes,3,opt,name=platform_id,json=platformId" json:"platform_id,omitempty"`
	ObservedSummonerIds []int64 `protobuf:"varint,4,rep,packed,name=observed_summoner_ids,json=observedSummonerIds" json:"observed_summoner_ids,omitempty"`
	ReplayServer        string  `protobuf:"bytes,5,opt,name=replay_server,json=replayServer" json:"replay_server,omitempty"`
}

func (m *LolMatchDownload) Reset()                    { *m = LolMatchDownload{} }
func (m *LolMatchDownload) String() string            { return proto.CompactTextString(m) }
func (*LolMatchDownload) ProtoMessage()               {}
func (*LolMatchDownload) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *LolMatchDownload) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *LolMatchDownload) GetMatchId() int64 {
	if m != nil {
		return m.MatchId
	}
	return 0
}

func (m *LolMatchDownload) GetPlatformId() string {
	if m != nil {
		return m.PlatformId
	}
	return ""
}

func (m *LolMatchDownload) GetObservedSummonerIds() []int64 {
	if m != nil {
		return m.ObservedSummonerIds
	}
	return nil
}

func (m *LolMatchDownload) GetReplayServer() string {
	if m != nil {
		return m.ReplayServer
	}
	return ""
}

// A request to translate a lol replay into a stats programatically.
type LolReplayDataExtract struct {
	GameLengthSeconds      int64   `protobuf:"varint,1,opt,name=game_length_seconds,json=gameLengthSeconds" json:"game_length_seconds,omitempty"`
	IncreasedPlaybackSpeed bool    `protobuf:"varint,2,opt,name=increased_playback_speed,json=increasedPlaybackSpeed" json:"increased_playback_speed,omitempty"`
	Key                    string  `protobuf:"bytes,3,opt,name=key" json:"key,omitempty"`
	MatchId                int64   `protobuf:"varint,4,opt,name=match_id,json=matchId" json:"match_id,omitempty"`
	MatchDetailsPath       string  `protobuf:"bytes,5,opt,name=match_details_path,json=matchDetailsPath" json:"match_details_path,omitempty"`
	MatchVersion           string  `protobuf:"bytes,6,opt,name=match_version,json=matchVersion" json:"match_version,omitempty"`
	PlatformId             string  `protobuf:"bytes,7,opt,name=platform_id,json=platformId" json:"platform_id,omitempty"`
	SpectatorServer        string  `protobuf:"bytes,8,opt,name=spectator_server,json=spectatorServer" json:"spectator_server,omitempty"`
	SummonerIds            []int64 `protobuf:"varint,9,rep,packed,name=summoner_ids,json=summonerIds" json:"summoner_ids,omitempty"`
	// iff true, elogen will happen whether text file exists or not.
	Override bool `protobuf:"varint,10,opt,name=override" json:"override,omitempty"`
	// sometimes we want to retry data generation after failures we believe are
	// transient, and this number is incremented each time so we can stop
	// retrying once we hit a certain threshold.
	RetryNum int64 `protobuf:"varint,11,opt,name=retry_num,json=retryNum" json:"retry_num,omitempty"`
}

func (m *LolReplayDataExtract) Reset()                    { *m = LolReplayDataExtract{} }
func (m *LolReplayDataExtract) String() string            { return proto.CompactTextString(m) }
func (*LolReplayDataExtract) ProtoMessage()               {}
func (*LolReplayDataExtract) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *LolReplayDataExtract) GetGameLengthSeconds() int64 {
	if m != nil {
		return m.GameLengthSeconds
	}
	return 0
}

func (m *LolReplayDataExtract) GetIncreasedPlaybackSpeed() bool {
	if m != nil {
		return m.IncreasedPlaybackSpeed
	}
	return false
}

func (m *LolReplayDataExtract) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *LolReplayDataExtract) GetMatchId() int64 {
	if m != nil {
		return m.MatchId
	}
	return 0
}

func (m *LolReplayDataExtract) GetMatchDetailsPath() string {
	if m != nil {
		return m.MatchDetailsPath
	}
	return ""
}

func (m *LolReplayDataExtract) GetMatchVersion() string {
	if m != nil {
		return m.MatchVersion
	}
	return ""
}

func (m *LolReplayDataExtract) GetPlatformId() string {
	if m != nil {
		return m.PlatformId
	}
	return ""
}

func (m *LolReplayDataExtract) GetSpectatorServer() string {
	if m != nil {
		return m.SpectatorServer
	}
	return ""
}

func (m *LolReplayDataExtract) GetSummonerIds() []int64 {
	if m != nil {
		return m.SummonerIds
	}
	return nil
}

func (m *LolReplayDataExtract) GetOverride() bool {
	if m != nil {
		return m.Override
	}
	return false
}

func (m *LolReplayDataExtract) GetRetryNum() int64 {
	if m != nil {
		return m.RetryNum
	}
	return 0
}

// LolSummonerCrawl describes a summoner to retrieve recent matches for.
type LolSummonerCrawl struct {
	SummonerId  int64  `protobuf:"varint,1,opt,name=summoner_id,json=summonerId" json:"summoner_id,omitempty"`
	PlatformId  string `protobuf:"bytes,2,opt,name=platform_id,json=platformId" json:"platform_id,omitempty"`
	HistoryType string `protobuf:"bytes,3,opt,name=history_type,json=historyType" json:"history_type,omitempty"`
	// RFC 3339 timestamp string. only relevant if history_type == "ranked_history"
	Since string `protobuf:"bytes,4,opt,name=since" json:"since,omitempty"`
}

func (m *LolSummonerCrawl) Reset()                    { *m = LolSummonerCrawl{} }
func (m *LolSummonerCrawl) String() string            { return proto.CompactTextString(m) }
func (*LolSummonerCrawl) ProtoMessage()               {}
func (*LolSummonerCrawl) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *LolSummonerCrawl) GetSummonerId() int64 {
	if m != nil {
		return m.SummonerId
	}
	return 0
}

func (m *LolSummonerCrawl) GetPlatformId() string {
	if m != nil {
		return m.PlatformId
	}
	return ""
}

func (m *LolSummonerCrawl) GetHistoryType() string {
	if m != nil {
		return m.HistoryType
	}
	return ""
}

func (m *LolSummonerCrawl) GetSince() string {
	if m != nil {
		return m.Since
	}
	return ""
}

// LolVideo describes a single match/summoner video to generate.
type LolVideo struct {
	MatchId           int64  `protobuf:"varint,1,opt,name=match_id,json=matchId" json:"match_id,omitempty"`
	PlatformId        string `protobuf:"bytes,2,opt,name=platform_id,json=platformId" json:"platform_id,omitempty"`
	Key               string `protobuf:"bytes,3,opt,name=key" json:"key,omitempty"`
	ChampFocus        int64  `protobuf:"varint,4,opt,name=champ_focus,json=champFocus" json:"champ_focus,omitempty"`
	GameLengthSeconds int64  `protobuf:"varint,5,opt,name=game_length_seconds,json=gameLengthSeconds" json:"game_length_seconds,omitempty"`
	SpectatorServer   string `protobuf:"bytes,6,opt,name=spectator_server,json=spectatorServer" json:"spectator_server,omitempty"`
	MatchVersion      string `protobuf:"bytes,7,opt,name=match_version,json=matchVersion" json:"match_version,omitempty"`
	TargetSummonerId  int64  `protobuf:"varint,8,opt,name=target_summoner_id,json=targetSummonerId" json:"target_summoner_id,omitempty"`
	MatchDetailsPath  string `protobuf:"bytes,9,opt,name=match_details_path,json=matchDetailsPath" json:"match_details_path,omitempty"`
}

func (m *LolVideo) Reset()                    { *m = LolVideo{} }
func (m *LolVideo) String() string            { return proto.CompactTextString(m) }
func (*LolVideo) ProtoMessage()               {}
func (*LolVideo) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *LolVideo) GetMatchId() int64 {
	if m != nil {
		return m.MatchId
	}
	return 0
}

func (m *LolVideo) GetPlatformId() string {
	if m != nil {
		return m.PlatformId
	}
	return ""
}

func (m *LolVideo) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *LolVideo) GetChampFocus() int64 {
	if m != nil {
		return m.ChampFocus
	}
	return 0
}

func (m *LolVideo) GetGameLengthSeconds() int64 {
	if m != nil {
		return m.GameLengthSeconds
	}
	return 0
}

func (m *LolVideo) GetSpectatorServer() string {
	if m != nil {
		return m.SpectatorServer
	}
	return ""
}

func (m *LolVideo) GetMatchVersion() string {
	if m != nil {
		return m.MatchVersion
	}
	return ""
}

func (m *LolVideo) GetTargetSummonerId() int64 {
	if m != nil {
		return m.TargetSummonerId
	}
	return 0
}

func (m *LolVideo) GetMatchDetailsPath() string {
	if m != nil {
		return m.MatchDetailsPath
	}
	return ""
}

// LolGoalsUpdate describes a request to update goals based on a single match
type LolGoalsUpdate struct {
	MatchId      int64  `protobuf:"varint,1,opt,name=match_id,json=matchId" json:"match_id,omitempty"`
	PlatformId   string `protobuf:"bytes,2,opt,name=platform_id,json=platformId" json:"platform_id,omitempty"`
	SummonerId   int64  `protobuf:"varint,3,opt,name=summoner_id,json=summonerId" json:"summoner_id,omitempty"`
	RolePosition string `protobuf:"bytes,4,opt,name=role_position,json=rolePosition" json:"role_position,omitempty"`
	ChampionId   int64  `protobuf:"varint,5,opt,name=champion_id,json=championId" json:"champion_id,omitempty"`
}

func (m *LolGoalsUpdate) Reset()                    { *m = LolGoalsUpdate{} }
func (m *LolGoalsUpdate) String() string            { return proto.CompactTextString(m) }
func (*LolGoalsUpdate) ProtoMessage()               {}
func (*LolGoalsUpdate) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *LolGoalsUpdate) GetMatchId() int64 {
	if m != nil {
		return m.MatchId
	}
	return 0
}

func (m *LolGoalsUpdate) GetPlatformId() string {
	if m != nil {
		return m.PlatformId
	}
	return ""
}

func (m *LolGoalsUpdate) GetSummonerId() int64 {
	if m != nil {
		return m.SummonerId
	}
	return 0
}

func (m *LolGoalsUpdate) GetRolePosition() string {
	if m != nil {
		return m.RolePosition
	}
	return ""
}

func (m *LolGoalsUpdate) GetChampionId() int64 {
	if m != nil {
		return m.ChampionId
	}
	return 0
}

func init() {
	proto.RegisterType((*LolAdvancedStatsIngest)(nil), "messages.LolAdvancedStatsIngest")
	proto.RegisterType((*LolBasicStatsIngest)(nil), "messages.LolBasicStatsIngest")
	proto.RegisterType((*LolEloDataProcess)(nil), "messages.LolEloDataProcess")
	proto.RegisterType((*LolMatchDownload)(nil), "messages.LolMatchDownload")
	proto.RegisterType((*LolReplayDataExtract)(nil), "messages.LolReplayDataExtract")
	proto.RegisterType((*LolSummonerCrawl)(nil), "messages.LolSummonerCrawl")
	proto.RegisterType((*LolVideo)(nil), "messages.LolVideo")
	proto.RegisterType((*LolGoalsUpdate)(nil), "messages.LolGoalsUpdate")
}

func init() { proto.RegisterFile("lol.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 725 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xa4, 0x55, 0xc1, 0x6e, 0xd3, 0x4c,
	0x18, 0x94, 0xe3, 0xa4, 0x75, 0x36, 0xc9, 0xff, 0xa7, 0x6e, 0xa9, 0x02, 0x1c, 0x1a, 0xd2, 0x4b,
	0x90, 0xaa, 0x1e, 0xe0, 0xc2, 0x15, 0x68, 0x41, 0x91, 0x0c, 0x8a, 0x1c, 0xe8, 0xd5, 0xda, 0x78,
	0xbf, 0x26, 0x56, 0xd7, 0xfe, 0xac, 0xdd, 0x4d, 0x4a, 0x1e, 0x82, 0xf7, 0xe0, 0xce, 0x09, 0x89,
	0xd7, 0xe0, 0xc4, 0x85, 0x47, 0x41, 0x5e, 0xc7, 0xa9, 0x13, 0x3b, 0x01, 0xd4, 0x5b, 0x3c, 0xf3,
	0x39, 0x9a, 0xf9, 0x66, 0x76, 0x4d, 0xea, 0x1c, 0xf9, 0x79, 0x2c, 0x50, 0xa1, 0x6d, 0x85, 0x20,
	0x25, 0x9d, 0x80, 0xec, 0xfd, 0x34, 0xc8, 0xb1, 0x83, 0xfc, 0x25, 0x9b, 0xd3, 0xc8, 0x07, 0x36,
	0x52, 0x54, 0xc9, 0x41, 0x34, 0x01, 0xa9, 0xec, 0x53, 0xd2, 0x1a, 0x53, 0x09, 0xf3, 0x00, 0x6e,
	0xbd, 0x98, 0xaa, 0x69, 0xc7, 0xe8, 0x1a, 0xfd, 0xba, 0xdb, 0xcc, 0xc0, 0x21, 0x55, 0xd3, 0xb5,
	0x21, 0xb5, 0x88, 0xa1, 0x53, 0x59, 0x1f, 0xfa, 0xb0, 0x88, 0xc1, 0x7e, 0x48, 0xac, 0x90, 0x2a,
	0x7f, 0xea, 0x05, 0xac, 0x63, 0x76, 0x8d, 0xbe, 0xe9, 0xee, 0xeb, 0xe7, 0x01, 0xb3, 0x4f, 0x48,
	0x23, 0xe6, 0x54, 0x5d, 0xa3, 0x08, 0x13, 0xb6, 0xaa, 0xdf, 0x26, 0x19, 0x94, 0x0e, 0xc8, 0x59,
	0x18, 0x62, 0x04, 0x22, 0x19, 0xa8, 0xe9, 0xd7, 0x49, 0x06, 0x0d, 0x98, 0xfd, 0x88, 0x58, 0x38,
	0x07, 0x21, 0x02, 0x06, 0x9d, 0xbd, 0xae, 0xd1, 0xb7, 0xdc, 0xd5, 0x73, 0xef, 0xbb, 0x41, 0x0e,
	0x1d, 0xe4, 0xaf, 0xa8, 0x0c, 0xfc, 0xbc, 0xb5, 0x33, 0x62, 0xa7, 0x82, 0x18, 0x28, 0x1a, 0x70,
	0x99, 0xf7, 0xd7, 0xd6, 0xcc, 0x45, 0x4a, 0x68, 0x8f, 0x79, 0xf9, 0x95, 0x9d, 0xf2, 0xcd, 0x3f,
	0xc9, 0xaf, 0xee, 0x94, 0x5f, 0xdb, 0x90, 0xff, 0xcb, 0x20, 0x07, 0x0e, 0xf2, 0x4b, 0x8e, 0x17,
	0x54, 0xd1, 0xa1, 0x40, 0x1f, 0xa4, 0xb4, 0x7b, 0xa4, 0x05, 0x1c, 0x3d, 0x46, 0x15, 0xcd, 0xeb,
	0x6e, 0xc0, 0x72, 0x2c, 0x91, 0x5c, 0x6e, 0xb0, 0xf2, 0x17, 0x06, 0xff, 0x35, 0x9f, 0x27, 0xa4,
	0x99, 0x33, 0x28, 0x3b, 0xb5, 0xae, 0xd9, 0x37, 0xdd, 0xc6, 0x9d, 0x43, 0xb9, 0x33, 0xa1, 0x6f,
	0x06, 0x69, 0x3b, 0xc8, 0xdf, 0x69, 0x49, 0x78, 0x1b, 0x71, 0xa4, 0xcc, 0x6e, 0x13, 0xf3, 0x06,
	0x16, 0x4b, 0x5f, 0xc9, 0xcf, 0x7b, 0x45, 0xf0, 0x8c, 0x3c, 0xc0, 0xb1, 0x04, 0x31, 0x07, 0xe6,
	0xad, 0x49, 0xad, 0x6a, 0xa9, 0x87, 0x19, 0x39, 0xca, 0x49, 0x3e, 0x25, 0x2d, 0x01, 0x31, 0xa7,
	0x0b, 0x4f, 0x73, 0x42, 0x47, 0x53, 0x77, 0x9b, 0x29, 0x38, 0xd2, 0x58, 0xef, 0x8b, 0x49, 0x8e,
	0x1c, 0xe4, 0xae, 0xc6, 0x92, 0xd5, 0x5f, 0x7e, 0x52, 0x82, 0xfa, 0xca, 0x3e, 0x27, 0x87, 0x13,
	0x1a, 0x82, 0xc7, 0x21, 0x9a, 0xa8, 0xa9, 0x27, 0xc1, 0xc7, 0x88, 0x49, 0xed, 0xc7, 0x74, 0x0f,
	0x12, 0xca, 0xd1, 0xcc, 0x28, 0x25, 0xec, 0x17, 0xa4, 0x13, 0x44, 0xbe, 0x00, 0x2a, 0x81, 0x79,
	0xc9, 0x9f, 0x8d, 0xa9, 0x7f, 0xe3, 0xc9, 0x18, 0x20, 0x75, 0x6b, 0xb9, 0xc7, 0x2b, 0x7e, 0xb8,
	0xa4, 0x47, 0x09, 0x9b, 0x6d, 0xca, 0x2c, 0xdf, 0x54, 0x75, 0x7d, 0x53, 0xe5, 0xa5, 0xa8, 0x6d,
	0x29, 0xc5, 0x29, 0x69, 0xa5, 0xd3, 0x73, 0x10, 0x32, 0xc0, 0x48, 0x47, 0x57, 0x77, 0x9b, 0x1a,
	0xbc, 0x4a, 0xb1, 0xcd, 0xe5, 0xef, 0x17, 0x96, 0xff, 0x94, 0xb4, 0x65, 0x0c, 0xbe, 0xa2, 0x0a,
	0x45, 0xb6, 0x4b, 0x4b, 0x4f, 0xfd, 0xbf, 0xc2, 0xd3, 0x75, 0x16, 0x9a, 0x54, 0xdf, 0xdd, 0x24,
	0xb2, 0xde, 0x24, 0xfb, 0x31, 0xa9, 0x0b, 0x50, 0x62, 0xe1, 0x45, 0xb3, 0xb0, 0xd3, 0xd0, 0xce,
	0x2d, 0x0d, 0xbc, 0x9f, 0x85, 0xbd, 0xcf, 0x69, 0xcd, 0xb2, 0x88, 0x5f, 0x0b, 0x7a, 0xcb, 0x37,
	0xcf, 0xa6, 0x51, 0x38, 0x9b, 0x1b, 0xee, 0x2a, 0x65, 0xe5, 0x9f, 0x06, 0x52, 0xa1, 0x58, 0xa4,
	0x97, 0x5f, 0x9a, 0x43, 0x63, 0x89, 0xe9, 0xbb, 0xef, 0x88, 0xd4, 0x64, 0x10, 0xf9, 0xb0, 0x3c,
	0x3a, 0xe9, 0x43, 0xef, 0x47, 0x85, 0x58, 0x0e, 0xf2, 0xab, 0x80, 0x01, 0xae, 0x45, 0x66, 0xec,
	0x2c, 0x77, 0x51, 0x41, 0xb1, 0x00, 0x27, 0xa4, 0xe1, 0x4f, 0x69, 0x18, 0x7b, 0xd7, 0xe8, 0xcf,
	0x64, 0x76, 0xe3, 0x68, 0xe8, 0x4d, 0x82, 0x6c, 0x6b, 0x67, 0x6d, 0x5b, 0x3b, 0xcb, 0x22, 0xdc,
	0x2b, 0x8f, 0xb0, 0xd0, 0x99, 0xfd, 0x92, 0xce, 0x9c, 0x11, 0x5b, 0x51, 0x31, 0x01, 0x95, 0x3f,
	0x8d, 0xba, 0x14, 0xa6, 0xdb, 0x4e, 0x99, 0xbb, 0xa3, 0xb8, 0xa5, 0xb4, 0xf5, 0xf2, 0xd2, 0xf6,
	0xbe, 0x1a, 0xe4, 0x3f, 0x07, 0xf9, 0x5b, 0xa4, 0x5c, 0x7e, 0x8c, 0x19, 0x55, 0x70, 0xaf, 0xed,
	0x6e, 0x34, 0xc4, 0x2c, 0x34, 0x24, 0xb9, 0x27, 0x90, 0x83, 0x17, 0xa3, 0x0c, 0x54, 0x62, 0xb8,
	0xba, 0xbc, 0x27, 0x90, 0xc3, 0x70, 0x89, 0xad, 0x12, 0x09, 0x30, 0xca, 0x7d, 0xc2, 0x32, 0x68,
	0xc0, 0xc6, 0x7b, 0xfa, 0xab, 0xfc, 0xfc, 0x77, 0x00, 0x00, 0x00, 0xff, 0xff, 0x6f, 0x1b, 0xf5,
	0x75, 0xa2, 0x07, 0x00, 0x00,
}
