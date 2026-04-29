#!/usr/bin/env python3
"""
Elite Nine — MLB Data Population Script
========================================
Pulls data from the free MLB Stats API and populates your MySQL database
with players, criteria, and valid grid templates.
"""

import os
import sys
import time
import random
import itertools
import requests
import mysql.connector
from dotenv import load_dotenv

load_dotenv()

# ═══════════════════════════════════════════════════════════
# CONFIG
# ═══════════════════════════════════════════════════════════
MLB_API = "https://statsapi.mlb.com/api/v1"

# Minimum number of valid answers a cell must have to be usable
MIN_ANSWERS_PER_CELL = 3

# Minimum answers across all 9 cells for a grid to be valid
MIN_TOTAL_ANSWERS = 27

# How many grid templates to generate
TARGET_GRIDS = 50

# Rate limiting — be polite to the free API
REQUEST_DELAY = 0.3  # seconds between requests

# ═══════════════════════════════════════════════════════════
# STAT CRITERIA DEFINITIONS
# These are the thresholds players must meet
# ═══════════════════════════════════════════════════════════
STAT_CRITERIA = [
    # Batting
    {"label": ".300+ Career AVG",      "short_label": ".300 AVG",   "stat_field": "avg",            "stat_value": 0.300, "stat_group": "hitting"},
    {"label": ".400+ Career OBP",      "short_label": ".400 OBP",   "stat_field": "obp",            "stat_value": 0.400, "stat_group": "hitting"},
    {"label": "500+ Career HR",        "short_label": "500 HR",     "stat_field": "homeRuns",       "stat_value": 500,   "stat_group": "hitting"},
    {"label": "400+ Career HR",        "short_label": "400 HR",     "stat_field": "homeRuns",       "stat_value": 400,   "stat_group": "hitting"},
    {"label": "300+ Career HR",        "short_label": "300 HR",     "stat_field": "homeRuns",       "stat_value": 300,   "stat_group": "hitting"},
    {"label": "3000+ Career Hits",     "short_label": "3000 H",     "stat_field": "hits",           "stat_value": 3000,  "stat_group": "hitting"},
    {"label": "2000+ Career Hits",     "short_label": "2000 H",     "stat_field": "hits",           "stat_value": 2000,  "stat_group": "hitting"},
    {"label": "300+ Career SB",        "short_label": "300 SB",     "stat_field": "stolenBases",    "stat_value": 300,   "stat_group": "hitting"},
    {"label": "1500+ Career RBI",      "short_label": "1500 RBI",   "stat_field": "rbi",            "stat_value": 1500,  "stat_group": "hitting"},
    {"label": "1000+ Career RBI",      "short_label": "1000 RBI",   "stat_field": "rbi",            "stat_value": 1000,  "stat_group": "hitting"},
    # Pitching
    {"label": "200+ Career Wins",      "short_label": "200 W",      "stat_field": "wins",           "stat_value": 200,   "stat_group": "pitching"},
    {"label": "150+ Career Wins",      "short_label": "150 W",      "stat_field": "wins",           "stat_value": 150,   "stat_group": "pitching"},
    {"label": "3000+ Career K",        "short_label": "3000 K",     "stat_field": "strikeOuts",     "stat_value": 3000,  "stat_group": "pitching"},
    {"label": "2000+ Career K",        "short_label": "2000 K",     "stat_field": "strikeOuts",     "stat_value": 2000,  "stat_group": "pitching"},
    {"label": "Sub-3.00 Career ERA",   "short_label": "ERA < 3.00", "stat_field": "era",            "stat_value": 3.00,  "stat_group": "pitching"},
    {"label": "300+ Career Saves",     "short_label": "300 SV",     "stat_field": "saves",          "stat_value": 300,   "stat_group": "pitching"},
]

# Award criteria — we'll pull winners from the API
AWARD_CRITERIA = [
    # MLB-wide awards
    {"label": "Hall of Fame",           "short_label": "HOF",         "award_id": "MLBHOF"},
    {"label": "World Series Champion",  "short_label": "WS Champ",    "award_id": "WSCHAMP"},
    {"label": "World Series MVP",       "short_label": "WS MVP",      "award_id": "WSMVP"},

    # League MVP (AL + NL separately since there's no combined MLB MVP)
    {"label": "AL MVP",                 "short_label": "AL MVP",      "award_id": "ALMVP"},
    {"label": "NL MVP",                 "short_label": "NL MVP",      "award_id": "NLMVP"},

    # Cy Young
    {"label": "AL Cy Young",            "short_label": "AL CYA",      "award_id": "ALCY"},
    {"label": "NL Cy Young",            "short_label": "NL CYA",      "award_id": "NLCY"},

    # Rookie of the Year
    {"label": "AL Rookie of the Year",  "short_label": "AL ROY",      "award_id": "ALROY"},
    {"label": "NL Rookie of the Year",  "short_label": "NL ROY",      "award_id": "NLROY"},

    # All-Star
    {"label": "AL All-Star",            "short_label": "AL All-Star",  "award_id": "ALAS"},
    {"label": "NL All-Star",            "short_label": "NL All-Star",  "award_id": "NLAS"},

    # Gold Glove
    {"label": "AL Gold Glove",          "short_label": "AL GG",       "award_id": "ALGG"},
    {"label": "NL Gold Glove",          "short_label": "NL GG",       "award_id": "NLGG"},

    # Silver Slugger
    {"label": "AL Silver Slugger",      "short_label": "AL SS",       "award_id": "ALSS"},
    {"label": "NL Silver Slugger",      "short_label": "NL SS",       "award_id": "NLSS"},
]

# ═══════════════════════════════════════════════════════════
# DB CONNECTION
# ═══════════════════════════════════════════════════════════
def get_db():
    return mysql.connector.connect(
        host=os.getenv("DB_HOST", "localhost"),
        port=int(os.getenv("DB_PORT", 3306)),
        user=os.getenv("DB_USER", "gameuser"),
        password=os.getenv("DB_PASSWORD", "gamepassword"),
        database=os.getenv("DB_NAME", "baseball_game"),
    )

# ═══════════════════════════════════════════════════════════
# API HELPERS
# ═══════════════════════════════════════════════════════════
session = requests.Session()
session.headers.update({"User-Agent": "EliteNine/1.0 (personal project)"})

def api_get(path, params=None, retries=3):
    url = f"{MLB_API}{path}"
    for attempt in range(retries):
        try:
            time.sleep(REQUEST_DELAY)
            resp = session.get(url, params=params, timeout=15)
            resp.raise_for_status()
            return resp.json()
        except requests.RequestException as e:
            if attempt == retries - 1:
                print(f"  ✗ API error for {path}: {e}")
                return None
            time.sleep(2 ** attempt)
    return None

def headshot_url(mlb_id):
    return (
        f"https://img.mlbstatic.com/mlb-photos/image/upload/"
        f"d_people:generic:headshot:67:current.png/w_213,q_auto:best/"
        f"v1/people/{mlb_id}/headshot/67/current"
    )

# ═══════════════════════════════════════════════════════════
# STEP 1 — INSERT TEAM CRITERIA
# ═══════════════════════════════════════════════════════════
def populate_teams(db):
    cursor = db.cursor()
    print("\n📋 Fetching all 30 MLB teams...")

    data = api_get("/teams", {"sportId": 1, "activeStatus": "Yes"})
    if not data:
        print("  ✗ Failed to fetch teams")
        return {}

    teams = data.get("teams", [])
    team_map = {}  # label → criteria_id

    for team in teams:
        team_id  = team["id"]
        name     = team["name"]
        abbrev   = team.get("abbreviation", "")

        try:
            cursor.execute("""
                INSERT INTO criteria (type, label, short_label, mlb_team_id)
                VALUES ('team', %s, %s, %s)
                ON DUPLICATE KEY UPDATE mlb_team_id = VALUES(mlb_team_id)
            """, (name, abbrev, team_id))
            db.commit()

            cursor.execute("SELECT id FROM criteria WHERE label = %s", (name,))
            row = cursor.fetchone()
            if row:
                team_map[name] = {"criteria_id": row[0], "mlb_team_id": team_id}
                print(f"  ✓ {name} ({abbrev})")
        except Exception as e:
            print(f"  ✗ Error inserting {name}: {e}")
            db.rollback()

    cursor.close()
    print(f"  → {len(team_map)} teams inserted")
    return team_map

# ═══════════════════════════════════════════════════════════
# STEP 2 — INSERT STAT AND AWARD CRITERIA
# ═══════════════════════════════════════════════════════════
def populate_criteria(db):
    cursor = db.cursor()
    print("\n📊 Inserting stat criteria...")

    stat_map = {}

    for crit in STAT_CRITERIA:
        try:
            cursor.execute("""
                INSERT INTO criteria (type, label, short_label, stat_field, stat_value, stat_group)
                VALUES ('stat', %s, %s, %s, %s, %s)
                ON DUPLICATE KEY UPDATE stat_value = VALUES(stat_value)
            """, (crit["label"], crit["short_label"], crit["stat_field"],
                  crit["stat_value"], crit["stat_group"]))
            db.commit()

            cursor.execute("SELECT id FROM criteria WHERE label = %s", (crit["label"],))
            row = cursor.fetchone()
            if row:
                stat_map[crit["label"]] = row[0]
                print(f"  ✓ {crit['label']}")
        except Exception as e:
            print(f"  ✗ Error inserting {crit['label']}: {e}")
            db.rollback()

    print("\n🏆 Inserting award criteria...")
    award_map = {}

    for award in AWARD_CRITERIA:
        try:
            cursor.execute("""
                INSERT INTO criteria (type, label, short_label, award_id)
                VALUES ('award', %s, %s, %s)
                ON DUPLICATE KEY UPDATE award_id = VALUES(award_id)
            """, (award["label"], award["short_label"], award["award_id"]))
            db.commit()

            cursor.execute("SELECT id FROM criteria WHERE label = %s", (award["label"],))
            row = cursor.fetchone()
            if row:
                award_map[award["label"]] = {"criteria_id": row[0], "award_id": award["award_id"]}
                print(f"  ✓ {award['label']}")
        except Exception as e:
            print(f"  ✗ Error inserting {award['label']}: {e}")
            db.rollback()

    cursor.close()
    return stat_map, award_map

# ═══════════════════════════════════════════════════════════
# STEP 3 — POPULATE PLAYERS VIA STAT LEADERS
# ═══════════════════════════════════════════════════════════
def populate_players_from_leaders(db):
    """
    Use the stat leaders endpoint to find players who meet thresholds.
    This is more reliable than pulling all players and checking stats.
    """
    cursor = db.cursor()
    print("\n⚾ Populating players from stat leaders...")

    # Mapping: (stat_field, stat_group) → list of player mlb_ids who qualify
    qualifying_players = {}  # criteria_label → set of mlb_ids

    # Hitting stats — use career leaders
    hitting_stats = [c for c in STAT_CRITERIA if c["stat_group"] == "hitting"]
    for crit in hitting_stats:
        field    = crit["stat_field"]
        threshold = crit["stat_value"]
        label    = crit["label"]
        print(f"\n  Fetching leaders for {label}...")

        # Pull top 2000 career leaders for this stat
        data = api_get("/stats/leaders", {
            "leaderCategories": field,
            "statGroup":        "hitting",
            "statType":         "career",
            "sportId":          1,
            "limit":            2000,
        })

        if not data:
            continue

        qualifying = set()
        leaders = data.get("leagueLeaders", [])
        for category in leaders:
            for leader in category.get("leaders", []):
                try:
                    value = float(leader.get("value", 0))
                    # For ERA, lower is better
                    passes = value >= threshold

                    if passes:
                        player    = leader.get("person", {})
                        mlb_id    = player.get("id")
                        full_name = player.get("fullName", "")
                        if mlb_id:
                            qualifying.add(mlb_id)
                            upsert_player(cursor, db, mlb_id, full_name)
                except (ValueError, TypeError):
                    continue

        qualifying_players[label] = qualifying
        print(f"    → {len(qualifying)} players qualify")

    # Pitching stats
    pitching_stats = [c for c in STAT_CRITERIA if c["stat_group"] == "pitching"]
    for crit in pitching_stats:
        field     = crit["stat_field"]
        threshold = crit["stat_value"]
        label     = crit["label"]
        print(f"\n  Fetching leaders for {label}...")

        data = api_get("/stats/leaders", {
            "leaderCategories": field,
            "statGroup":        "pitching",
            "statType":         "career",
            "sportId":          1,
            "limit":            2000,
        })

        if not data:
            continue

        qualifying = set()
        leaders = data.get("leagueLeaders", [])
        for category in leaders:
            for leader in category.get("leaders", []):
                try:
                    value = float(leader.get("value", 0))
                    # ERA: lower is better
                    if field == "era":
                        passes = value <= threshold
                    else:
                        passes = value >= threshold

                    if passes:
                        player    = leader.get("person", {})
                        mlb_id    = player.get("id")
                        full_name = player.get("fullName", "")
                        if mlb_id:
                            qualifying.add(mlb_id)
                            upsert_player(cursor, db, mlb_id, full_name)
                except (ValueError, TypeError):
                    continue

        qualifying_players[label] = qualifying
        print(f"    → {len(qualifying)} players qualify")

    cursor.close()
    return qualifying_players

def upsert_player(cursor, db, mlb_id, full_name, position=None):
    try:
        hs = headshot_url(mlb_id)
        cursor.execute("""
            INSERT INTO mlb_players (mlb_id, full_name, position, headshot_url)
            VALUES (%s, %s, %s, %s)
            ON DUPLICATE KEY UPDATE full_name = VALUES(full_name), headshot_url = VALUES(headshot_url)
        """, (mlb_id, full_name, position, hs))
        db.commit()
    except Exception as e:
        db.rollback()

# ═══════════════════════════════════════════════════════════
# STEP 4 — POPULATE AWARD WINNERS
# ═══════════════════════════════════════════════════════════
def populate_award_winners(db, award_map):
    cursor = db.cursor()
    print("\n🏅 Fetching award winners...")

    award_players = {}

    for label, info in award_map.items():
        award_id    = info["award_id"]
        criteria_id = info["criteria_id"]
        print(f"\n  Fetching {label} winners...")

        qualifying = set()
        current_year = 2024

        # HOF doesn't use season parameter — fetch all at once
        if award_id == "MLBHOF":
            data = api_get(f"/awards/{award_id}/recipients")
            if data:
                for award in data.get("awards", []):
                    player    = award.get("player", {})
                    mlb_id    = player.get("id")
                    full_name = player.get("fullName", "")
                    if mlb_id:
                        qualifying.add(mlb_id)
                        upsert_player(cursor, db, mlb_id, full_name)
        else:
            for year in range(1950, current_year + 1):
                data = api_get(f"/awards/{award_id}/recipients", {"season": year})
                if not data:
                    continue

                awards = data.get("awards", [])
                for award in awards:
                    player    = award.get("player", {})
                    mlb_id    = player.get("id")
                    full_name = player.get("fullName", "")
                    if mlb_id:
                        qualifying.add(mlb_id)
                        upsert_player(cursor, db, mlb_id, full_name)

        award_players[label] = qualifying
        print(f"    → {len(qualifying)} winners found")

    cursor.close()
    return award_players

# ═══════════════════════════════════════════════════════════
# STEP 5 — POPULATE TEAM HISTORY PER PLAYER
# ═══════════════════════════════════════════════════════════
def populate_team_history(db, team_map):
    """
    For each team, pull historical rosters by season to find
    which players played for that team.
    """
    cursor = db.cursor()
    print("\n🏟️  Fetching team roster history...")

    # team label → set of mlb_ids
    team_players = {}

    current_year = 2024

    for team_label, team_info in team_map.items():
        mlb_team_id = team_info["mlb_team_id"]
        criteria_id = team_info["criteria_id"]
        print(f"\n  {team_label}...")

        qualifying = set()

        # Pull rosters from 1969 onwards (expansion era)
        for year in range(1969, current_year + 1):
            data = api_get(f"/teams/{mlb_team_id}/roster", {
                "rosterType": "40Man",
                "season":     year,
            })
            if not data:
                continue

            roster = data.get("roster", [])
            for entry in roster:
                person    = entry.get("person", {})
                mlb_id    = person.get("id")
                full_name = person.get("fullName", "")
                if mlb_id:
                    qualifying.add(mlb_id)
                    upsert_player(cursor, db, mlb_id, full_name)

        team_players[team_label] = qualifying
        print(f"    → {len(qualifying)} players found")

    cursor.close()
    return team_players

# ═══════════════════════════════════════════════════════════
# STEP 6 — INSERT PLAYER CRITERIA LINKS
# ═══════════════════════════════════════════════════════════
def insert_player_criteria(db, criteria_label, player_mlb_ids):
    cursor = db.cursor()

    cursor.execute("SELECT id FROM criteria WHERE label = %s", (criteria_label,))
    row = cursor.fetchone()
    if not row:
        cursor.close()
        return
    criteria_id = row[0]

    inserted = 0
    for mlb_id in player_mlb_ids:
        try:
            cursor.execute("""
                INSERT IGNORE INTO player_criteria (mlb_id, criteria_id)
                VALUES (%s, %s)
            """, (mlb_id, criteria_id))
            inserted += cursor.rowcount
        except Exception:
            pass

    db.commit()
    cursor.close()
    print(f"    → Inserted {inserted} player_criteria rows for '{criteria_label}'")

# ═══════════════════════════════════════════════════════════
# STEP 7 — BUILD GRID TEMPLATES
# ═══════════════════════════════════════════════════════════
def get_valid_answers_for_cell(db, row_criteria_id, col_criteria_id):
    """
    Find all players who satisfy BOTH row and column criteria.
    """
    cursor = db.cursor()
    cursor.execute("""
        SELECT p.mlb_id, p.full_name, p.headshot_url
        FROM mlb_players p
        JOIN player_criteria pc1 ON p.mlb_id = pc1.mlb_id AND pc1.criteria_id = %s
        JOIN player_criteria pc2 ON p.mlb_id = pc2.mlb_id AND pc2.criteria_id = %s
    """, (row_criteria_id, col_criteria_id))
    results = cursor.fetchall()
    cursor.close()
    return results

def build_grid_templates(db, all_criteria_ids):
    cursor = db.cursor()
    print(f"\n🏗️  Building grid templates from {len(all_criteria_ids)} criteria...")

    # Separate team and non-team criteria
    cursor.execute("SELECT id, type FROM criteria WHERE id IN (%s)" %
                   ",".join(["%s"] * len(all_criteria_ids)), all_criteria_ids)
    rows = cursor.fetchall()
    team_ids = [r[0] for r in rows if r[1] == 'team']
    stat_ids = [r[0] for r in rows if r[1] != 'team']

    print(f"  {len(team_ids)} team criteria, {len(stat_ids)} stat/award criteria")

    grids_created = 0
    attempts      = 0
    max_attempts  = TARGET_GRIDS * 20

    while grids_created < TARGET_GRIDS and attempts < max_attempts:
        attempts += 1

        # Random grid type: all teams, all stats, or mixed
        grid_type = random.choice(["all_teams", "mixed", "mixed", "mixed"])

        if grid_type == "all_teams":
            if len(team_ids) < 6:
                continue
            selected = random.sample(team_ids, 6)
            row_ids  = selected[:3]
            col_ids  = selected[3:]
        else:
            # Mixed: 2-4 teams + some stats
            n_teams = random.randint(2, 4)
            n_stats = 6 - n_teams
            if len(team_ids) < n_teams or len(stat_ids) < n_stats:
                continue
            selected = random.sample(team_ids, n_teams) + random.sample(stat_ids, n_stats)
            random.shuffle(selected)
            row_ids = selected[:3]
            col_ids = selected[3:]

        # Ensure no duplicates across rows and cols
        if len(set(row_ids + col_ids)) < 6:
            continue

        # Check every cell has enough answers
        valid = True
        cell_data = {}
        total_answers = 0

        for ri, row_c in enumerate(row_ids):
            for ci, col_c in enumerate(col_ids):
                answers = get_valid_answers_for_cell(db, row_c, col_c)
                if len(answers) < MIN_ANSWERS_PER_CELL:
                    valid = False
                    break
                cell_data[(ri, ci)] = answers
                total_answers += len(answers)
            if not valid:
                break

        if not valid or total_answers < MIN_TOTAL_ANSWERS:
            continue

        # Determine difficulty
        avg_answers = total_answers / 9
        if avg_answers > 20:
            difficulty = "easy"
        elif avg_answers > 8:
            difficulty = "medium"
        else:
            difficulty = "hard"

        # Insert the grid template
        try:
            cursor.execute("""
                INSERT INTO grid_templates
                (row_criteria_1, row_criteria_2, row_criteria_3,
                 col_criteria_1, col_criteria_2, col_criteria_3,
                 min_answers, difficulty)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
            """, (*row_ids, *col_ids, total_answers, difficulty))
            grid_id = cursor.lastrowid
            db.commit()

            # Insert cell answers with rarity scores
            for (ri, ci), answers in cell_data.items():
                total = len(answers)
                for rank, (mlb_id, player_name, hs_url) in enumerate(answers):
                    # Rarity: players that satisfy fewer cells are rarer
                    # Lower score = rarer (harder to guess)
                    rarity = (rank + 1) / total
                    cursor.execute("""
                        INSERT IGNORE INTO cell_answers
                        (grid_template_id, row_index, col_index, mlb_id, player_name, headshot_url, rarity_score)
                        VALUES (%s, %s, %s, %s, %s, %s, %s)
                    """, (grid_id, ri, ci, mlb_id, player_name, hs_url, rarity))

            db.commit()
            grids_created += 1
            print(f"  ✓ Grid {grids_created}/{TARGET_GRIDS} created (difficulty: {difficulty}, avg answers: {avg_answers:.1f})")

        except Exception as e:
            print(f"  ✗ Error inserting grid: {e}")
            db.rollback()

    cursor.close()
    print(f"\n  → {grids_created} grid templates created")

# ═══════════════════════════════════════════════════════════
# STEP 8 — CALCULATE RARITY SCORES
# ═══════════════════════════════════════════════════════════
def calculate_rarity(db):
    """
    Update rarity scores based on how many cells each player
    appears in across all grid templates.
    Players who appear in fewer cells are rarer.
    """
    cursor = db.cursor()
    print("\n🎯 Calculating rarity scores...")

    cursor.execute("""
        UPDATE cell_answers ca
        JOIN (
            SELECT mlb_id, COUNT(*) as appearances
            FROM cell_answers
            GROUP BY mlb_id
        ) counts ON ca.mlb_id = counts.mlb_id
        JOIN (
            SELECT MAX(appearances) as max_appearances
            FROM (
                SELECT mlb_id, COUNT(*) as appearances
                FROM cell_answers
                GROUP BY mlb_id
            ) sub
        ) maxc ON 1=1
        SET ca.rarity_score = counts.appearances / maxc.max_appearances
    """)
    db.commit()
    cursor.close()
    print("  → Rarity scores updated")

# ═══════════════════════════════════════════════════════════
# MAIN
# ═══════════════════════════════════════════════════════════
def main():
    print("=" * 60)
    print("  ELITE NINE — MLB DATA POPULATION SCRIPT")
    print("=" * 60)

    db = get_db()
    print(f"\n✅ Connected to database: {os.getenv('DB_NAME', 'baseball_game')}")

    # Step 1 — Teams
    team_map = populate_teams(db)
    if not team_map:
        print("❌ No teams found — aborting")
        sys.exit(1)

    # Step 2 — Stat + award criteria
    stat_map, award_map = populate_criteria(db)

    # Step 3 — Players from stat leaders
    qualifying_players = populate_players_from_leaders(db)
    for label, player_ids in qualifying_players.items():
        insert_player_criteria(db, label, player_ids)

    # Step 4 — Award winners
    award_players = populate_award_winners(db, award_map)
    for label, player_ids in award_players.items():
        insert_player_criteria(db, label, player_ids)

    # Step 5 — Team history (this is the slow step — ~30 teams × 55 years)
    print("\n⚠️  Team history step is slow (~30 min). Press Ctrl+C to skip.")
    try:
        team_players = populate_team_history(db, team_map)
        for label, player_ids in team_players.items():
            insert_player_criteria(db, label, player_ids)
    except KeyboardInterrupt:
        print("\n  Skipped team history — you can re-run just this step later")

    # Step 6 — Build grids
    cursor = db.cursor()
    cursor.execute("SELECT id FROM criteria")
    all_criteria_ids = [row[0] for row in cursor.fetchall()]
    cursor.close()

    build_grid_templates(db, all_criteria_ids)

    # Step 7 — Calculate rarity
    calculate_rarity(db)

    db.close()

    print("\n" + "=" * 60)
    print("  ✅ POPULATION COMPLETE")
    print("=" * 60)

if __name__ == "__main__":
    main()