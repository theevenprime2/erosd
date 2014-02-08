package main

import (
	"errors"
	"fmt"
	"github.com/Starbow/erosd/buffers"
	"log"
	"math/rand"
)

var _ = log.Ldate
var (
	ErrLadderPlayerNotFound           = errors.New("The player was not found in the database.")
	ErrLadderClientNotInvolved        = errors.New("None of the client's registered characters were found in the replay participant list.")
	ErrLadderInvalidMatchParticipents = errors.New("All participents of a game must be registered.")
	ErrLadderInvalidMap               = errors.New("Matches must be on a valid map in the map pool.")
	ErrLadderInvalidFormat            = errors.New("Matches must be a 1v1 with no observers.")
	ErrLadderDuplicateReplay          = errors.New("The provided has been processed previously.")
	ErrLadderGameTooShort             = errors.New("The provided game was too short.")
	ErrLadderWrongOpponent            = errors.New("The provided game was not against your matchmade opponent. You have been forefeited.")
	ErrLadderWrongMap                 = errors.New("The provided game was not on the correct map.")
)

type Division struct {
	Name   string
	Points int64
}

type Divisions []Division
type Maps map[int64]*Map

var (
	divisionNames []string = []string{"Bronze", "Silver", "Gold", "Platinum", "Diamond"}
	divisions     Divisions
	maps          Maps = Maps{
		1: &Map{
			Id:            1,
			Region:        BATTLENET_REGION_EU,
			BattleNetName: "Starbow - Texas 3.0",
			InRankedPool:  true,
		},
		2: &Map{
			Id:            2,
			Region:        BATTLENET_REGION_NA,
			BattleNetName: "Frost LE",
			InRankedPool:  false,
		},
		3: &Map{
			Id:            3,
			Region:        BATTLENET_REGION_NA,
			BattleNetName: "Starbow - Fighting Spirit",
			InRankedPool:  true,
		},
		4: &Map{
			Id:            4,
			Region:        BATTLENET_REGION_NA,
			BattleNetName: "Starbow - Circuit breaker",
			InRankedPool:  true,
		},
		5: &Map{
			Id:            5,
			Region:        BATTLENET_REGION_NA,
			BattleNetName: "Starbow - Neo Tau Cross",
			InRankedPool:  true,
		},
	}

	divisionCount             int64
	subdivisionCount          int64
	divisionPoints            int64
	ladderStartingPoints      int64   = 1250
	ladderWinPointsBase       int64   = 100
	ladderLosePointsBase      int64   = 50
	ladderWinPointsIncrement  float64 = 25
	ladderLosePointsIncrement float64 = 12.5
	ladderMaxMapVetos         int64   = 3
)

// Create divisions. There will be [subdivisionCount] subdivisions per division.
// The final division will only have one subdivision.
func initDivisions() {
	divisions = make(Divisions, 0, divisionCount*subdivisionCount+1)

	divisionSize := divisionPoints * subdivisionCount

	i := int64(0)
	for {
		if i == divisionCount {
			break
		}

		j := int64(0)
		for {
			if j == subdivisionCount {
				break
			}

			divisions = append(divisions, Division{
				Points: int64((divisionSize * i) + (divisionPoints * j)),
				Name:   fmt.Sprintf("%s %d", divisionNames[i], subdivisionCount-j),
			})

			j++
		}

		i++
	}

	divisions = append(divisions, Division{
		Points: int64((divisionSize * i)),
		Name:   fmt.Sprintf("%s", divisionNames[i]),
	})

	return
}

func (d Divisions) GetDivision(points int64) (division *Division, position int64) {
	i := int64(len(d))
	for {
		i--

		if points >= d[i].Points {
			return &d[i], i
		}

		if i == 0 {
			break
		}
	}

	return nil, 0
}

// Get the difference in ranks
func (d Divisions) GetDifference(points, points2 int64) int64 {
	_, p1 := d.GetDivision(points)
	_, p2 := d.GetDivision(points2)

	return p2 - p1
}

func (m Maps) Get(region BattleNetRegion, name string) *Map {
	for x := range m {
		if m[x].Region != region {
			continue
		}

		if m[x].BattleNetName == name {
			return m[x]
		}
	}

	return nil
}

func (m Maps) Random(region BattleNetRegion, veto ...[]*Map) *Map {
	var pool []*Map = make([]*Map, 0, 5)

mapLoop:
	for x := range m {
		if m[x].Region != region || !m[x].InRankedPool {
			continue
		}

		// Check vetoes, and continue main loop if found
		for y := range veto {
			for z := range veto[y] {
				if veto[y][z] == nil {
					continue
				}

				if m[x].BattleNetID == veto[y][z].BattleNetID && m[x].Region == veto[y][z].Region {
					continue mapLoop
				}
			}
		}

		pool = append(pool, m[x])
	}

	if len(pool) == 0 {
		return nil
	}

	return pool[rand.Intn(len(pool))]

}

type Map struct {
	Id            int64
	Region        BattleNetRegion
	BattleNetID   int
	BattleNetName string
	InRankedPool  bool
}

func (m *Map) MapMessage() protobufs.Map {
	var (
		msg    protobufs.Map
		region protobufs.Region = protobufs.Region(m.Region)
		id     int32            = int32(m.BattleNetID)
	)
	msg.Region = &region
	msg.BattleNetName = &m.BattleNetName
	msg.BattleNetId = &id

	return msg
}

type MapVeto struct {
	Id       int64
	ClientId int64
	MapId    int64
}

type MatchResult struct {
	Id                int64
	MapId             int64 // Map
	MatchmakerMatchId int64
	DateTime          int64 // unix
}

type MatchResultPlayer struct {
	Id               int64
	MatchId          int64
	ClientId         int64
	CharacterId      int64
	PointsBefore     int64
	PointsAfter      int64
	PointsDifference int64
	Race             string
	Victory          bool
}

type MatchResultSource struct {
	Id         int64
	MatchId    int64
	ReplayHash string
}

func NewMatchResult(replay *Replay, client *Client) (result *MatchResult, players []*MatchResultPlayer, err error) {
	// Find the local character
	// Find the opponent
	log.Println(*replay)
	region := ParseBattleNetRegion(replay.Region)

	if replay.GameLength < 120 {
		err = ErrLadderGameTooShort
		return
	}

	m := maps.Get(region, replay.MapName)

	if m == nil || !m.InRankedPool {
		err = ErrLadderInvalidMap
		return
	}

	if len(replay.Observers) > 0 || len(replay.Players) != 2 {
		err = ErrLadderInvalidFormat
		return
	}

	count, err := dbMap.SelectInt("SELECT COUNT(*) FROM match_result_sources WHERE ReplayHash=?", replay.Filehash)
	if err == nil && count > 0 {
		err = ErrLadderDuplicateReplay
		return
	}

	var (
		player   *MatchResultPlayer
		opponent *MatchResultPlayer
	)
	for x := range replay.Players {
		mrp, merr := NewMatchResultPlayer(replay, &replay.Players[x])
		if merr != nil || mrp == nil {
			err = ErrLadderInvalidMatchParticipents
			return
		}

		if mrp.ClientId == client.Id {
			player = mrp
		} else {
			opponent = mrp
		}
	}

	//We don't have the player that submitted the replay.
	if player == nil {
		err = ErrLadderClientNotInvolved
		return
	}

	//We don't have an opponent.
	if opponent == nil {
		err = ErrLadderInvalidMatchParticipents
		return
	}

	//Make sure we only have one and only one victor.
	if player.Victory == opponent.Victory {
		err = ErrLadderInvalidFormat
		return
	}

	// We're only going to accept replays from the victor.
	// In future this should be changed to match games against the start time
	// I made a nice client-ID based mutex manager just for this purpose.
	clientLockouts.LockIds(player.ClientId, opponent.ClientId)
	clientLockouts.UnlockIds(player.ClientId, opponent.ClientId)

	var res MatchResult
	res.DateTime = replay.UnixTimestamp
	res.MapId = m.Id
	res.MatchmakerMatchId = client.PendingMatchmakingId

	if player.Victory {
		opponentClient := clientCache.Get(opponent.ClientId)

		if opponentClient == nil {
			err = ErrLadderPlayerNotFound // new error for lookup fail
			log.Println("wut")
			return
		}

		if !client.IsMatchedWith(opponentClient) {
			client.ForefeitMatchmadeMatch()
			err = ErrLadderWrongOpponent
			return
		}

		if !opponentClient.IsMatchedWith(client) {
			opponentClient.ForefeitMatchmadeMatch()
		}

		var source MatchResultSource
		source.MatchId = res.Id
		source.ReplayHash = replay.Filehash
		dbMap.Insert(&source)

		if !client.IsOnMap(m.Id) {
			err = ErrLadderWrongMap
			return
		}

		err = dbMap.Insert(&res)
		if err != nil {
			err = ErrDbInsert
			return
		}
		player.MatchId = res.Id
		opponent.MatchId = res.Id

		player.PointsBefore = client.LadderPoints
		opponent.PointsBefore = opponentClient.LadderPoints

		client.Defeat(opponentClient)

		player.PointsAfter = client.LadderPoints
		opponent.PointsAfter = opponentClient.LadderPoints

		player.PointsDifference = player.PointsAfter - player.PointsBefore
		opponent.PointsDifference = player.PointsAfter - player.PointsBefore

		client.PendingMatchmakingId = 0
		client.PendingMatchmakingOpponentId = 0
		opponentClient.PendingMatchmakingId = 0
		opponentClient.PendingMatchmakingOpponentId = 0
		_, uerr := dbMap.Update(client, opponentClient)
		if uerr != nil {
			err = ErrDbInsert
			return
		}

		dbMap.Insert(player, opponent)
		result = &res
		players = []*MatchResultPlayer{player, opponent}

		go client.BroadcastStatsMessage()
		go opponentClient.BroadcastStatsMessage()
	}

	return
}

func NewMatchResultPlayer(replay *Replay, player *ReplayPlayer) (matchplayer *MatchResultPlayer, err error) {
	region, subregion, id, _ := ParseBattleNetProfileUrl(player.Url)
	character := characterCache.Get(region, subregion, id)

	if character == nil {
		return nil, ErrLadderPlayerNotFound
	}

	var mrp MatchResultPlayer

	mrp.CharacterId = character.Id
	mrp.ClientId = character.ClientId
	mrp.Race = player.GameRace
	mrp.Victory = player.Victory == "Win"

	return &mrp, nil
}