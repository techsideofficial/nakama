package server

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama/v3/server/evr"
)

func CalculateSmoothedPlayerRankPercentile(ctx context.Context, logger *zap.Logger, nk runtime.NakamaModule, userID string, mode evr.Symbol, globalSettings, userSettings *MatchmakingSettings) (float64, error) {

	if mode != evr.ModeSocialPublic {
		mode = evr.ModeArenaPublic
	}

	globalSettingsVersion, _, err := LoadMatchmakingSettingsWithVersion(ctx, nk, userID)
	if err != nil {
		return 0.0, fmt.Errorf("failed to load matchmaking settings: %w", err)
	}

	if userSettings.StaticBaseRankPercentile > 0 {
		return userSettings.StaticBaseRankPercentile, nil
	}

	if globalSettings.StaticBaseRankPercentile > 0 {
		return globalSettings.StaticBaseRankPercentile, nil
	}

	if userSettings.PreviousRankPercentile > 0 || globalSettingsVersion == userSettings.GlobalSettingsVersion {
		return userSettings.PreviousRankPercentile, nil
	}

	defaultRankPercentile := globalSettings.RankPercentileDefault
	if userSettings.RankPercentileDefault != 0 {
		defaultRankPercentile = userSettings.RankPercentileDefault
	}

	activeSchedule := "daily"
	if globalSettings.RankResetSchedule != "" {
		activeSchedule = globalSettings.RankResetSchedule
	}

	if userSettings.RankResetSchedule != "" {
		activeSchedule = userSettings.RankResetSchedule
	}

	dampingSchedule := "weekly"
	if globalSettings.RankResetScheduleDamping != "" {
		dampingSchedule = globalSettings.RankResetScheduleDamping
	}

	if userSettings.RankResetScheduleDamping != "" {
		dampingSchedule = userSettings.RankResetScheduleDamping
	}

	dampingFactor := globalSettings.RankPercentileDampingFactor
	if userSettings.RankPercentileDampingFactor != 0 {
		dampingFactor = userSettings.RankPercentileDampingFactor
	}

	var boardWeights map[string]float64

	if len(globalSettings.RankBoardWeights) > 0 {
		boardWeights = globalSettings.RankBoardWeights[mode.String()]
	}

	if len(userSettings.RankBoardWeights) > 0 {
		boardWeights = userSettings.RankBoardWeights[mode.String()]
	}

	if len(boardWeights) == 0 {
		return defaultRankPercentile, nil
	}

	dampingPercentile, err := RecalculatePlayerRankPercentile(ctx, logger, nk, userID, mode, dampingSchedule, defaultRankPercentile, boardWeights)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get damping percentile: %w", err)
	}

	activePercentile, err := RecalculatePlayerRankPercentile(ctx, logger, nk, userID, mode, activeSchedule, dampingPercentile, boardWeights)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get active percentile: %w", err)
	}

	percentile := activePercentile + (dampingPercentile-activePercentile)*dampingFactor

	userSettings.PreviousRankPercentile = percentile
	userSettings.GlobalSettingsVersion = globalSettingsVersion
	if _, err := SaveToStorage(ctx, nk, userID, userSettings); err != nil {
		logger.Warn("Failed to save user settings", zap.Error(err))
	}
	return percentile, nil
}

func RecalculatePlayerRankPercentile(ctx context.Context, logger *zap.Logger, nk runtime.NakamaModule, userID string, mode evr.Symbol, periodicity string, defaultRankPercentile float64, boardNameWeights map[string]float64) (float64, error) {

	percentiles := make([]float64, 0, len(boardNameWeights))
	weights := make([]float64, 0, len(boardNameWeights))

	for boardName, weight := range boardNameWeights {

		boardID := fmt.Sprintf("%s:%s:%s", mode.String(), boardName, periodicity)

		records, _, _, _, err := nk.LeaderboardRecordsList(ctx, boardID, []string{userID}, 10000, "", 0)
		if err != nil {
			continue
		}

		if len(records) == 0 {
			percentiles = append(percentiles, defaultRankPercentile)
			weights = append(weights, weight)
			continue
		}

		// Find the user's rank.
		var rank int64 = -1
		for _, record := range records {
			if record.OwnerId == userID {
				rank = record.Rank
				break
			}
		}
		if rank == -1 {
			continue
		}

		percentile := float64(rank) / float64(len(records))

		weights = append(weights, weight)
		percentiles = append(percentiles, percentile)

	}

	percentile := 0.0

	if len(percentiles) == 0 {
		return defaultRankPercentile, nil
	}

	for _, p := range percentiles {
		percentile += p
	}
	percentile /= float64(len(percentiles))

	percentile, err := normalizedWeightedAverage(percentiles, weights)
	if err != nil {
		return defaultRankPercentile, err
	}

	return percentile, nil
}

func normalizedWeightedAverage(values, weights []float64) (float64, error) {
	if len(values) != len(weights) {
		return 0, fmt.Errorf("values and weights must have the same length")
	}

	// Normalize weights to sum to 1
	var weightSum float64
	for _, w := range weights {
		weightSum += w
	}

	if weightSum == 0 {
		return 0, fmt.Errorf("sum of weights must not be zero")
	}

	var sum float64
	for i := range values {
		normalizedWeight := weights[i] / weightSum
		sum += values[i] * normalizedWeight
	}

	return sum, nil
}
