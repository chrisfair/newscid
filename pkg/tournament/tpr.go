package tournament

import "math"

// fideTableDp maps percentage p (from 0.50 to 1.00 in steps of 0.01) to rating difference dp.
// Table 8.1a from FIDE Handbook (The working of the FIDE Rating System).
var fideTableDp = [...]int{
	0,   // 0.50
	7,   // 0.51
	14,  // 0.52
	21,  // 0.53
	29,  // 0.54
	36,  // 0.55
	43,  // 0.56
	50,  // 0.57
	57,  // 0.58
	65,  // 0.59
	72,  // 0.60
	80,  // 0.61
	87,  // 0.62
	95,  // 0.63
	102, // 0.64
	110, // 0.65
	117, // 0.66
	125, // 0.67
	133, // 0.68
	141, // 0.69
	149, // 0.70
	158, // 0.71
	166, // 0.72
	175, // 0.73
	184, // 0.74
	193, // 0.75
	202, // 0.76
	211, // 0.77
	220, // 0.78
	230, // 0.79
	240, // 0.80
	251, // 0.81
	262, // 0.82
	273, // 0.83
	284, // 0.84
	296, // 0.85
	309, // 0.86
	322, // 0.87
	336, // 0.88
	351, // 0.89
	366, // 0.90
	383, // 0.91
	401, // 0.92
	422, // 0.93
	444, // 0.94
	470, // 0.95
	501, // 0.96
	538, // 0.97
	589, // 0.98
	677, // 0.99
	800, // 1.00
}

// GetFideRatingDifference returns the rating difference (dp) for a given score fraction p (0.0 to 1.0).
func GetFideRatingDifference(p float64) int {
	if p >= 1.0 {
		return 800
	}
	if p <= 0.0 {
		return -800
	}

	isNegative := false
	if p < 0.50 {
		isNegative = true
		p = 1.0 - p
	}

	idx := int(math.Round((p - 0.50) * 100.0))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(fideTableDp) {
		idx = len(fideTableDp) - 1
	}

	dp := fideTableDp[idx]
	if isNegative {
		return -dp
	}
	return dp
}

// CalculatePerformanceRating computes the Tournament Performance Rating (TPR)
// given the total score, number of games, and the average Elo of the opponents faced.
func CalculatePerformanceRating(score float64, played int, avgOpponentElo int) int {
	if played <= 0 {
		return 0
	}
	if avgOpponentElo <= 0 {
		return 0
	}
	if score <= 0 {
		tpr := avgOpponentElo - 800
		if tpr < 0 {
			return 0
		}
		return tpr
	}
	if score >= float64(played) {
		return avgOpponentElo + 800
	}

	p := score / float64(played)
	dp := GetFideRatingDifference(p)
	tpr := avgOpponentElo + dp
	if tpr < 0 {
		return 0
	}
	return tpr
}

// CalculateFideCategory determines the official FIDE tournament category based on the average Elo rating of all participants.
// Category 1 starts at 2251-2275, Category 2 at 2276-2300, etc.
func CalculateFideCategory(avgElo int) int {
	if avgElo < 2251 {
		return 0
	}
	return (avgElo-2251)/25 + 1
}
