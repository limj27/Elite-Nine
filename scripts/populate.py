#!/usr/bin/env python3
"""
Elite Nine — MLB Data Population Script
========================================
Pulls data from the free MLB Stats API and populates your MySQL database
with players, criteria, and valid grid templates.

Usage:
    pip install requests mysql-connector-python python-dotenv
    python3 populate.py

Set these in scripts/.env:
    DB_HOST=localhost
    DB_PORT=3306
    DB_USER=gameuser
    DB_PASSWORD=gamepassword
    DB_NAME=baseball_game
"""

import os
import sys
import time
import random
import requests
import mysql.connector
from dotenv import load_dotenv

load_dotenv()

# ═══════════════════════════════════════════════════════════
# CONFIG
# ═══════════════════════════════════════════════════════════
MLB_API              = "https://statsapi.mlb.com/api/v1"
MIN_ANSWERS_PER_CELL = 3
MIN_TOTAL_ANSWERS    = 27
TARGET_GRIDS         = 50
REQUEST_DELAY        = 0.3

# ═══════════════════════════════════════════════════════════
# STAT CRITERIA
# ═══════════════════════════════════════════════════════════
STAT_CRITERIA = [
    {"label": ".300+ Career AVG",    "short_label": ".300 AVG",   "stat_field": "avg",         "stat_value": 0.300, "stat_group": "hitting"},
    {"label": ".400+ Career OBP",    "short_label": ".400 OBP",   "stat_field": "obp",         "stat_value": 0.400, "stat_group": "hitting"},
    {"label": "500+ Career HR",      "short_label": "500 HR",     "stat_field": "homeRuns",    "stat_value": 500,   "stat_group": "hitting"},
    {"label": "400+ Career HR",      "short_label": "400 HR",     "stat_field": "homeRuns",    "stat_value": 400,   "stat_group": "hitting"},
    {"label": "300+ Career HR",      "short_label": "300 HR",     "stat_field": "homeRuns",    "stat_value": 300,   "stat_group": "hitting"},
    {"label": "3000+ Career Hits",   "short_label": "3000 H",     "stat_field": "hits",        "stat_value": 3000,  "stat_group": "hitting"},
    {"label": "2000+ Career Hits",   "short_label": "2000 H",     "stat_field": "hits",        "stat_value": 2000,  "stat_group": "hitting"},
    {"label": "300+ Career SB",      "short_label": "300 SB",     "stat_field": "stolenBases", "stat_value": 300,   "stat_group": "hitting"},
    {"label": "1500+ Career RBI",    "short_label": "1500 RBI",   "stat_field": "rbi",         "stat_value": 1500,  "stat_group": "hitting"},
    {"label": "1000+ Career RBI",    "short_label": "1000 RBI",   "stat_field": "rbi",         "stat_value": 1000,  "stat_group": "hitting"},
    {"label": "200+ Career Wins",    "short_label": "200 W",      "stat_field": "wins",        "stat_value": 200,   "stat_group": "pitching"},
    {"label": "150+ Career Wins",    "short_label": "150 W",      "stat_field": "wins",        "stat_value": 150,   "stat_group": "pitching"},
    {"label": "3000+ Career K",      "short_label": "3000 K",     "stat_field": "strikeOuts",  "stat_value": 3000,  "stat_group": "pitching"},
    {"label": "2000+ Career K",      "short_label": "2000 K",     "stat_field": "strikeOuts",  "stat_value": 2000,  "stat_group": "pitching"},
    {"label": "Sub-3.00 Career ERA", "short_label": "ERA < 3.00", "stat_field": "era",         "stat_value": 3.00,  "stat_group": "pitching"},
    {"label": "300+ Career Saves",   "short_label": "300 SV",     "stat_field": "saves",       "stat_value": 300,   "stat_group": "pitching"},
]

# ═══════════════════════════════════════════════════════════
# AWARD CRITERIA
# ═══════════════════════════════════════════════════════════
AWARD_CRITERIA = [
    {"label": "Hall of Fame",          "short_label": "Hall of Fame",       "award_id": "MLBHOF",  "start_year": None},
    {"label": "World Series Champion", "short_label": "WS Champion",        "award_id": "WSCHAMP", "start_year": 1903},
    {"label": "World Series MVP",      "short_label": "WS MVP",             "award_id": "WSMVP",   "start_year": 1955},
    {"label": "AL MVP",                "short_label": "AL MVP",             "award_id": "ALMVP",   "start_year": 1931},
    {"label": "NL MVP",                "short_label": "NL MVP",             "award_id": "NLMVP",   "start_year": 1931},
    {"label": "AL Cy Young",           "short_label": "AL Cy Young",        "award_id": "ALCY",    "start_year": 1967},
    {"label": "NL Cy Young",           "short_label": "NL Cy Young",        "award_id": "NLCY",    "start_year": 1967},
    {"label": "AL Rookie of the Year", "short_label": "AL Rookie of Year",  "award_id": "ALROY",   "start_year": 1949},
    {"label": "NL Rookie of the Year", "short_label": "NL Rookie of Year",  "award_id": "NLROY",   "start_year": 1947},
    {"label": "AL All-Star",           "short_label": "AL All-Star",        "award_id": "ALAS",    "start_year": 1933},
    {"label": "NL All-Star",           "short_label": "NL All-Star",        "award_id": "NLAS",    "start_year": 1933},
    {"label": "AL Gold Glove",         "short_label": "AL Gold Glove",      "award_id": "ALGG",    "start_year": 1957},
    {"label": "NL Gold Glove",         "short_label": "NL Gold Glove",      "award_id": "NLGG",    "start_year": 1957},
    {"label": "AL Silver Slugger",     "short_label": "AL Silver Slugger",  "award_id": "ALSS",    "start_year": 1980},
    {"label": "NL Silver Slugger",     "short_label": "NL Silver Slugger",  "award_id": "NLSS",    "start_year": 1980},
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
        autocommit=False,
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
# CORE DB HELPERS
# Each function opens + closes its own cursor and commits
# immediately so data is visible to all subsequent queries
# ═══════════════════════════════════════════════════════════
def upsert_player(db, mlb_id, full_name, position=None):
    cursor = db.cursor()
    try:
        hs = headshot_url(mlb_id)
        cursor.execute("""
            INSERT INTO mlb_players (mlb_id, full_name, position, headshot_url)
            VALUES (%s, %s, %s, %s)
            ON DUPLICATE KEY UPDATE
                full_name    = VALUES(full_name),
                headshot_url = VALUES(headshot_url)
        """, (mlb_id, full_name, position, hs))
        db.commit()
        return True
    except Exception:
        db.rollback()
        return False
    finally:
        cursor.close()

def insert_player_criteria_row(db, mlb_id, criteria_id):
    cursor = db.cursor()
    try:
        cursor.execute("""
            INSERT IGNORE INTO player_criteria (mlb_id, criteria_id)
            VALUES (%s, %s)
        """, (mlb_id, criteria_id))
        inserted = cursor.rowcount
        db.commit()
        return inserted
    except Exception:
        db.rollback()
        return 0
    finally:
        cursor.close()

def get_criteria_id(db, label):
    cursor = db.cursor()
    try:
        cursor.execute("SELECT id FROM criteria WHERE label = %s", (label,))
        row = cursor.fetchone()
        return row[0] if row else None
    finally:
        cursor.close()

# ═══════════════════════════════════════════════════════════
# STEP 1 — INSERT TEAM CRITERIA
# ═══════════════════════════════════════════════════════════
def populate_teams(db):
    print("\n📋 Fetching all 30 MLB teams...")
    data = api_get("/teams", {"sportId": 1, "activeStatus": "Yes"})
    if not data:
        print("  ✗ Failed to fetch teams")
        return {}

    team_map = {}
    cursor   = db.cursor()

    for team in data.get("teams", []):
        team_id = team["id"]
        name    = team["name"]
        abbrev  = team.get("abbreviation", "")
        try:
            cursor.execute("""
                INSERT INTO criteria (type, label, short_label, mlb_team_id)
                VALUES ('team', %s, %s, %s)
                ON DUPLICATE KEY UPDATE mlb_team_id = VALUES(mlb_team_id)
            """, (name, abbrev, team_id))
            db.commit()
            cid = get_criteria_id(db, name)
            if cid:
                team_map[name] = {"criteria_id": cid, "mlb_team_id": team_id}
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
    cursor    = db.cursor()
    stat_map  = {}
    award_map = {}

    print("\n📊 Inserting stat criteria...")
    for crit in STAT_CRITERIA:
        try:
            cursor.execute("""
                INSERT INTO criteria (type, label, short_label, stat_field, stat_value, stat_group)
                VALUES ('stat', %s, %s, %s, %s, %s)
                ON DUPLICATE KEY UPDATE stat_value = VALUES(stat_value)
            """, (crit["label"], crit["short_label"], crit["stat_field"],
                  crit["stat_value"], crit["stat_group"]))
            db.commit()
            cid = get_criteria_id(db, crit["label"])
            if cid:
                stat_map[crit["label"]] = cid
                print(f"  ✓ {crit['label']}")
        except Exception as e:
            print(f"  ✗ Error inserting {crit['label']}: {e}")
            db.rollback()

    print("\n🏆 Inserting award criteria...")
    for award in AWARD_CRITERIA:
        try:
            cursor.execute("""
                INSERT INTO criteria (type, label, short_label, award_id)
                VALUES ('award', %s, %s, %s)
                ON DUPLICATE KEY UPDATE award_id = VALUES(award_id)
            """, (award["label"], award["short_label"], award["award_id"]))
            db.commit()
            cid = get_criteria_id(db, award["label"])
            if cid:
                award_map[award["label"]] = {
                    "criteria_id": cid,
                    "award_id":    award["award_id"],
                    "start_year":  award.get("start_year"),
                }
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
    print("\n⚾ Populating players from stat leaders...")
    qualifying_players = {}

    for crit in STAT_CRITERIA:
        field     = crit["stat_field"]
        threshold = crit["stat_value"]
        label     = crit["label"]
        group     = crit["stat_group"]
        print(f"\n  Fetching leaders for {label}...")

        data = api_get("/stats/leaders", {
            "leaderCategories": field,
            "statGroup":        group,
            "statType":         "career",
            "sportId":          1,
            "limit":            2000,
        })

        if not data:
            continue

        qualifying = set()
        for category in data.get("leagueLeaders", []):
            for leader in category.get("leaders", []):
                try:
                    value  = float(leader.get("value", 0))
                    passes = (value <= threshold) if field == "era" else (value >= threshold)
                    if passes:
                        person    = leader.get("person", {})
                        mlb_id    = person.get("id")
                        full_name = person.get("fullName", "")
                        if mlb_id:
                            qualifying.add(mlb_id)
                            upsert_player(db, mlb_id, full_name)
                except (ValueError, TypeError):
                    continue

        qualifying_players[label] = qualifying
        print(f"    → {len(qualifying)} players qualify")

    return qualifying_players

# ═══════════════════════════════════════════════════════════
# STEP 4 — POPULATE AWARD WINNERS
# ═══════════════════════════════════════════════════════════
def populate_award_winners(db, award_map):
    print("\n🏅 Fetching award winners...")
    award_players = {}
    current_year  = 2024

    for label, info in award_map.items():
        award_id   = info["award_id"]
        start_year = info.get("start_year")
        print(f"\n  Fetching {label} winners...")
        qualifying = set()

        if award_id == "MLBHOF" or start_year is None:
            data = api_get(f"/awards/{award_id}/recipients")
            if data:
                for award_entry in data.get("awards", []):
                    person    = award_entry.get("player", {})
                    mlb_id    = person.get("id")
                    full_name = person.get("fullName", "")
                    if mlb_id:
                        qualifying.add(mlb_id)
                        upsert_player(db, mlb_id, full_name)
        else:
            for year in range(start_year, current_year + 1):
                data = api_get(f"/awards/{award_id}/recipients", {"season": year})
                if not data:
                    continue
                for award_entry in data.get("awards", []):
                    person    = award_entry.get("player", {})
                    mlb_id    = person.get("id")
                    full_name = person.get("fullName", "")
                    if mlb_id:
                        qualifying.add(mlb_id)
                        upsert_player(db, mlb_id, full_name)

        award_players[label] = qualifying
        print(f"    → {len(qualifying)} winners found")

    return award_players

# ═══════════════════════════════════════════════════════════
# STEP 5 — POPULATE TEAM HISTORY PER PLAYER
# ═══════════════════════════════════════════════════════════
def populate_team_history(db, team_map):
    print("\n🏟️  Fetching team roster history...")
    team_players = {}
    current_year = 2024

    for team_label, team_info in team_map.items():
        mlb_team_id = team_info["mlb_team_id"]
        print(f"\n  {team_label}...")
        qualifying = set()

        for year in range(1969, current_year + 1):
            data = api_get(f"/teams/{mlb_team_id}/roster", {
                "rosterType": "40Man",
                "season":     year,
            })
            if not data:
                continue
            for entry in data.get("roster", []):
                person    = entry.get("person", {})
                mlb_id    = person.get("id")
                full_name = person.get("fullName", "")
                if mlb_id:
                    qualifying.add(mlb_id)
                    upsert_player(db, mlb_id, full_name)

        team_players[team_label] = qualifying
        print(f"    → {len(qualifying)} players found")

    return team_players

# ═══════════════════════════════════════════════════════════
# STEP 6 — INSERT PLAYER CRITERIA LINKS
# ═══════════════════════════════════════════════════════════
def insert_player_criteria(db, criteria_label, player_mlb_ids):
    criteria_id = get_criteria_id(db, criteria_label)
    if not criteria_id:
        print(f"    ✗ Criteria not found: '{criteria_label}'")
        return

    cursor = db.cursor()
    cursor.execute("SELECT COUNT(*) FROM mlb_players")
    total_players = cursor.fetchone()[0]
    cursor.close()

    if total_players == 0:
        print(f"    ✗ mlb_players table is empty — skipping '{criteria_label}'")
        return

    inserted = 0
    for mlb_id in player_mlb_ids:
        inserted += insert_player_criteria_row(db, mlb_id, criteria_id)

    print(f"    → Inserted {inserted} player_criteria rows for '{criteria_label}'")

# ═══════════════════════════════════════════════════════════
# STEP 7 — BUILD GRID TEMPLATES
# ═══════════════════════════════════════════════════════════
def get_valid_answers_for_cell(db, row_criteria_id, col_criteria_id):
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

def build_grid_templates(db):
    cursor = db.cursor()
    cursor.execute("SELECT id, type FROM criteria")
    rows     = cursor.fetchall()
    team_ids = [r[0] for r in rows if r[1] == 'team']
    stat_ids = [r[0] for r in rows if r[1] != 'team']
    cursor.close()

    print(f"\n🏗️  Building grid templates ({len(team_ids)} team, {len(stat_ids)} stat/award criteria)...")

    grids_created = 0
    attempts      = 0
    max_attempts  = TARGET_GRIDS * 50

    while grids_created < TARGET_GRIDS and attempts < max_attempts:
        attempts += 1

        grid_type = random.choice(["all_teams", "mixed", "mixed", "mixed"])

        if grid_type == "all_teams":
            if len(team_ids) < 6:
                continue
            selected = random.sample(team_ids, 6)
        else:
            n_teams = random.randint(2, 4)
            n_stats = 6 - n_teams
            if len(team_ids) < n_teams or len(stat_ids) < n_stats:
                continue
            selected = random.sample(team_ids, n_teams) + random.sample(stat_ids, n_stats)
            random.shuffle(selected)

        row_ids = selected[:3]
        col_ids = selected[3:]

        if len(set(row_ids + col_ids)) < 6:
            continue

        valid         = True
        cell_data     = {}
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

        avg_answers = total_answers / 9
        difficulty  = "easy" if avg_answers > 20 else "medium" if avg_answers > 8 else "hard"

        cursor = db.cursor()
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

            for (ri, ci), answers in cell_data.items():
                total = len(answers)
                for rank, (mlb_id, player_name, hs_url) in enumerate(answers):
                    rarity = (rank + 1) / total
                    cursor.execute("""
                        INSERT IGNORE INTO cell_answers
                        (grid_template_id, row_index, col_index, mlb_id,
                         player_name, headshot_url, rarity_score)
                        VALUES (%s, %s, %s, %s, %s, %s, %s)
                    """, (grid_id, ri, ci, mlb_id, player_name, hs_url, rarity))
            db.commit()

            grids_created += 1
            print(f"  ✓ Grid {grids_created}/{TARGET_GRIDS} (difficulty: {difficulty}, avg answers: {avg_answers:.1f})")
        except Exception as e:
            print(f"  ✗ Error inserting grid: {e}")
            db.rollback()
        finally:
            cursor.close()

    print(f"\n  → {grids_created} grid templates created")

# ═══════════════════════════════════════════════════════════
# STEP 8 — CALCULATE RARITY SCORES
# Rarity is based on player accomplishments:
#   More accomplished = lower rarity score (common/easy to guess)
#   Less accomplished = higher rarity score (rare/hard to guess)
# ═══════════════════════════════════════════════════════════
def calculate_rarity(db):
    print("\n🎯 Calculating accomplishment-based rarity scores...")
    cursor = db.cursor()
    try:
        # Score each player based on weighted accomplishments
        cursor.execute("""
            SELECT
                pc.mlb_id,
                SUM(CASE
                    WHEN c.label IN ('Hall of Fame', 'AL MVP', 'NL MVP')
                        THEN 5
                    WHEN c.label IN ('AL Cy Young', 'NL Cy Young',
                                     'World Series MVP', 'World Series Champion')
                        THEN 4
                    WHEN c.label IN ('AL All-Star', 'NL All-Star',
                                     'AL Gold Glove', 'NL Gold Glove',
                                     'AL Silver Slugger', 'NL Silver Slugger')
                        THEN 3
                    WHEN c.label IN ('AL Rookie of the Year', 'NL Rookie of the Year')
                        THEN 2
                    WHEN c.type = 'stat'  THEN 2
                    ELSE 1
                END) AS weighted_score
            FROM player_criteria pc
            JOIN criteria c ON pc.criteria_id = c.id
            GROUP BY pc.mlb_id
        """)
        rows = cursor.fetchall()

        if not rows:
            print("  ✗ No player criteria found")
            return

        max_score = float(max(r[1] for r in rows))
        print(f"  Max accomplishment score: {max_score}")

        # rarity = weighted_score / max_score
        # Most accomplished → rarity near 1.0 (common/easy to guess)
        # Least accomplished → rarity near 0.0 (rare/hard to guess)
        updated = 0
        for mlb_id, weighted_score in rows:
            rarity = round(float(weighted_score) / max_score, 4)
            cursor.execute("""
                UPDATE cell_answers SET rarity_score = %s WHERE mlb_id = %s
            """, (rarity, mlb_id))
            updated += cursor.rowcount

        db.commit()
        print(f"  → Updated rarity scores for {len(rows)} players ({updated} cell rows)")

        # Show sample for verification
        cursor.execute("""
            SELECT p.full_name, ca.rarity_score
            FROM cell_answers ca
            JOIN mlb_players p ON ca.mlb_id = p.mlb_id
            GROUP BY ca.mlb_id, p.full_name, ca.rarity_score
            ORDER BY ca.rarity_score ASC
            LIMIT 5
        """)
        print("\n  Most common (easiest to guess):")
        for row in cursor.fetchall():
            print(f"    {row[0]:<30} rarity={row[1]:.3f}")

        cursor.execute("""
            SELECT p.full_name, ca.rarity_score
            FROM cell_answers ca
            JOIN mlb_players p ON ca.mlb_id = p.mlb_id
            GROUP BY ca.mlb_id, p.full_name, ca.rarity_score
            ORDER BY ca.rarity_score DESC
            LIMIT 5
        """)
        print("\n  Rarest (hardest to guess):")
        for row in cursor.fetchall():
            print(f"    {row[0]:<30} rarity={row[1]:.3f}")

    except Exception as e:
        print(f"  ✗ Error updating rarity: {e}")
        db.rollback()
    finally:
        cursor.close()

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

    # Step 2 — Criteria
    stat_map, award_map = populate_criteria(db)

    # Step 3 — Stat leaders → players + link
    qualifying_players = populate_players_from_leaders(db)
    print("\n🔗 Linking stat players to criteria...")
    for label, player_ids in qualifying_players.items():
        insert_player_criteria(db, label, player_ids)

    # Step 4 — Award winners → players + link
    award_players = populate_award_winners(db, award_map)
    print("\n🔗 Linking award players to criteria...")
    for label, player_ids in award_players.items():
        insert_player_criteria(db, label, player_ids)

    # Step 5 — Team history (slow ~30 min, Ctrl+C to skip)
    print("\n⚠️  Team history step is slow (~30 min). Press Ctrl+C to skip.")
    try:
        team_players = populate_team_history(db, team_map)
        print("\n🔗 Linking team players to criteria...")
        for label, player_ids in team_players.items():
            insert_player_criteria(db, label, player_ids)
    except KeyboardInterrupt:
        print("\n  Skipped team history — re-run separately if needed")

    # Step 6 — Check data before building grids
    cursor = db.cursor()
    cursor.execute("SELECT COUNT(*) FROM player_criteria")
    pc_count = cursor.fetchone()[0]
    cursor.close()

    if pc_count == 0:
        print("\n❌ No player_criteria rows — cannot build grids yet")
        print("   Re-run and let team history complete for full grid support.")
    else:
        print(f"\n✅ {pc_count} player_criteria rows — building grids...")
        build_grid_templates(db)
        calculate_rarity(db)

    db.close()

    print("\n" + "=" * 60)
    print("  ✅ POPULATION COMPLETE")
    print("=" * 60)

if __name__ == "__main__":
    main()