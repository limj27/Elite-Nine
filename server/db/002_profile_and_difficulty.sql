-- migrations/002_profile_and_difficulty.sql

-- Favorite team for grid generation
ALTER TABLE users ADD COLUMN favorite_team_id INT DEFAULT NULL;
ALTER TABLE users ADD COLUMN favorite_team_name VARCHAR(100) DEFAULT NULL;

-- Difficulty per game (set by room creator)
ALTER TABLE games ADD COLUMN difficulty ENUM('easy','regular','hard') DEFAULT 'regular';

-- Soft-delete support — preserves game history for opponents
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;