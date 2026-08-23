package rating

import "testing"

func TestApply(t *testing.T) {
	cases := []struct {
		name         string
		whiteRating  int
		blackRating  int
		result       Result
		wantWhiteNew int
		wantBlackNew int
	}{
		{"equal ratings, white wins", 1200, 1200, WhiteWon, 1216, 1184},
		{"equal ratings, black wins", 1200, 1200, BlackWon, 1184, 1216},
		{"equal ratings, draw", 1200, 1200, Draw, 1200, 1200},
		{"higher rated white wins", 1600, 1400, WhiteWon, 1608, 1392},
		{"lower rated white wins (upset)", 1400, 1600, WhiteWon, 1424, 1576},
		{"equal high ratings, draw", 2000, 2000, Draw, 2000, 2000},
		{"underdog black wins", 1000, 1200, BlackWon, 992, 1208},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotWhite, gotBlack := Apply(tc.whiteRating, tc.blackRating, tc.result)

			if gotWhite != tc.wantWhiteNew {
				t.Errorf("white rating = %d, want %d", gotWhite, tc.wantWhiteNew)
			}
			if gotBlack != tc.wantBlackNew {
				t.Errorf("black rating = %d, want %d", gotBlack, tc.wantBlackNew)
			}
		})
	}
}

func TestApplyAbandonment(t *testing.T) {
	cases := []struct {
		name          string
		winnerRating  int
		loserRating   int
		wantWinnerNew int
		wantLoserNew  int
	}{
		{"equal ratings", 1200, 1200, 1208, 1184},
		{"favorite wins by abandonment", 1800, 1200, 1801, 1199},
		{"underdog wins by abandonment", 1200, 1800, 1216, 1769},
		{"underdog wins by abandonment, low ratings", 1000, 1200, 1012, 1176},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotWinner, gotLoser := ApplyAbandonment(tc.winnerRating, tc.loserRating)

			if gotWinner != tc.wantWinnerNew {
				t.Errorf("winner rating = %d, want %d", gotWinner, tc.wantWinnerNew)
			}
			if gotLoser != tc.wantLoserNew {
				t.Errorf("loser rating = %d, want %d", gotLoser, tc.wantLoserNew)
			}
		})
	}
}

func TestApplyAbandonment_WinnerGainsHalfOfNormalWin(t *testing.T) {
	normalWinner, _ := Apply(1200, 1200, WhiteWon)
	normalGain := normalWinner - 1200

	abandonmentWinner, _ := ApplyAbandonment(1200, 1200)
	abandonmentGain := abandonmentWinner - 1200

	if abandonmentGain*2 != normalGain {
		t.Fatalf("abandonment gain (%d) should be exactly half of a normal win's gain (%d)", abandonmentGain, normalGain)
	}
}

func TestApplyAbandonment_LoserLosesFullAmount(t *testing.T) {
	_, normalLoser := Apply(1200, 1200, WhiteWon)
	normalLoss := 1200 - normalLoser

	_, abandonmentLoser := ApplyAbandonment(1200, 1200)
	abandonmentLoss := 1200 - abandonmentLoser

	if abandonmentLoss != normalLoss {
		t.Fatalf("loser should still lose the full normal amount (%d), got %d", normalLoss, abandonmentLoss)
	}
}

func TestApply_UpsetSwingsMoreThanExpectedResult(t *testing.T) {
	favoriteWins, _ := Apply(1800, 1200, WhiteWon)
	expectedResultSwing := favoriteWins - 1800

	upsetWins, _ := Apply(1800, 1200, BlackWon)
	upsetSwing := 1800 - upsetWins

	if expectedResultSwing <= 0 {
		t.Fatalf("favorite should gain rating on an expected win, got delta %d", expectedResultSwing)
	}
	if upsetSwing <= expectedResultSwing {
		t.Fatalf("an upset loss (%d) should swing the favorite's rating more than an expected win (%d)", upsetSwing, expectedResultSwing)
	}
}
