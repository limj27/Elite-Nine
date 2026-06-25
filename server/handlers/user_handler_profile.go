// ═══════════════════════════════════════════════════════════
// PROFILE HANDLER METHODS
// Add these methods to your existing UserHandler in
// handlers/user_handler.go (same package, same struct receiver)
// ═══════════════════════════════════════════════════════════

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ── Request types ─────────────────────────────────────────

type UpdateUsernameRequest struct {
	Username string `json:"username"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UpdateFavoriteTeamRequest struct {
	TeamID   int    `json:"team_id"` // 0 clears the favorite team
	TeamName string `json:"team_name"`
}

type CheckUsernameRequest struct {
	Username string `json:"username"`
}

// ── GET /api/profile ─────────────────────────────────────────
// Returns full profile including favorite team. Overrides the
// simpler GetProfile if you want one richer endpoint — or keep
// both and point the frontend at this one.

func (uh *UserHandler) GetFullProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	user, err := uh.userService.GetUserByIDWithTeam(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ── POST /api/profile/check-username ─────────────────────────
// Body: {"username": "newname"}
// Returns: {"available": true/false}

func (uh *UserHandler) CheckUsernameAvailable(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	var req CheckUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	available, err := uh.userService.IsUsernameAvailable(username, userID)
	if err != nil {
		http.Error(w, "Failed to check username", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"available": available,
	})
}

// ── PUT /api/profile/username ─────────────────────────────────
// Body: {"username": "newname"}

func (uh *UserHandler) UpdateUsername(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	var req UpdateUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(req.Username)
	if len(username) < 3 {
		http.Error(w, "Username must be at least 3 characters", http.StatusBadRequest)
		return
	}

	if err := uh.userService.UpdateUsername(userID, username); err != nil {
		if strings.Contains(err.Error(), "already taken") {
			http.Error(w, "Username already taken", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to update username", http.StatusInternalServerError)
		return
	}

	user, err := uh.userService.GetUserByIDWithTeam(userID)
	if err != nil {
		http.Error(w, "Username updated but failed to fetch profile", http.StatusInternalServerError)
		return
	}

	// Issue a fresh token since username is embedded in the JWT claims
	token, err := uh.jwtService.GenerateToken(user)
	if err != nil {
		http.Error(w, "Username updated but failed to issue new token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
		User:  *user,
	})
}

// ── PUT /api/profile/password ─────────────────────────────────
// Body: {"current_password": "...", "new_password": "..."}

func (uh *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		http.Error(w, "New password must be at least 8 characters long", http.StatusBadRequest)
		return
	}

	if err := uh.userService.UpdatePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		if strings.Contains(err.Error(), "incorrect") {
			http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Password updated successfully"))
}

// ── PUT /api/profile/team ─────────────────────────────────────
// Body: {"team_id": 147, "team_name": "New York Yankees"}
// Send {"team_id": 0} to clear the favorite team.

func (uh *UserHandler) UpdateFavoriteTeam(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	var req UpdateFavoriteTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := uh.userService.UpdateFavoriteTeam(userID, req.TeamID, req.TeamName); err != nil {
		http.Error(w, "Failed to update favorite team", http.StatusInternalServerError)
		return
	}

	user, err := uh.userService.GetUserByIDWithTeam(userID)
	if err != nil {
		http.Error(w, "Team updated but failed to fetch profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ── DELETE /api/profile ───────────────────────────────────────
// Soft-deletes the account. Game history involving this user is
// preserved for opponents; the username is anonymized and freed up.

func (uh *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	if err := uh.userService.DeleteAccount(userID); err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	// Revoke any active session token
	_ = uh.jwtService.RevokeToken(userID)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Account deleted"))
}

// ── GET /api/profile/history ──────────────────────────────────
// Returns recent completed games for the logged-in user.

func (uh *UserHandler) GetGameHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(int)

	history, err := uh.userService.GetGameHistory(userID, 20)
	if err != nil {
		http.Error(w, "Failed to fetch game history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
	})
}
