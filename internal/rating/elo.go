package rating

import "math"

const kFactor = 32

type Result int

const (
	WhiteWon Result = iota
	BlackWon
	Draw
)

func Apply(whiteRating, blackRating int, result Result) (newWhiteRating, newBlackRating int) {
	whiteDelta := Delta(whiteRating, blackRating, scoreFor(result, true))
	blackDelta := Delta(blackRating, whiteRating, scoreFor(result, false))

	newWhiteRating = whiteRating + whiteDelta
	newBlackRating = blackRating + blackDelta
	return
}

func ApplyAbandonment(winnerRating, loserRating int) (newWinnerRating, newLoserRating int) {
	winnerDelta := Delta(winnerRating, loserRating, 1)
	loserDelta := Delta(loserRating, winnerRating, 0)

	newWinnerRating = winnerRating + round(float64(winnerDelta)/2)
	newLoserRating = loserRating + loserDelta
	return
}

func Delta(playerRating, opponentRating int, score float64) int {
	return round(kFactor * (score - expectedScore(playerRating, opponentRating)))
}

func expectedScore(a, b int) float64 {
	return 1 / (1 + math.Pow(10, float64(b-a)/400))
}

func scoreFor(result Result, isWhite bool) float64 {
	switch result {
	case WhiteWon:
		if isWhite {
			return 1
		}
		return 0
	case BlackWon:
		if isWhite {
			return 0
		}
		return 1
	default:
		return 0.5
	}
}

func round(f float64) int {
	return int(math.Round(f))
}
