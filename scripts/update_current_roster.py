#!/usr/bin/env python3
"""
Elite Nine — Current Roster & Award Update Script
===================================================
Updates player-team links for 2025-2026 and award winners for 2024-2025.
Run this regularly to keep data current.

Usage:
    python3 update_current_rosters.py

Cron (daily at 4am on the Pi):
    0 4 * * * cd /path/to/Elite-Nine/scripts && python3 update_current_rosters.py
"""

import os
import time
import requests
import mysql.connector
from datetime import datetime
from dotenv import load_dotenv

load_dotenv()

# ═══════════════════════════════════════════════════════════
# CONFIG
# ═══════════════════════════════════════════════════════════
MLB_API            = "https://statsapi.mlb.com/api/v1"
REQUEST_DELAY      = 0.25

# Only pull recent years — 2023/2024 already populated by populate.py
UPDATE_YEARS       = [2025, 2026]
ROSTER_TYPES       = ["40Man", "active"]

# 2024 awards were announced after populate.py ran
AWARD_UPDATE_YEARS = [2024, 2025]

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
# HELPERS
# ═══════════════════════════════════════════════════════════
def headshot_url(mlb_id):
    return (
        f"https://img.mlbstatic.com/mlb-photos/image/upload/"
        f"d_people:generic:headshot:67:current.png/w_213,q_auto:best/"
        f"v1/people/{mlb_id}/headshot/67/current"
    )

def upsert_player(db, mlb_id, full_name, position=None):
    cursor = db.cursor()
    try:
        cursor.execute("""
            INSERT INTO mlb_players (mlb_id, full_name, position, headshot_url)
            VALUES (%s, %s, %s, %s)
            ON DUPLICATE KEY UPDATE
                full_name    = VALUES(full_name),
                headshot_url = VALUES(headshot_url)
        """, (mlb_id, full_name, position, headshot_url(mlb_id)))
        db.commit()
    except Exception:
        db.rollback()
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

# ═══════════════════════════════════════════════════════════
# TEAM ROSTER UPDATE
# ═══════════════════════════════════════════════════════════
def update_rosters(db, session):
    cursor = db.cursor()
    cursor.execute("""
        SELECT id, label, mlb_team_id
        FROM criteria
        WHERE type = 'team' AND mlb_team_id IS NOT NULL
        ORDER BY label
    """)
    teams = cursor.fetchall()
    cursor.close()

    print(f"\n⚾ Updating {len(teams)} team rosters for {UPDATE_YEARS}...")

    total_new_players = 0
    total_new_links   = 0

    for criteria_id, team_label, mlb_team_id in teams:
        print(f"  {team_label}...")
        team_new_players = 0
        team_new_links   = 0

        for year in UPDATE_YEARS:
            for roster_type in ROSTER_TYPES:
                time.sleep(REQUEST_DELAY)
                try:
                    resp = session.get(
                        f"{MLB_API}/teams/{mlb_team_id}/roster",
                        params={"rosterType": roster_type, "season": year},
                        timeout=10,
                    )
                    if not resp.ok:
                        continue

                    for entry in resp.json().get("roster", []):
                        person    = entry.get("person", {})
                        mlb_id    = person.get("id")
                        full_name = person.get("fullName", "")
                        position  = entry.get("position", {}).get("abbreviation")

                        if not mlb_id:
                            continue

                        # Check if player is new
                        cur = db.cursor()
                        cur.execute("SELECT mlb_id FROM mlb_players WHERE mlb_id = %s", (mlb_id,))
                        is_new = cur.fetchone() is None
                        cur.close()

                        upsert_player(db, mlb_id, full_name, position)
                        if is_new:
                            team_new_players += 1

                        team_new_links += insert_player_criteria_row(db, mlb_id, criteria_id)

                except requests.RequestException as e:
                    print(f"    ✗ API error ({year} {roster_type}): {e}")
                    continue

        total_new_players += team_new_players
        total_new_links   += team_new_links

        if team_new_links > 0:
            print(f"    → +{team_new_players} new players, +{team_new_links} new links")
        else:
            print(f"    → No new data")

    return total_new_players, total_new_links

# ═══════════════════════════════════════════════════════════
# AWARD UPDATE
# ═══════════════════════════════════════════════════════════
def update_awards(db, session):
    print(f"\n🏆 Updating award winners for {AWARD_UPDATE_YEARS}...")

    cursor = db.cursor()
    cursor.execute("""
        SELECT id, label, award_id
        FROM criteria
        WHERE type = 'award'
        AND award_id IS NOT NULL
        AND award_id != 'MLBHOF'
    """)
    awards = cursor.fetchall()
    cursor.close()

    total_new = 0

    for criteria_id, label, award_id in awards:
        new_links = 0

        for year in AWARD_UPDATE_YEARS:
            time.sleep(REQUEST_DELAY)
            try:
                resp = session.get(
                    f"{MLB_API}/awards/{award_id}/recipients",
                    params={"season": year},
                    timeout=10,
                )
                if not resp.ok:
                    continue

                for award_entry in resp.json().get("awards", []):
                    person    = award_entry.get("player", {})
                    mlb_id    = person.get("id")
                    full_name = person.get("fullName", "")
                    if mlb_id:
                        upsert_player(db, mlb_id, full_name)
                        new_links += insert_player_criteria_row(db, mlb_id, criteria_id)

            except requests.RequestException:
                continue

        if new_links > 0:
            print(f"  ✓ {label} → +{new_links} new links")

        total_new += new_links

    print(f"  → {total_new} total new award links")
    return total_new

# ═══════════════════════════════════════════════════════════
# MAIN
# ═══════════════════════════════════════════════════════════
def main():
    print("=" * 60)
    print("  ELITE NINE — ROSTER & AWARD UPDATE")
    print(f"  {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)

    db      = get_db()
    session = requests.Session()
    session.headers.update({"User-Agent": "EliteNine/1.0 (personal project)"})

    # Update team rosters
    new_players, new_roster_links = update_rosters(db, session)

    # Update award winners
    new_award_links = update_awards(db, session)

    db.close()

    print(f"\n{'=' * 60}")
    print(f"  ✅ UPDATE COMPLETE")
    print(f"  New players added:      {new_players}")
    print(f"  New roster links:       {new_roster_links}")
    print(f"  New award links:        {new_award_links}")
    print(f"  Timestamp: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{'=' * 60}")

if __name__ == "__main__":
    main()