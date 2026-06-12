// ═══════════════════════════════════════════════════════════
// PROFILE MANAGEMENT METHODS
// Add these methods to your existing UserService in
// sessions/user_service.go (same package, same struct receiver)
// ═══════════════════════════════════════════════════════════

package sessions

import (
	"database/sql"
	"fmt"
	"trivia-server/models"

	"golang.org/x/crypto/bcrypt"
)

// IsUsernameAvailable checks if a username is free to use.
// excludeUserID lets a user "change" their name to the same value
// without tripping the duplicate check against themselves.
func (us *UserService) IsUsernameAvailable(username string, excludeUserID int) (bool, error) {
	var count int
	err := us.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE username = ? AND id != ?`,
		username, excludeUserID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check username availability: %w", err)
	}
	return count == 0, nil
}

// UpdateUsername changes a user's username after confirming availability.
func (us *UserService) UpdateUsername(userID int, newUsername string) error {
	available, err := us.IsUsernameAvailable(newUsername, userID)
	if err != nil {
		return err
	}
	if !available {
		return fmt.Errorf("username already taken")
	}

	_, err = us.db.Exec(
		`UPDATE users SET username = ?, updated_at = NOW() WHERE id = ?`,
		newUsername, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}
	return nil
}

// UpdatePassword verifies the current password, then sets the new one.
func (us *UserService) UpdatePassword(userID int, currentPassword, newPassword string) error {
	user, err := us.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	_, err = us.db.Exec(
		`UPDATE users SET password_hash = ?, updated_at = NOW() WHERE id = ?`,
		string(hashed), userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// UpdateFavoriteTeam sets or clears the user's favorite team.
// Pass teamID = 0 and teamName = "" to clear.
func (us *UserService) UpdateFavoriteTeam(userID int, teamID int, teamName string) error {
	if teamID == 0 {
		_, err := us.db.Exec(
			`UPDATE users SET favorite_team_id = NULL, favorite_team_name = NULL, updated_at = NOW() WHERE id = ?`,
			userID,
		)
		if err != nil {
			return fmt.Errorf("failed to clear favorite team: %w", err)
		}
		return nil
	}

	_, err := us.db.Exec(
		`UPDATE users SET favorite_team_id = ?, favorite_team_name = ?, updated_at = NOW() WHERE id = ?`,
		teamID, teamName, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update favorite team: %w", err)
	}
	return nil
}

// DeleteAccount soft-deletes a user: the row is kept (so historical
// game_players / game_moves rows referencing this user_id remain valid
// for opponents), but the account is deactivated and the username is
// anonymized so it's freed up for reuse and no longer identifies the person.
func (us *UserService) DeleteAccount(userID int) error {
	anonymizedName := fmt.Sprintf("deleted_user_%d", userID)

	_, err := us.db.Exec(`
		UPDATE users
		SET username      = ?,
		    email         = ?,
		    password_hash = '',
		    is_active     = FALSE,
		    deleted_at    = NOW(),
		    favorite_team_id   = NULL,
		    favorite_team_name = NULL,
		    updated_at    = NOW()
		WHERE id = ?
	`, anonymizedName, fmt.Sprintf("deleted_%d@deleted.local", userID), userID)

	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

// GetGameHistory returns recent completed games for a user, including
// the opponent's username and the result from this user's perspective.
func (us *UserService) GetGameHistory(userID int, limit int) ([]models.GameHistoryEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := us.db.Query(`
		SELECT
			g.id,
			COALESCE(opp.username, 'Unknown') AS opponent_name,
			CASE
				WHEN g.winner_id IS NULL THEN 'draw'
				WHEN g.winner_id = ? THEN 'win'
				ELSE 'loss'
			END AS result,
			COALESCE(g.difficulty, 'regular') AS difficulty,
			g.completed_at
		FROM games g
		JOIN game_players gp_me  ON gp_me.game_id = g.id AND gp_me.user_id = ?
		JOIN game_players gp_opp ON gp_opp.game_id = g.id AND gp_opp.user_id != ?
		LEFT JOIN users opp ON opp.id = gp_opp.user_id
		WHERE g.status = 'completed'
		ORDER BY g.completed_at DESC
		LIMIT ?
	`, userID, userID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game history: %w", err)
	}
	defer rows.Close()

	history := make([]models.GameHistoryEntry, 0)
	for rows.Next() {
		var entry models.GameHistoryEntry
		var completedAt sql.NullTime

		if err := rows.Scan(
			&entry.GameID,
			&entry.OpponentName,
			&entry.Result,
			&entry.Difficulty,
			&completedAt,
		); err != nil {
			continue
		}

		if completedAt.Valid {
			entry.PlayedAt = completedAt.Time
		}

		history = append(history, entry)
	}

	return history, nil
}

// GetUserByIDWithTeam is like GetUserByID but also includes favorite team info.
// Use this for the profile page.
func (us *UserService) GetUserByIDWithTeam(userID int) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at,
		       games_played, games_won, favorite_team_id, favorite_team_name
		FROM users WHERE id = ? AND is_active = TRUE`

	err := us.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.GamesPlayed,
		&user.GamesWon,
		&user.FavoriteTeamID,
		&user.FavoriteTeamName,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return user, nil
}
