-- migrations/001_initial_schema.sql
-- Initial database schema for Baseball Grid Game

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    games_played INT DEFAULT 0,
    games_won INT DEFAULT 0,
    INDEX idx_username (username),
    INDEX idx_email (email)
);

-- Teams table
CREATE TABLE IF NOT EXISTS teams (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    city VARCHAR(100) NOT NULL,
    abbreviation VARCHAR(5) NOT NULL,
    league VARCHAR(20) NOT NULL, -- AL or NL
    division VARCHAR(20) NOT NULL, -- East, Central, West
    founded_year INT,
    colors VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_abbreviation (abbreviation),
    INDEX idx_league (league)
);

INSERT INTO teams (name, city, abbreviation, league, division, founded_year) VALUES
-- American League East
('Yankees', 'New York', 'NYY', 'AL', 'East', 1901),
('Red Sox', 'Boston', 'BOS', 'AL', 'East', 1901),
('Blue Jays', 'Toronto', 'TOR', 'AL', 'East', 1977),
('Rays', 'Tampa Bay', 'TB', 'AL', 'East', 1998),
('Orioles', 'Baltimore', 'BAL', 'AL', 'East', 1901),

-- American League Central
('White Sox', 'Chicago', 'CWS', 'AL', 'Central', 1901),
('Guardians', 'Cleveland', 'CLE', 'AL', 'Central', 1901),
('Tigers', 'Detroit', 'DET', 'AL', 'Central', 1901),
('Royals', 'Kansas City', 'KC', 'AL', 'Central', 1969),
('Twins', 'Minnesota', 'MIN', 'AL', 'Central', 1901),

-- American League West
('Astros', 'Houston', 'HOU', 'AL', 'West', 1962),
('Angels', 'Los Angeles', 'LAA', 'AL', 'West', 1961),
('Athletics', 'Oakland', 'OAK', 'AL', 'West', 1901),
('Mariners', 'Seattle', 'SEA', 'AL', 'West', 1977),
('Rangers', 'Texas', 'TEX', 'AL', 'West', 1961),

-- National League East
('Braves', 'Atlanta', 'ATL', 'NL', 'East', 1871),
('Marlins', 'Miami', 'MIA', 'NL', 'East', 1993),
('Mets', 'New York', 'NYM', 'NL', 'East', 1962),
('Phillies', 'Philadelphia', 'PHI', 'NL', 'East', 1883),
('Nationals', 'Washington', 'WSH', 'NL', 'East', 1969),

-- National League Central
('Cubs', 'Chicago', 'CHC', 'NL', 'Central', 1876),
('Reds', 'Cincinnati', 'CIN', 'NL', 'Central', 1882),
('Brewers', 'Milwaukee', 'MIL', 'NL', 'Central', 1969),
('Pirates', 'Pittsburgh', 'PIT', 'NL', 'Central', 1882),
('Cardinals', 'St. Louis', 'STL', 'NL', 'Central', 1882),

-- National League West
('Diamondbacks', 'Arizona', 'ARI', 'NL', 'West', 1998),
('Rockies', 'Colorado', 'COL', 'NL', 'West', 1993),
('Dodgers', 'Los Angeles', 'LAD', 'NL', 'West', 1883),
('Padres', 'San Diego', 'SD', 'NL', 'West', 1969),
('Giants', 'San Francisco', 'SF', 'NL', 'West', 1883);

-- Add some historical teams (inactive) for more interesting grid possibilities
INSERT INTO teams (name, city, abbreviation, league, division, founded_year, is_active) VALUES
('Expos', 'Montreal', 'MON', 'NL', 'East', 1969, FALSE),
('Senators', 'Washington', 'WAS', 'AL', 'East', 1901, FALSE),
('Pilots', 'Seattle', 'SEA', 'AL', 'West', 1969, FALSE),
('Browns', 'St. Louis', 'SLB', 'AL', 'Central', 1902, FALSE);

-- Players table
CREATE TABLE IF NOT EXISTS players (
    id INT PRIMARY KEY AUTO_INCREMENT,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    birth_date DATE,
    birth_city VARCHAR(100),
    birth_state VARCHAR(50),
    birth_country VARCHAR(50),
    debut_date DATE,
    final_game_date DATE,
    primary_position VARCHAR(20),
    bats VARCHAR(10), -- L, R, S (switch)
    throws VARCHAR(10), -- L, R
    height_inches INT,
    weight_lbs INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_last_name (last_name),
    INDEX idx_full_name (last_name, first_name),
    INDEX idx_position (primary_position),
    INDEX idx_debut_date (debut_date)
);

-- Player-Team relationships (since players can play for multiple teams)
CREATE TABLE IF NOT EXISTS player_teams (
    id INT PRIMARY KEY AUTO_INCREMENT,
    player_id INT NOT NULL,
    team_id INT NOT NULL,
    start_year INT NOT NULL,
    end_year INT,
    position VARCHAR(20),
    is_primary_team BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    INDEX idx_player_team (player_id, team_id),
    INDEX idx_team_year (team_id, start_year),
    INDEX idx_player_year (player_id, start_year)
);

-- Awards table (MVP, Cy Young, ROY, etc.)
CREATE TABLE IF NOT EXISTS awards (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50), -- MVP, Pitching, Batting, Fielding, etc.
    league VARCHAR(20), -- AL, NL, or MLB for league-wide awards
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_category (category)
);

-- Player Awards junction table
CREATE TABLE IF NOT EXISTS player_awards (
    id INT PRIMARY KEY AUTO_INCREMENT,
    player_id INT NOT NULL,
    award_id INT NOT NULL,
    year INT NOT NULL,
    team_id INT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (award_id) REFERENCES awards(id) ON DELETE CASCADE,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL,
    UNIQUE KEY unique_player_award_year (player_id, award_id, year),
    INDEX idx_player_year (player_id, year),
    INDEX idx_award_year (award_id, year)
);

-- Basic player statistics (simplified for immaculate grid)
CREATE TABLE IF NOT EXISTS player_stats (
    id INT PRIMARY KEY AUTO_INCREMENT,
    player_id INT NOT NULL,
    team_id INT,
    year INT NOT NULL,
    games_played INT DEFAULT 0,
    at_bats INT DEFAULT 0,
    hits INT DEFAULT 0,
    doubles INT DEFAULT 0,
    triples INT DEFAULT 0,
    home_runs INT DEFAULT 0,
    rbis INT DEFAULT 0,
    stolen_bases INT DEFAULT 0,
    batting_average DECIMAL(4,3) DEFAULT 0.000,
    -- Pitching stats
    wins INT DEFAULT 0,
    losses INT DEFAULT 0,
    saves INT DEFAULT 0,
    innings_pitched DECIMAL(5,1) DEFAULT 0.0,
    strikeouts INT DEFAULT 0,
    era DECIMAL(4,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL,
    UNIQUE KEY unique_player_team_year (player_id, team_id, year),
    INDEX idx_player_year (player_id, year),
    INDEX idx_team_year (team_id, year),
    INDEX idx_home_runs (home_runs),
    INDEX idx_batting_average (batting_average)
);

-- Game sessions
CREATE TABLE IF NOT EXISTS games (
    id INT PRIMARY KEY AUTO_INCREMENT,
    game_uuid VARCHAR(36) UNIQUE NOT NULL,
    status ENUM('waiting', 'active', 'completed', 'abandoned') DEFAULT 'waiting',
    grid_config JSON, -- Store the 3x3 grid categories
    max_players INT DEFAULT 2,
    current_turn INT DEFAULT 1,
    winner_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    FOREIGN KEY (winner_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_game_uuid (game_uuid),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);

-- Game participants
CREATE TABLE IF NOT EXISTS game_players (
    id INT PRIMARY KEY AUTO_INCREMENT,
    game_id INT NOT NULL,
    user_id INT NOT NULL,
    player_number INT NOT NULL, -- 1 or 2
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_game_user (game_id, user_id),
    UNIQUE KEY unique_game_player_number (game_id, player_number),
    INDEX idx_game_id (game_id),
    INDEX idx_user_id (user_id)
);

-- Game moves (grid cell selections)
CREATE TABLE IF NOT EXISTS game_moves (
    id INT PRIMARY KEY AUTO_INCREMENT,
    game_id INT NOT NULL,
    user_id INT NOT NULL,
    grid_row INT NOT NULL, -- 0, 1, 2
    grid_col INT NOT NULL, -- 0, 1, 2
    player_answer VARCHAR(100), -- The player they chose
    player_id INT, -- Reference to actual player if valid
    is_valid BOOLEAN DEFAULT FALSE,
    move_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE SET NULL,
    UNIQUE KEY unique_game_position (game_id, grid_row, grid_col),
    INDEX idx_game_id (game_id),
    INDEX idx_user_id (user_id),
    INDEX idx_game_position (game_id, grid_row, grid_col)
);

-- Grid categories (for generating random grids)
CREATE TABLE IF NOT EXISTS grid_categories (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    type ENUM('team', 'award', 'stat', 'position', 'era') NOT NULL,
    criteria JSON, -- Store specific criteria (e.g., {"min_home_runs": 30})
    difficulty ENUM('easy', 'medium', 'hard') DEFAULT 'medium',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_type (type),
    INDEX idx_difficulty (difficulty),
    INDEX idx_active (is_active)
);

-- ═══════════════════════════════════════════════════════════
-- ELITE NINE — GRID DATA SCHEMA
-- ═══════════════════════════════════════════════════════════

-- Criteria: teams and stats that appear as row/column headers
CREATE TABLE IF NOT EXISTS criteria (
    id          INT PRIMARY KEY AUTO_INCREMENT,
    type        ENUM('team', 'stat', 'award', 'position') NOT NULL,
    label       VARCHAR(100) NOT NULL,   -- "New York Yankees", "500+ HR Career"
    short_label VARCHAR(50),             -- "NYY", "500 HR"
    description VARCHAR(255),
    mlb_team_id INT DEFAULT NULL,        -- MLB Stats API team ID (for team criteria)
    stat_field  VARCHAR(50) DEFAULT NULL, -- "homeRuns", "battingAverage" etc
    stat_value  FLOAT DEFAULT NULL,      -- threshold value e.g. 0.300, 500, 3000
    stat_group  ENUM('hitting', 'pitching') DEFAULT NULL,
    award_id    VARCHAR(50) DEFAULT NULL, -- "MLBHOF", "MLBMVP" etc
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_label (label)
);

-- Players pulled from MLB Stats API
CREATE TABLE IF NOT EXISTS mlb_players (
    id           INT PRIMARY KEY AUTO_INCREMENT,
    mlb_id       INT NOT NULL UNIQUE,    -- MLB Stats API player ID
    full_name    VARCHAR(100) NOT NULL,
    position     VARCHAR(10),
    headshot_url VARCHAR(255),
    active       BOOLEAN DEFAULT FALSE,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Which players satisfy which criteria
CREATE TABLE IF NOT EXISTS player_criteria (
    id          INT PRIMARY KEY AUTO_INCREMENT,
    mlb_id      INT NOT NULL,
    criteria_id INT NOT NULL,
    FOREIGN KEY (mlb_id) REFERENCES mlb_players(mlb_id),
    FOREIGN KEY (criteria_id) REFERENCES criteria(id),
    UNIQUE KEY unique_player_criteria (mlb_id, criteria_id)
);

-- Pre-built grid templates
CREATE TABLE IF NOT EXISTS grid_templates (
    id              INT PRIMARY KEY AUTO_INCREMENT,
    row_criteria_1  INT NOT NULL,
    row_criteria_2  INT NOT NULL,
    row_criteria_3  INT NOT NULL,
    col_criteria_1  INT NOT NULL,
    col_criteria_2  INT NOT NULL,
    col_criteria_3  INT NOT NULL,
    min_answers     INT DEFAULT 0,   -- minimum valid answers across all 9 cells
    difficulty      ENUM('easy', 'medium', 'hard') DEFAULT 'medium',
    active          BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (row_criteria_1) REFERENCES criteria(id),
    FOREIGN KEY (row_criteria_2) REFERENCES criteria(id),
    FOREIGN KEY (row_criteria_3) REFERENCES criteria(id),
    FOREIGN KEY (col_criteria_1) REFERENCES criteria(id),
    FOREIGN KEY (col_criteria_2) REFERENCES criteria(id),
    FOREIGN KEY (col_criteria_3) REFERENCES criteria(id)
);

-- Valid answers per cell (row x col intersection)
-- Pre-computed so validation is instant at game time
CREATE TABLE IF NOT EXISTS cell_answers (
    id               INT PRIMARY KEY AUTO_INCREMENT,
    grid_template_id INT NOT NULL,
    row_index        INT NOT NULL,        -- 0, 1, 2
    col_index        INT NOT NULL,        -- 0, 1, 2
    mlb_id           INT NOT NULL,
    player_name      VARCHAR(100) NOT NULL,
    headshot_url     VARCHAR(255),
    rarity_score     FLOAT DEFAULT 0.5,   -- 0.0 = very rare, 1.0 = very common
    FOREIGN KEY (grid_template_id) REFERENCES grid_templates(id),
    FOREIGN KEY (mlb_id) REFERENCES mlb_players(mlb_id),
    UNIQUE KEY unique_cell_answer (grid_template_id, row_index, col_index, mlb_id)
);

-- Track how often each player is used as an answer (for rarity)
CREATE TABLE IF NOT EXISTS answer_frequency (
    mlb_id       INT NOT NULL PRIMARY KEY,
    use_count    INT DEFAULT 0,
    FOREIGN KEY (mlb_id) REFERENCES mlb_players(mlb_id)
);