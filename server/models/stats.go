package models

type PlayerStats struct {
	ID             int     `json:"id" db:"id"`
	PlayerID       int     `json:"player_id" db:"player_id"`
	TeamID         *int    `json:"team_id" db:"team_id"`
	Year           int     `json:"year" db:"year"`
	GamesPlayed    int     `json:"games_played" db:"games_played"`
	AtBats         int     `json:"at_bats" db:"at_bats"`
	Hits           int     `json:"hits" db:"hits"`
	Doubles        int     `json:"doubles" db:"doubles"`
	Triples        int     `json:"triples" db:"triples"`
	HomeRuns       int     `json:"home_runs" db:"home_runs"`
	RBIs           int     `json:"rbis" db:"rbis"`
	StolenBases    int     `json:"stolen_bases" db:"stolen_bases"`
	BattingAverage float64 `json:"batting_average" db:"batting_average"`
	Wins           int     `json:"wins" db:"wins"`
	Losses         int     `json:"losses" db:"losses"`
	Saves          int     `json:"saves" db:"saves"`
	InningsPitched float64 `json:"innings_pitched" db:"innings_pitched"`
	Strikeouts     int     `json:"strikeouts" db:"strikeouts"`
	ERA            float64 `json:"era" db:"era"`
}

type Award struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	Category    string `json:"category" db:"category"`
	League      string `json:"league" db:"league"`
}

type PlayerAward struct {
	ID       int    `json:"id" db:"id"`
	PlayerID int    `json:"player_id" db:"player_id"`
	AwardID  int    `json:"award_id" db:"award_id"`
	Year     int    `json:"year" db:"year"`
	TeamID   *int   `json:"team_id" db:"team_id"`
	Notes    string `json:"notes" db:"notes"`

	// Joined fields
	AwardName  string `json:"award_name,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	TeamName   string `json:"team_name,omitempty"`
}
