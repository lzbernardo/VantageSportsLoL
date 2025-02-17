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

func init() {
	proto.RegisterType((*LolAdvancedStatsIngest)(nil), "messages.LolAdvancedStatsIngest")
	proto.RegisterType((*LolBasicStatsIngest)(nil), "messages.LolBasicStatsIngest")
	proto.RegisterType((*LolEloDataProcess)(nil), "messages.LolEloDataProcess")
	proto.RegisterType((*LolMatchDownload)(nil), "messages.LolMatchDownload")
	proto.RegisterType((*LolReplayDataExtract)(nil), "messages.LolReplayDataExtract")
	proto.RegisterType((*LolSummonerCrawl)(nil), "messages.LolSummonerCrawl")
	proto.RegisterType((*LolVideo)(nil), "messages.LolVideo")
}

func init() { proto.RegisterFile("lol.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 674 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xa4, 0x55, 0xc1, 0x6e, 0xd3, 0x4a,
	0x14, 0x95, 0xe3, 0xa4, 0xb5, 0x6f, 0x12, 0xbd, 0xd4, 0xed, 0xab, 0xfc, 0x1e, 0x8b, 0x86, 0x74,
	0x13, 0xa4, 0xaa, 0x0b, 0xd8, 0xb0, 0x05, 0x5a, 0xa4, 0x4a, 0x06, 0x55, 0x0e, 0xea, 0xd6, 0x9a,
	0x78, 0x6e, 0x13, 0xab, 0x63, 0x8f, 0x35, 0x33, 0x49, 0xc9, 0x47, 0xf0, 0x1f, 0xfc, 0x02, 0x12,
	0xbf, 0xc1, 0x8a, 0x0d, 0x9f, 0x82, 0x3c, 0x13, 0xb7, 0x4e, 0xe2, 0x04, 0x10, 0xbb, 0xce, 0x39,
	0xd7, 0xd5, 0x39, 0xf7, 0x9c, 0x99, 0x80, 0xcb, 0x38, 0x3b, 0xcf, 0x05, 0x57, 0xdc, 0x73, 0x52,
	0x94, 0x92, 0x4c, 0x50, 0x0e, 0xbe, 0x5b, 0x70, 0x1c, 0x70, 0xf6, 0x8a, 0xce, 0x49, 0x16, 0x23,
	0x1d, 0x29, 0xa2, 0xe4, 0x55, 0x36, 0x41, 0xa9, 0xbc, 0x53, 0xe8, 0x8e, 0x89, 0xc4, 0x79, 0x82,
	0xf7, 0x51, 0x4e, 0xd4, 0xd4, 0xb7, 0xfa, 0xd6, 0xd0, 0x0d, 0x3b, 0x25, 0x78, 0x4d, 0xd4, 0x74,
	0x65, 0x48, 0x2d, 0x72, 0xf4, 0x1b, 0xab, 0x43, 0x1f, 0x16, 0x39, 0x7a, 0xff, 0x81, 0x93, 0x12,
	0x15, 0x4f, 0xa3, 0x84, 0xfa, 0x76, 0xdf, 0x1a, 0xda, 0xe1, 0xbe, 0x3e, 0x5f, 0x51, 0xef, 0x04,
	0xda, 0x39, 0x23, 0xea, 0x96, 0x8b, 0xb4, 0x60, 0x9b, 0xfa, 0x6b, 0x28, 0x21, 0x33, 0x20, 0x67,
	0x69, 0xca, 0x33, 0x14, 0xc5, 0x40, 0x4b, 0x7f, 0x0e, 0x25, 0x74, 0x45, 0xbd, 0xff, 0xc1, 0xe1,
	0x73, 0x14, 0x22, 0xa1, 0xe8, 0xef, 0xf5, 0xad, 0xa1, 0x13, 0x3e, 0x9c, 0x07, 0x5f, 0x2d, 0x38,
	0x0c, 0x38, 0x7b, 0x4d, 0x64, 0x12, 0x57, 0xad, 0x9d, 0x81, 0x67, 0x04, 0x51, 0x54, 0x24, 0x61,
	0xb2, 0xea, 0xaf, 0xa7, 0x99, 0x0b, 0x43, 0x68, 0x8f, 0x55, 0xf9, 0x8d, 0x9d, 0xf2, 0xed, 0x5f,
	0xc9, 0x6f, 0xee, 0x94, 0xdf, 0x5a, 0x93, 0xff, 0xc3, 0x82, 0x83, 0x80, 0xb3, 0x4b, 0xc6, 0x2f,
	0x88, 0x22, 0xd7, 0x82, 0xc7, 0x28, 0xa5, 0x37, 0x80, 0x2e, 0x32, 0x1e, 0x51, 0xa2, 0x48, 0x55,
	0x77, 0x1b, 0x97, 0x63, 0x85, 0xe4, 0x7a, 0x83, 0x8d, 0xdf, 0x30, 0xf8, 0xa7, 0xf9, 0x3c, 0x85,
	0x4e, 0xc5, 0xa0, 0xf4, 0x5b, 0x7d, 0x7b, 0x68, 0x87, 0xed, 0x47, 0x87, 0x72, 0x67, 0x42, 0x5f,
	0x2c, 0xe8, 0x05, 0x9c, 0xbd, 0xd3, 0x92, 0xf8, 0x7d, 0xc6, 0x38, 0xa1, 0x5e, 0x0f, 0xec, 0x3b,
	0x5c, 0x2c, 0x7d, 0x15, 0x7f, 0xfe, 0x55, 0x04, 0xcf, 0xe1, 0x5f, 0x3e, 0x96, 0x28, 0xe6, 0x48,
	0xa3, 0x15, 0xa9, 0x4d, 0x2d, 0xf5, 0xb0, 0x24, 0x47, 0x15, 0xc9, 0xa7, 0xd0, 0x15, 0x98, 0x33,
	0xb2, 0x88, 0x34, 0x27, 0x74, 0x34, 0x6e, 0xd8, 0x31, 0xe0, 0x48, 0x63, 0x83, 0xcf, 0x36, 0x1c,
	0x05, 0x9c, 0x85, 0x1a, 0x2b, 0x56, 0x7f, 0xf9, 0x51, 0x09, 0x12, 0x2b, 0xef, 0x1c, 0x0e, 0x27,
	0x24, 0xc5, 0x88, 0x61, 0x36, 0x51, 0xd3, 0x48, 0x62, 0xcc, 0x33, 0x2a, 0xb5, 0x1f, 0x3b, 0x3c,
	0x28, 0xa8, 0x40, 0x33, 0x23, 0x43, 0x78, 0x2f, 0xc1, 0x4f, 0xb2, 0x58, 0x20, 0x91, 0x48, 0xa3,
	0xe2, 0x9f, 0x8d, 0x49, 0x7c, 0x17, 0xc9, 0x1c, 0xd1, 0xb8, 0x75, 0xc2, 0xe3, 0x07, 0xfe, 0x7a,
	0x49, 0x8f, 0x0a, 0xb6, 0xdc, 0x94, 0x5d, 0xbf, 0xa9, 0xe6, 0xea, 0xa6, 0xea, 0x4b, 0xd1, 0xda,
	0x52, 0x8a, 0x53, 0xe8, 0x9a, 0xe9, 0x39, 0x0a, 0x99, 0xf0, 0x4c, 0x47, 0xe7, 0x86, 0x1d, 0x0d,
	0xde, 0x18, 0x6c, 0x7d, 0xf9, 0xfb, 0x1b, 0xcb, 0x7f, 0x06, 0x3d, 0x99, 0x63, 0xac, 0x88, 0xe2,
	0xa2, 0xdc, 0xa5, 0xa3, 0xa7, 0xfe, 0x79, 0xc0, 0xcd, 0x3a, 0x37, 0x9a, 0xe4, 0xee, 0x6e, 0x12,
	0xac, 0x36, 0xc9, 0x7b, 0x02, 0xae, 0x40, 0x25, 0x16, 0x51, 0x36, 0x4b, 0xfd, 0xb6, 0x76, 0xee,
	0x68, 0xe0, 0xfd, 0x2c, 0x1d, 0x7c, 0x32, 0x35, 0x2b, 0x23, 0x7e, 0x23, 0xc8, 0x3d, 0x5b, 0xbf,
	0x9b, 0xd6, 0xc6, 0xdd, 0x5c, 0x73, 0xd7, 0xa8, 0x2b, 0xff, 0x34, 0x91, 0x8a, 0x8b, 0x85, 0x79,
	0xfc, 0x4c, 0x0e, 0xed, 0x25, 0xa6, 0xdf, 0xbe, 0x23, 0x68, 0xc9, 0x24, 0x8b, 0x71, 0x79, 0x75,
	0xcc, 0x61, 0xf0, 0xad, 0x01, 0x4e, 0xc0, 0xd9, 0x4d, 0x42, 0x91, 0xaf, 0x44, 0x66, 0xed, 0x2c,
	0xf7, 0xa6, 0x82, 0xcd, 0x02, 0x9c, 0x40, 0x3b, 0x9e, 0x92, 0x34, 0x8f, 0x6e, 0x79, 0x3c, 0x93,
	0xe5, 0x8b, 0xa3, 0xa1, 0xb7, 0x05, 0xb2, 0xad, 0x9d, 0xad, 0x6d, 0xed, 0xac, 0x8b, 0x70, 0xaf,
	0x3e, 0xc2, 0x8d, 0xce, 0xec, 0xd7, 0x74, 0xe6, 0x0c, 0x3c, 0x45, 0xc4, 0x04, 0x55, 0xf5, 0x36,
	0xea, 0x52, 0xd8, 0x61, 0xcf, 0x30, 0x8f, 0x57, 0x71, 0x4b, 0x69, 0xdd, 0xfa, 0xd2, 0x8e, 0xf7,
	0xf4, 0xef, 0xdb, 0x8b, 0x9f, 0x01, 0x00, 0x00, 0xff, 0xff, 0xe5, 0x79, 0xaa, 0x90, 0xec, 0x06,
	0x00, 0x00,
}
