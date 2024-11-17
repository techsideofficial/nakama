package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama/v3/server/evr"
	"github.com/intinig/go-openskill/rating"
	"github.com/intinig/go-openskill/types"
	"go.uber.org/thriftrw/ptr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type skillBasedMatchmaker struct{}

var SkillBasedMatchmaker = &skillBasedMatchmaker{}

func (*skillBasedMatchmaker) TeamStrength(team RatedEntryTeam) float64 {
	s := 0.0
	for _, p := range team {
		s += p.Rating.Mu
	}
	return s
}

var writerMu sync.Mutex

// Function to be used as a matchmaker function in Nakama (RegisterMatchmakerOverride)
func (m *skillBasedMatchmaker) EvrMatchmakerFn(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, candidates [][]runtime.MatchmakerEntry) [][]runtime.MatchmakerEntry {

	allPlayers := make(map[string]struct{}, 0)
	for _, c := range candidates {
		for _, e := range c {
			allPlayers[e.GetPresence().GetUserId()] = struct{}{}
		}
	}

	logger.WithFields(map[string]interface{}{
		"num_candidates": len(candidates),
	}).Info("Running skill-based matchmaker.")

	if len(candidates) == 0 || len(candidates[0]) == 0 {
		return nil
	}
	groupID := candidates[0][0].GetProperties()["group_id"].(string)
	if groupID == "" {
		logger.Error("Group ID not found in entry properties.")
	}

	// Write the candidates to storage
	if writerMu.TryLock() {
		defer writerMu.Unlock()
		if data, err := json.Marshal(map[string]interface{}{"candidates": candidates}); err != nil {
			logger.WithField("error", err).Error("Error marshalling candidates.")
		} else if len(data) > 8*1024*1024 {
			logger.WithField("size", len(data)).Error("Data is too large to write to storage.")
		} else {
			data, err := json.Marshal(map[string]interface{}{"candidates": candidates})
			if err != nil {
				logger.WithField("error", err).Error("Error marshalling candidates.")
			} else {
				if _, err := nk.StorageWrite(ctx, []*runtime.StorageWrite{
					{
						UserID:          SystemUserID,
						Collection:      "Matchmaker",
						Key:             "latestCandidates",
						PermissionRead:  0,
						PermissionWrite: 0,
						Value:           string(data),
					},
				}); err != nil {
					logger.WithField("error", err).Error("Error writing latest candidates to storage.")
				}

				// Send the candidates to the stream as well
				if err := nk.StreamSend(StreamModeMatchmaker, groupID, "", "", string(data), nil, false); err != nil {
					logger.WithField("error", err).Warn("Error streaming candidates")
				}
			}
		}
	}

	filterCounts := make(map[string]int)

	// Remove odd sized teams
	candidates, filterCounts["odd_size"] = m.removeOddSizedTeams(candidates)

	// Ensure that everyone in the match is within their max_rtt of a common server
	candidates, filterCounts["no_matching_servers"] = m.filterWithinMaxRTT(candidates)

	// Create a list of balanced matches with predictions
	predictions := m.buildPredictions(candidates)

	// Sort by matches that have players who have been waiting more than half the Matchmaking timeout
	// This is to prevent players from waiting too long

	if len(predictions) == 0 {
		logger.Warn("No predictions made. Matchmaker cannot run.")
		return nil
	}

	modestr := candidates[0][0].GetProperties()["mode"].(string)
	if modestr == "" {
		logger.Error("Mode not found in entry properties. Matchmaker cannot run.")
		return nil
	}

	switch modestr {

	case evr.ModeCombatPublic.String():
		m.sortCombat(predictions)

	case evr.ModeArenaPublic.String():
		m.sortArena(predictions)

	default:
		logger.WithField("mode", modestr).Error("Unknown mode. Matchmaker cannot run.")
		return nil
	}

	madeMatches := m.assembleUniqueMatches(predictions)

	includedPlayers := make(map[string]struct{}, 0)
	for _, c := range madeMatches {
		for _, e := range c {
			includedPlayers[e.GetPresence().GetUserId()] = struct{}{}
		}
	}

	// Create a list of excluded players
	unmatchedPlayers := make([]string, 0)
	for p := range allPlayers {
		if _, ok := includedPlayers[p]; !ok {
			unmatchedPlayers = append(unmatchedPlayers, p)
		}
	}

	logger.WithFields(map[string]interface{}{
		"mode":                modestr,
		"num_player_total":    len(allPlayers),
		"num_player_included": len(includedPlayers),
		"num_match_options":   len(candidates),
		"num_match_made":      len(madeMatches),
		"made_matches":        madeMatches,
		"filter_counts":       filterCounts,
		"unmatched_players":   unmatchedPlayers,
	}).Info("Skill-based matchmaker completed.")

	return madeMatches
}

func (*skillBasedMatchmaker) PredictDraw(teams []RatedEntryTeam) float64 {
	team1 := make(types.Team, 0, len(teams[0]))
	team2 := make(types.Team, 0, len(teams[1]))
	for _, e := range teams[0] {
		team1 = append(team1, e.Rating)
	}
	for _, e := range teams[1] {
		team2 = append(team2, e.Rating)
	}
	return rating.PredictDraw([]types.Team{team1, team2}, nil)
}

func (m *skillBasedMatchmaker) removeOddSizedTeams(candidates [][]runtime.MatchmakerEntry) ([][]runtime.MatchmakerEntry, int) {
	oddSizedCount := 0
	for i := 0; i < len(candidates); i++ {
		if len(candidates[i])%2 != 0 {
			oddSizedCount++
			candidates = append(candidates[:i], candidates[i+1:]...)
			i--
		}
	}
	return candidates, oddSizedCount
}

func (m *skillBasedMatchmaker) sortCombat(predictions []PredictedMatch) {
	// Sort by size, then by prediction of a draw
	slices.SortStableFunc(predictions, func(a, b PredictedMatch) int {
		if len(a.Entrants()) > len(b.Entrants()) {
			return -1
		} else if len(a.Entrants()) < len(b.Entrants()) {
			return 1
		}

		if a.Draw > b.Draw {
			return -1
		}
		if a.Draw < b.Draw {
			return 1
		}
		return 0

	})
}

// Sort the matches by the players that have been waiting for more than 2/3s of the matchmaking timeout
func (m skillBasedMatchmaker) sortArena(predictions []PredictedMatch) {
	now := time.Now().UTC().Unix()

	// This is to prevent players from waiting too long
	slices.SortStableFunc(predictions, func(a, b PredictedMatch) int {

		// Sort by players that have priority
		aPriority := false
		bPriority := false
		for _, e := range a.Entrants() {
			if ts, ok := e.Entry.GetProperties()["priority_threshold"].(float64); ok && now > int64(ts) {
				aPriority = true
				break
			}
		}
		for _, e := range b.Entrants() {
			if ts, ok := e.Entry.GetProperties()["priority_threshold"].(float64); ok && now > int64(ts) {
				bPriority = true
				break
			}
		}

		if aPriority && !bPriority {
			return -1
		} else if !aPriority && bPriority {
			return 1
		}

		// Sort by rank spread by team

		rankTeamAvgs := make([][]float64, 2)

		for i, o := range []PredictedMatch{a, b} {
			for j, team := range o.Teams() {
				rankTeamAvgs[i] = append(rankTeamAvgs[i], 0)
				for _, player := range team {
					rankTeamAvgs[i][j] += player.Entry.GetProperties()["rank_percentile"].(float64)
				}
				rankTeamAvgs[i][j] /= float64(len(team))
			}
		}

		// get the delta between the two teams
		rankSpreadA := math.Abs(rankTeamAvgs[0][0] - rankTeamAvgs[0][1])
		rankSpreadB := math.Abs(rankTeamAvgs[1][0] - rankTeamAvgs[1][1])

		if math.Abs(rankSpreadA-rankSpreadB) > 0.10 {
			return -1
		} else if math.Abs(rankSpreadA-rankSpreadB) > 0.10 {
			return 1
		}

		// Sort by size of the match
		if len(a.Entrants()) > len(b.Entrants()) {
			return -1
		} else if len(a.Entrants()) < len(b.Entrants()) {
			return 1
		}

		// Sort by prediction of a draw
		if a.Draw > b.Draw {
			return -1
		} else if a.Draw < b.Draw {
			return 1
		}

		return 0

	})
}

func (m *skillBasedMatchmaker) CreateBalancedMatch(groups [][]*RatedEntry, teamSize int) (RatedEntryTeam, RatedEntryTeam) {
	// Split out the solo players

	team1 := make(RatedEntryTeam, 0, teamSize)
	team2 := make(RatedEntryTeam, 0, teamSize)

	// Sort the groups by party size, largest first.

	sort.Slice(groups, func(i, j int) bool {
		// first by party size
		if len(groups[i]) > len(groups[j]) {
			return true
		} else if len(groups[i]) < len(groups[j]) {
			return false
		}

		// Then by strength
		if m.TeamStrength(groups[i]) > m.TeamStrength(groups[j]) {
			return true
		} else if m.TeamStrength(groups[i]) < m.TeamStrength(groups[j]) {
			return false
		}

		return false
	})

	// Organize groups onto teams, balancing by strength
	for _, group := range groups {
		if len(team1)+len(group) <= teamSize && (len(team2)+len(group) > teamSize || m.TeamStrength(team1) <= m.TeamStrength(team2)) {
			team1 = append(team1, group...)
		} else if len(team2)+len(group) <= teamSize {
			team2 = append(team2, group...)
		}
	}

	// Sort the players on the team by their rating
	sort.Slice(team1, func(i, j int) bool {
		return team1[i].Rating.Mu > team1[j].Rating.Mu
	})
	sort.Slice(team2, func(i, j int) bool {
		return team2[i].Rating.Mu > team2[j].Rating.Mu
	})

	// Sort so that team1 (blue) is the stronger team
	if m.TeamStrength(team1) < m.TeamStrength(team2) {
		team1, team2 = team2, team1
	}

	return team1, team2
}

func (m *skillBasedMatchmaker) balanceByTicket(candidate []runtime.MatchmakerEntry) RatedMatch {
	// Group based on ticket

	ticketMap := make(map[string][]*RatedEntry)
	for _, e := range candidate {
		ticketMap[e.GetTicket()] = append(ticketMap[e.GetTicket()], NewRatedEntryFromMatchmakerEntry(e))
	}

	byTicket := make([][]*RatedEntry, 0)
	for _, entries := range ticketMap {
		byTicket = append(byTicket, entries)
	}

	team1, team2 := m.CreateBalancedMatch(byTicket, len(candidate)/2)
	return RatedMatch{team1, team2}
}

// Ensure that everyone in the match is within their max_rtt of a common server
func (m *skillBasedMatchmaker) filterWithinMaxRTT(candidates [][]runtime.MatchmakerEntry) ([][]runtime.MatchmakerEntry, int) {

	var filteredCount int
	for i := 0; i < len(candidates); i++ {

		serverRTTs := make(map[string][]float64)

		for _, entry := range candidates[i] {

			maxRTT := 500.0
			if rtt, ok := entry.GetProperties()["max_rtt"].(float64); ok && rtt > 0 {
				maxRTT = rtt
			}

			for k, v := range entry.GetProperties() {

				if !strings.HasPrefix(k, "rtt") {
					continue
				}

				if v.(float64) > maxRTT {
					// Server is too far away from this player
					continue
				}

				serverRTTs[k] = append(serverRTTs[k], v.(float64))
			}
		}

		for k, rtts := range serverRTTs {
			if len(rtts) != len(candidates[i]) {
				// Server is unreachable to one or more players
				delete(serverRTTs, k)
			}
		}

		if len(serverRTTs) == 0 {
			// No common servers for players
			candidates = append(candidates[:i], candidates[i+1:]...)
			i--
			filteredCount++
		}
	}
	return candidates, filteredCount
}

func (m *skillBasedMatchmaker) buildPredictions(candidates [][]runtime.MatchmakerEntry) []PredictedMatch {
	predictions := make([]PredictedMatch, 0, len(candidates))
	for _, match := range candidates {
		ratedMatch := m.balanceByTicket(match)

		predictions = append(predictions, PredictedMatch{
			Team1: ratedMatch[0],
			Team2: ratedMatch[1],
			Draw:  m.PredictDraw(ratedMatch),
		})
	}
	return predictions
}
func (m *skillBasedMatchmaker) assembleUniqueMatches(ratedMatches []PredictedMatch) [][]runtime.MatchmakerEntry {
	seen := make(map[string]struct{})
	selected := make([][]runtime.MatchmakerEntry, 0, len(ratedMatches))

OuterLoop:
	for _, ratedMatch := range ratedMatches {
		// The players are ordered by their team
		match := make([]runtime.MatchmakerEntry, 0, 8)

		// Ensure no player is in more than one match
		for _, e := range ratedMatch.Entrants() {
			sessionID := e.Entry.GetPresence().GetSessionId()

			// Skip match with players already in a match
			if _, ok := seen[sessionID]; ok {
				continue OuterLoop
			}
			seen[sessionID] = struct{}{}
			match = append(match, e.Entry)
		}

		selected = append(selected, match)
	}
	return selected
}

func GetRatingByUserID(ctx context.Context, db *sql.DB, userID string, defaultFallback bool) (r types.Rating, err error) {
	// Look for an existing account.
	query := "SELECT value->>'rating' FROM storage WHERE user_id = $1 AND collection = $2 and key = $3"
	var ratingJSON string
	var found = true
	if err = db.QueryRowContext(ctx, query, userID, GameProfileStorageCollection, GameProfileStorageKey).Scan(&ratingJSON); err != nil {
		if err == sql.ErrNoRows {
			found = false
		} else {
			return r, status.Error(codes.Internal, "error finding rating by user ID")
		}
	}
	if !found {
		if defaultFallback {
			return rating.NewWithOptions(&types.OpenSkillOptions{
				Mu:    ptr.Float64(25.0),
				Sigma: ptr.Float64(8.333),
			}), nil
		} else {
			return r, errors.New("rating not found")
		}
	}
	if err = json.Unmarshal([]byte(ratingJSON), &r); err != nil {
		return r, errors.New("error unmarshalling rating")
	}
	return r, nil
}
