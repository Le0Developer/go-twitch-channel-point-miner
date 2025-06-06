package miner

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

func (miner *Miner) OnPredictionUpdate(message WebsocketMessage) {
	var event predictionEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		fmt.Println("Failed to unmarshal prediction event", err)
		return
	}

	if event.Data.Event.Status == PredictionStatusResolved {
		delete(miner.Predictions, event.Data.Event.ID)
		fmt.Println("Prediction resolved", event.Data.Event.ID)
		if event.Data.Event.WinningOutcomeID != nil {
			winner := ""
			for _, outcome := range event.Data.Event.Outcomes {
				if outcome.ID == *event.Data.Event.WinningOutcomeID {
					winner = outcome.Title
					break
				}
			}

			if winner != "" {
				data, ok := miner.Persistent.PredictionResults[event.Data.Event.PredictionID()]
				if !ok {
					data = map[string]int{}
					miner.Persistent.PredictionResults[event.Data.Event.PredictionID()] = data
				}
				data[winner]++

				if err := miner.Persistent.Save(miner.Options); err != nil {
					fmt.Println("Failed to save prediction results", err)
				}
			}
		}

		return
	}

	pred, ok := miner.Predictions[event.Data.Event.ID]
	if !ok {
		pred = &Prediction{
			Event: &event.Data.Event,
			Miner: miner,
		}
		miner.Predictions[event.Data.Event.ID] = pred
	}

	pred.Event = &event.Data.Event

	if pred.Event.Status == PredictionStatusActive && !pred.Bet && !pred.BetPending {
		pred.BetPending = true
		go pred.DelayedBet()
	}
}

type Prediction struct {
	Event      *predictionModel
	Miner      *Miner
	Bet        bool
	BetPending bool
}

func (p *Prediction) DelayedBet() {
	// sleep until 5s before the end of the prediction
	// this is to avoid the prediction being locked before we can place a bet
	time.Sleep(time.Until(p.Event.CreatedAt.Add(time.Duration(p.Event.PredictionWindowSeconds-5) * time.Second)))
	if p.Event.Status != PredictionStatusActive {
		// welp, prediction is no longer active
		// can be caused by early resolve
		return
	}

	p.SmartBet()
}

type PredictionStrategy = string

const (
	PredictionStrategyRandom               PredictionStrategy = "RANDOM"
	PredictionStrategyMostPoints           PredictionStrategy = "MOST_POINTS"
	PredictionStrategyMostIndividuals      PredictionStrategy = "MOST_INDIVIDUALS"
	PredictionStrategyMostIndividualPoints PredictionStrategy = "MOST_INDIVIDUAL_POINTS"
	PredictionStrategyCautious             PredictionStrategy = "CAUTIOUS"
)

var (
	insymGhostGambling = regexp.MustCompile(`Will it be a [^ ]+ or a Mimic?`)
)

func (p *Prediction) SmartBet() {
	var bet predictionOutcome

	totalPointsBet := 0
	for _, outcome := range p.Event.Outcomes {
		totalPointsBet += outcome.TotalPoints
	}

	betAmount := p.Miner.Options.PredictionsMaxBet

	switch p.Miner.Options.PredictionsStrategy {
	case PredictionStrategyRandom:
		bet = p.Event.Outcomes[rand.Intn(len(p.Event.Outcomes))]
	case PredictionStrategyMostPoints:
		mostPoints := 0
		for _, outcome := range p.Event.Outcomes {
			if outcome.TotalPoints > mostPoints {
				bet = outcome
				mostPoints = outcome.TotalPoints
			}
		}
	case PredictionStrategyMostIndividuals:
		mostIndividuals := 0
		for _, outcome := range p.Event.Outcomes {
			if outcome.TotalUsers > mostIndividuals {
				bet = outcome
				mostIndividuals = outcome.TotalUsers
			}
		}
	case PredictionStrategyMostIndividualPoints:
		mostIndividualPoints := 0
		for _, outcome := range p.Event.Outcomes {
			for _, better := range outcome.TopPredictors {
				if better.Points > mostIndividualPoints {
					bet = outcome
					mostIndividualPoints = better.Points
				}
			}
		}
	case PredictionStrategyCautious:
		data, ok := p.Miner.Persistent.PredictionResults[p.Event.PredictionID()]
		if p.Event.ChannelID == "75738685" && insymGhostGambling.Match([]byte(p.Event.Title)) { // insym
			// odds are 1:12
			data = map[string]int{
				"Yes": 1,
				"No":  12,
			}
			ok = true
		}

		if ok {
			total := 0
			for _, outcome := range data {
				total += outcome
			}

			if total > p.Miner.Options.PredictionsDataPoints {
				// we have enough data to make a bet
				// we check which option has the highest Return on Investment
				// eg if an option wins 1/3 of the time but only has 25% odds, we bet on it
				// since it's a 1/3 chance to win 4x our bet
				bestROI := 0.0
				pointDiff := 0
				for _, outcome := range p.Event.Outcomes {
					timesWon := data[outcome.Title]
					expectedWinrate := float64(timesWon) / float64(total)
					currentWinrate := float64(outcome.TotalPoints) / float64(totalPointsBet)
					expectedPointsBet := int(float64(totalPointsBet) * expectedWinrate)
					roi := expectedWinrate / currentWinrate
					fmt.Printf("ExpectedWinrate: %f (%d), CurrentWinrate: %f (%d), ROI: %f\n", expectedWinrate, expectedPointsBet, currentWinrate, outcome.TotalPoints, roi)
					if roi > bestROI {
						bestROI = roi
						bet = outcome
						pointDiff = expectedPointsBet - outcome.TotalPoints
					}
				}

				// cap bets so we dont bet more than the expected winrate
				if pointDiff < betAmount {
					betAmount = pointDiff
				}
			}
		}
	}

	if bet.ID == "" {
		return
	}

	if betAmount > totalPointsBet*p.Miner.Options.PredictionsMaxRatio {
		betAmount = totalPointsBet * p.Miner.Options.PredictionsMaxRatio
	}

	if p.Miner.Options.PredictionsStealth {
		// dont bet more than the highest individual bet
		// the highest individual bet is displayed publicly to everyone
		highestIndividualBet := 0
		for _, better := range bet.TopPredictors {
			if better.Points > highestIndividualBet {
				highestIndividualBet = better.Points
			}
		}

		if betAmount > highestIndividualBet {
			betAmount = highestIndividualBet
		}
	}

	streamer := p.Miner.GetStreamerByID(p.Event.ChannelID)

	for _, user := range p.Miner.Users {
		userBet := betAmount
		if streamer.Points[user] < p.Miner.Options.PredictionsMinPoints {
			continue
		}
		if userBet > streamer.Points[user] {
			userBet = streamer.Points[user]
		}
		if userBet < 10 {
			continue
		}

		fmt.Printf("Betting %d points on %s (%s) for %s\n", userBet, bet.Title, bet.ID, p.Event.Title)
		p.Miner.Alert(fmt.Sprintf("Betting %d points on %s (%s) for %s\n", userBet, bet.Title, bet.ID, p.Event.Title))
		if err := user.GraphQL.MakePrediction(p.Event.ID, bet.ID, userBet); err != nil {
			fmt.Println("Failed to place bet", err)
			continue
		}
	}

	p.Bet = true
}

func (p *predictionModel) PredictionID() string {
	// we construct a "prediction id" based on the prediction name and outcome options
	// this is so we can leverage previous bet results for future bets
	parts := []string{
		p.ChannelID,
		p.Title,
	}
	for _, outcome := range p.Outcomes {
		parts = append(parts, outcome.Title)
	}
	return strings.Join(parts, ";")
}

type PredictionStatus = string

const (
	PredictionStatusActive         PredictionStatus = "ACTIVE"
	PredictionStatusLocked         PredictionStatus = "LOCKED"
	PredictionStatusResolvePending PredictionStatus = "RESOLVE_PENDING"
	PredictionStatusResolved       PredictionStatus = "RESOLVED"
)

type predictionEvent struct {
	Data struct {
		Event predictionModel `json:"event"`
	} `json:"data"`
}

type predictionModel struct {
	Status                  PredictionStatus    `json:"status"`
	WinningOutcomeID        *string             `json:"winning_outcome_id"`
	ChannelID               string              `json:"channel_id"`
	CreatedAt               time.Time           `json:"created_at"`
	EndedAt                 time.Time           `json:"ended_at"`
	LockedAt                time.Time           `json:"locked_at"`
	PredictionWindowSeconds int                 `json:"prediction_window_seconds"`
	Title                   string              `json:"title"`
	ID                      string              `json:"id"`
	Outcomes                []predictionOutcome `json:"outcomes"`
}

func (p *predictionModel) CanBet() bool {
	return p.Status == PredictionStatusActive && p.LockedAt.IsZero() && time.Since(p.CreatedAt) < time.Duration(p.PredictionWindowSeconds)*time.Second
}

type predictionOutcome struct {
	ID            string `json:"id"`
	Color         string `json:"color"`
	Title         string `json:"title"`
	TotalPoints   int    `json:"total_points"`
	TotalUsers    int    `json:"total_users"`
	TopPredictors []struct {
		Points int `json:"points"`
	} `json:"top_predictors"`
}
