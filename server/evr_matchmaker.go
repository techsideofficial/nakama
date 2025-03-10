package server

import (
	"context"
	"database/sql"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/intinig/go-openskill/rating"
	"github.com/intinig/go-openskill/types"
	"github.com/samber/lo"
	"go.uber.org/atomic"
)

const (
	MaximumRankDelta  = 0.10
	RTTPropertyPrefix = "rtt_"
)

type SkillBasedMatchmaker struct {
	latestCandidates *atomic.Value // [][]runtime.MatchmakerEntry
	latestMatches    *atomic.Value // [][]runtime.MatchmakerEntry
}

func (s *SkillBasedMatchmaker) StoreLatestResult(candidates, madeMatches [][]runtime.MatchmakerEntry) {

	s.latestCandidates.Store(candidates)
	s.latestMatches.Store(madeMatches)

}

func (s *SkillBasedMatchmaker) GetLatestResult() (candidates, madeMatches [][]runtime.MatchmakerEntry) {
	var ok bool
	candidates, ok = s.latestCandidates.Load().([][]runtime.MatchmakerEntry)
	if !ok {
		return
	}
	madeMatches, ok = s.latestMatches.Load().([][]runtime.MatchmakerEntry)
	if !ok {
		return candidates, nil
	}
	return
}

func NewSkillBasedMatchmaker() *SkillBasedMatchmaker {
	sbmm := SkillBasedMatchmaker{
		latestCandidates: &atomic.Value{},
		latestMatches:    &atomic.Value{},
	}

	sbmm.latestCandidates.Store([][]runtime.MatchmakerEntry{})
	sbmm.latestMatches.Store([][]runtime.MatchmakerEntry{})
	return &sbmm
}

// Function to be used as a matchmaker function in Nakama (RegisterMatchmakerOverride)
func (m *SkillBasedMatchmaker) EvrMatchmakerFn(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, candidates [][]runtime.MatchmakerEntry) [][]runtime.MatchmakerEntry {

	startTime := time.Now()
	defer func() {
		nk.MetricsGaugeSet("matchmaker_evr_candidate_count", nil, float64(len(candidates)))

		// Divide the time by the number of candidates
		nk.MetricsTimerRecord("matchmaker_evr_per_candidate", nil, time.Since(startTime)/time.Duration(len(candidates)))
	}()

	if len(candidates) == 0 || len(candidates[0]) == 0 {
		logger.Error("No candidates found. Matchmaker cannot run.")
		return nil
	}

	groupID, ok := candidates[0][0].GetProperties()["group_id"].(string)
	if !ok || groupID == "" {
		logger.Error("Group ID not found in entry properties.")
		return nil
	}

	modestr, ok := candidates[0][0].GetProperties()["game_mode"].(string)
	if !ok || modestr == "" {
		logger.Error("Mode not found in entry properties. Matchmaker cannot run.")
		return nil
	}

	var (
		matches        [][]runtime.MatchmakerEntry
		filterCounts   map[string]int
		originalCount  = len(candidates)
		globalSettings = ServiceSettings()
	)

	candidates, matches, filterCounts = m.processPotentialMatches(candidates, globalSettings)

	// Extract all players from the candidates
	playerSet := make(map[string]struct{}, 0)
	ticketSet := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		for _, e := range c {
			ticketSet[e.GetTicket()] = struct{}{}
			playerSet[e.GetPresence().GetUserId()] = struct{}{}
		}
	}

	// Extract all players from the matches
	matchedPlayerSet := make(map[string]struct{}, 0)
	for _, c := range matches {
		for _, e := range c {
			matchedPlayerSet[e.GetPresence().GetUserId()] = struct{}{}
		}
	}

	matchedPlayers := lo.Keys(matchedPlayerSet)

	// Create a list of excluded players
	unmatchedPlayers := lo.FilterMap(lo.Keys(playerSet), func(p string, _ int) (string, bool) {
		_, ok := matchedPlayerSet[p]
		return p, !ok
	})

	logger.WithFields(map[string]interface{}{
		"mode":                 modestr,
		"num_player_total":     len(playerSet),
		"num_tickets":          len(ticketSet),
		"num_players_matched":  len(matchedPlayers),
		"num_match_candidates": originalCount,
		"num_matches_made":     len(matches),
		"filter_counts":        filterCounts,
		"matched_players":      matchedPlayerSet,
		"unmatched_players":    unmatchedPlayers,
	}).Info("Skill-based matchmaker completed.")

	if candidates != nil && matches != nil && len(candidates) > 0 {
		m.StoreLatestResult(candidates, matches)
	}

	return matches
}

func (m *SkillBasedMatchmaker) processPotentialMatches(candidates [][]runtime.MatchmakerEntry, globalSettings *GlobalSettingsData) ([][]runtime.MatchmakerEntry, [][]runtime.MatchmakerEntry, map[string]int) {

	// Write the candidates to a json filed called /tmp/candidates.json
	filterCounts := make(map[string]int)

	// Filter out duplicates
	filterCounts["duplicates"] = m.filterDuplicates(candidates)

	// Filter out odd-sized teams
	filterCounts["odd_sized_teams"] = m.filterOddSizedTeams(candidates)

	// Filter out players who are too far away from each other
	filterCounts["max_rtt"] = m.filterWithinMaxRTT(candidates)

	// Create a list of balanced matches with predictions
	predictions := m.predictOutcomes(candidates, globalSettings.Matchmaking.RankPercentile.Default)

	// Sort the predictions
	m.sortPredictions(predictions, globalSettings.Matchmaking.RankPercentile.MaxDelta, false)

	madeMatches := m.assembleUniqueMatches(predictions)

	return candidates, madeMatches, filterCounts
}

func (m *SkillBasedMatchmaker) filterDuplicates(candidates [][]runtime.MatchmakerEntry) int {
	seen := make(map[string]struct{}) // Map to track seen candidate combinations
	count := 0

	for i, candidate := range candidates {
		if candidate == nil {
			continue
		}

		// Create a key for the candidate based on the ticket IDs
		ticketIDs := make([]string, 0, len(candidate))
		for _, e := range candidate {
			ticketIDs = append(ticketIDs, e.GetTicket())
		}
		sort.Strings(ticketIDs)
		key := strings.Join(ticketIDs, "")

		// Check if the candidate has been seen before
		if _, ok := seen[key]; ok {
			count++
			candidates[i] = nil
			continue
		}

		// Mark the candidate as seen
		seen[key] = struct{}{}
	}

	return count // Return the count of duplicates and the filtered candidates
}

func (m *SkillBasedMatchmaker) filterOddSizedTeams(candidates [][]runtime.MatchmakerEntry) int {
	oddSizedCount := 0
	for i := 0; i < len(candidates); i++ {
		if candidates[i] == nil {
			continue
		}
		if len(candidates[i])%2 != 0 {
			oddSizedCount++
			candidates[i] = nil
			continue
		}

	}
	return oddSizedCount
}

// Ensure that everyone in the match is within their max_rtt of a common server
func (m *SkillBasedMatchmaker) filterWithinMaxRTT(candidates [][]runtime.MatchmakerEntry) int {

	var filteredCount int
OuterLoop:
	for i, candidate := range candidates {

		if candidate == nil {
			continue
		}

		var ok bool
		var maxRTT float64

		reachablePlayers := make(map[string]int)

		for _, entry := range candidate {

			if maxRTT, ok = entry.GetProperties()["max_rtt"].(float64); !ok || maxRTT <= 0 {
				maxRTT = 500.0
			}

			for k, v := range entry.GetProperties() {

				if !strings.HasPrefix(k, RTTPropertyPrefix) {
					continue
				}

				if v.(float64) > maxRTT {
					// Server is too far away from this player
					continue
				}

				reachablePlayers[k]++

				if reachablePlayers[k] == len(candidate) {
					continue OuterLoop
				}
			}
		}
		// Players have no common server
		candidates[i] = nil
		filteredCount++
	}

	return filteredCount
}

func (*SkillBasedMatchmaker) teamStrength(team RatedEntryTeam) float64 {
	s := 0.0
	for _, p := range team {
		s += p.Rating.Mu
	}
	return s
}

func (*SkillBasedMatchmaker) predictDraw(teams []RatedEntryTeam) float64 {
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

func (m *SkillBasedMatchmaker) createBalancedMatch(groups [][]*RatedEntry, teamSize int) (RatedEntryTeam, RatedEntryTeam) {
	// Split out the solo players

	team1 := make(RatedEntryTeam, 0, teamSize)
	team2 := make(RatedEntryTeam, 0, teamSize)

	// Sort the groups by party size, largest first.

	slices.SortStableFunc(groups, func(a, b []*RatedEntry) int {
		if len(a) > len(b) {
			return -1
		}

		if len(a) < len(b) {
			return 1
		}

		if m.teamStrength(a) > m.teamStrength(b) {
			return -1
		}

		if m.teamStrength(a) < m.teamStrength(b) {
			return 1
		}

		return 0
	})

	// Organize groups onto teams, balancing by strength
	for _, group := range groups {
		if len(team1)+len(group) <= teamSize && (len(team2)+len(group) > teamSize || m.teamStrength(team1) <= m.teamStrength(team2)) {
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
	if m.teamStrength(team1) < m.teamStrength(team2) {
		team1, team2 = team2, team1
	}

	return team1, team2
}

func (m *SkillBasedMatchmaker) balanceByTicket(candidate []runtime.MatchmakerEntry) RatedMatch {
	// Group based on ticket

	ticketMap := make(map[string][]*RatedEntry)
	for _, e := range candidate {
		ticketMap[e.GetTicket()] = append(ticketMap[e.GetTicket()], NewRatedEntryFromMatchmakerEntry(e))
	}

	byTicket := make([][]*RatedEntry, 0)
	for _, entries := range ticketMap {
		byTicket = append(byTicket, entries)
	}

	team1, team2 := m.createBalancedMatch(byTicket, len(candidate)/2)
	return RatedMatch{team1, team2}
}

func (m *SkillBasedMatchmaker) averageRankPercentile(team RatedEntryTeam, defaultRankPercentile float64) float64 {
	var sum float64
	for _, player := range team {
		if rankPercentile, ok := player.Entry.Properties["rank_percentile"].(float64); ok {
			sum += rankPercentile
		} else {
			sum += defaultRankPercentile
		}
	}
	return sum / float64(len(team))
}

func (m *SkillBasedMatchmaker) predictOutcomes(candidates [][]runtime.MatchmakerEntry, defaultRankPercentile float64) []PredictedMatch {
	predictions := make([]PredictedMatch, 0, len(candidates))

	for _, match := range candidates {
		if match == nil {
			continue
		}
		ratedMatch := m.balanceByTicket(match)

		var (
			percentileA     = m.averageRankPercentile(ratedMatch[0], defaultRankPercentile)
			percentileB     = m.averageRankPercentile(ratedMatch[1], defaultRankPercentile)
			percentileDelta = math.Abs(percentileA - percentileB)
			divisions       = make(map[string]struct{}, 1)
			priorityExpiry  = time.Now().UTC().Format(time.RFC3339)
		)

		for _, team := range ratedMatch {
			for _, player := range team {
				if p, ok := player.Entry.GetProperties()["division"].(string); !ok || p == "" {
					// Add a random division to the player (the session id)
					divisions[player.Entry.Presence.SessionId] = struct{}{}
				} else {
					divisions[p] = struct{}{}
				}
				if th, ok := player.Entry.GetProperties()["priority_threshold"].(string); ok {
					if th < priorityExpiry {
						priorityExpiry = th
					}
				}
			}
		}

		predictions = append(predictions, PredictedMatch{
			TeamA:              ratedMatch[0],
			TeamB:              ratedMatch[1],
			Draw:               m.predictDraw(ratedMatch),
			Size:               len(match),
			RankDelta:          percentileDelta,
			AvgRankPercentileA: percentileA,
			AvgRankPercentileB: percentileB,
			DivisionCount:      len(divisions),
			PriorityExpiry:     priorityExpiry,
		})
	}

	return predictions
}

func (m *SkillBasedMatchmaker) assembleUniqueMatches(ratedMatches []PredictedMatch) [][]runtime.MatchmakerEntry {

	matches := make([][]runtime.MatchmakerEntry, 0, len(ratedMatches))

	matchedPlayers := make(map[string]struct{}, 0)

OuterLoop:
	for _, r := range ratedMatches {

		match := make([]runtime.MatchmakerEntry, 0, 8)

		for _, e := range r.Entrants() {
			if _, ok := matchedPlayers[e.Entry.Presence.SessionId]; ok {
				continue OuterLoop
			}
		}

		for _, e := range r.Entrants() {
			match = append(match, e.Entry)
			matchedPlayers[e.Entry.Presence.SessionId] = struct{}{}
		}

		matches = append(matches, match)
	}

	return matches
}
