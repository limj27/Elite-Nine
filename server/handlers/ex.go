package handlers

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"
)

// Question represents a single trivia question
type Question struct {
	ID              string   `json:"id"`
	Text            string   `json:"text"`
	CorrectAnswer   string   `json:"correctAnswer"`
	PossibleAnswers []string `json:"possibleAnswers"`
	Difficulty      int      `json:"difficulty"` // 1-3 scale (easy, medium, hard)
	Category        string   `json:"category"`   // e.g., "History", "Stats", "Players", etc.
	PointValue      int      `json:"pointValue"` // Points awarded for correct answer
}

// QuestionBank manages the pool of available trivia questions
type QuestionBank struct {
	Questions       []Question
	UsedQuestionIDs map[string]bool
	rand            *rand.Rand
}

// NewQuestionBank creates a new question bank
func NewQuestionBank(questions []Question) *QuestionBank {
	return &QuestionBank{
		Questions:       questions,
		UsedQuestionIDs: make(map[string]bool),
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetRandomQuestion returns a random question that hasn't been used yet
func (qb *QuestionBank) GetRandomQuestion() (Question, error) {
	// If all questions have been used, reset the used questions
	if len(qb.UsedQuestionIDs) >= len(qb.Questions) {
		qb.ResetUsedQuestions()
	}

	availableQuestions := []Question{}
	for _, q := range qb.Questions {
		if !qb.UsedQuestionIDs[q.ID] {
			availableQuestions = append(availableQuestions, q)
		}
	}

	if len(availableQuestions) == 0 {
		return Question{}, errors.New("no questions available")
	}

	selectedQuestion := availableQuestions[qb.rand.Intn(len(availableQuestions))]
	qb.UsedQuestionIDs[selectedQuestion.ID] = true
	return selectedQuestion, nil
}

// GetQuestionByCategory returns a random question from a specific category
func (qb *QuestionBank) GetQuestionByCategory(category string) (Question, error) {
	availableQuestions := []Question{}
	for _, q := range qb.Questions {
		if q.Category == category && !qb.UsedQuestionIDs[q.ID] {
			availableQuestions = append(availableQuestions, q)
		}
	}

	if len(availableQuestions) == 0 {
		return Question{}, errors.New("no questions available in that category")
	}

	selectedQuestion := availableQuestions[qb.rand.Intn(len(availableQuestions))]
	qb.UsedQuestionIDs[selectedQuestion.ID] = true
	return selectedQuestion, nil
}

// GetQuestionByDifficulty returns a random question with a specific difficulty level
func (qb *QuestionBank) GetQuestionByDifficulty(difficulty int) (Question, error) {
	availableQuestions := []Question{}
	for _, q := range qb.Questions {
		if q.Difficulty == difficulty && !qb.UsedQuestionIDs[q.ID] {
			availableQuestions = append(availableQuestions, q)
		}
	}

	if len(availableQuestions) == 0 {
		return Question{}, errors.New("no questions available with that difficulty")
	}

	selectedQuestion := availableQuestions[qb.rand.Intn(len(availableQuestions))]
	qb.UsedQuestionIDs[selectedQuestion.ID] = true
	return selectedQuestion, nil
}

// ResetUsedQuestions clears the list of used questions
func (qb *QuestionBank) ResetUsedQuestions() {
	qb.UsedQuestionIDs = make(map[string]bool)
}

// LoadQuestionsFromJSON loads questions from a JSON byte array
func LoadQuestionsFromJSON(data []byte) ([]Question, error) {
	var questions []Question
	err := json.Unmarshal(data, &questions)
	if err != nil {
		return nil, err
	}
	return questions, nil
}

// Answer represents a player's answer to a question
type Answer struct {
	UserID     string    `json:"userId"`
	QuestionID string    `json:"questionId"`
	Answer     string    `json:"answer"`
	AnsweredAt time.Time `json:"answeredAt"`
	TimeTaken  float64   `json:"timeTaken"` // Time in seconds
}

// ScoreCalculator calculates scores based on correctness and speed
type ScoreCalculator struct {
	BasePoints         int     // Base points for a correct answer
	MaxTimeBonus       int     // Maximum bonus for quick answers
	TimeBonusThreshold float64 // Time threshold in seconds for maximum bonus
}

// NewScoreCalculator creates a new score calculator with default settings
func NewScoreCalculator() *ScoreCalculator {
	return &ScoreCalculator{
		BasePoints:         100,
		MaxTimeBonus:       50,
		TimeBonusThreshold: 3.0, // 3 seconds
	}
}

// CalculateScore calculates the score for an answer
func (sc *ScoreCalculator) CalculateScore(answer Answer, question Question, isCorrect bool) int {
	if !isCorrect {
		return 0
	}

	// Base score from question's point value
	score := question.PointValue

	// Time bonus: faster answers get more points
	if answer.TimeTaken <= sc.TimeBonusThreshold {
		// Full bonus for answering within threshold
		score += sc.MaxTimeBonus
	} else {
		// Decreasing bonus for slower answers (linear falloff)
		timeBonus := int(float64(sc.MaxTimeBonus) * (1 - (answer.TimeTaken-sc.TimeBonusThreshold)/10.0))
		if timeBonus > 0 {
			score += timeBonus
		}
	}

	// Difficulty multiplier
	score = score * question.Difficulty

	return score
}

// ValidateAnswer checks if a player's answer is correct
func ValidateAnswer(question Question, answer string) bool {
	return answer == question.CorrectAnswer
}

// GameRound represents a round of the trivia game
type GameRound struct {
	RoundID         string
	CurrentQuestion Question
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration // How long players have to answer
	PlayerAnswers   map[string]Answer
	Scores          map[string]int
}

// NewGameRound creates a new game round
func NewGameRound(roundID string, question Question, duration time.Duration) *GameRound {
	return &GameRound{
		RoundID:         roundID,
		CurrentQuestion: question,
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(duration),
		Duration:        duration,
		PlayerAnswers:   make(map[string]Answer),
		Scores:          make(map[string]int),
	}
}

// SubmitAnswer records a player's answer
func (gr *GameRound) SubmitAnswer(userID, answer string) (bool, int, error) {
	// Check if the round is still active
	if time.Now().After(gr.EndTime) {
		return false, 0, errors.New("round has ended")
	}

	// Check if player already answered
	if _, exists := gr.PlayerAnswers[userID]; exists {
		return false, 0, errors.New("player already submitted an answer")
	}

	// Record the answer
	answeredAt := time.Now()
	timeTaken := answeredAt.Sub(gr.StartTime).Seconds()

	gr.PlayerAnswers[userID] = Answer{
		UserID:     userID,
		QuestionID: gr.CurrentQuestion.ID,
		Answer:     answer,
		AnsweredAt: answeredAt,
		TimeTaken:  timeTaken,
	}

	// Validate the answer
	isCorrect := ValidateAnswer(gr.CurrentQuestion, answer)

	// Calculate score
	score := 0
	if isCorrect {
		calculator := NewScoreCalculator()
		score = calculator.CalculateScore(gr.PlayerAnswers[userID], gr.CurrentQuestion, isCorrect)
		gr.Scores[userID] = score
	}

	return isCorrect, score, nil
}

// IsActive checks if the round is still active
func (gr *GameRound) IsActive() bool {
	return time.Now().Before(gr.EndTime)
}

// GetResults returns the results of the round
func (gr *GameRound) GetResults() map[string]struct {
	Answer    string
	IsCorrect bool
	Score     int
} {
	results := make(map[string]struct {
		Answer    string
		IsCorrect bool
		Score     int
	})

	for userID, answer := range gr.PlayerAnswers {
		isCorrect := ValidateAnswer(gr.CurrentQuestion, answer.Answer)
		score := gr.Scores[userID]

		results[userID] = struct {
			Answer    string
			IsCorrect bool
			Score     int
		}{
			Answer:    answer.Answer,
			IsCorrect: isCorrect,
			Score:     score,
		}
	}

	return results
}
