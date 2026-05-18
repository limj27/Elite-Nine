#!/usr/bin/env python3
"""
Elite Nine — Fix Missing Team Links
=====================================
Finds all players with no team links and looks up their career
team history from the MLB Stats API to add the missing links.

Safe to rerun — automatically skips players already fixed.
If it times out, just rerun and it picks up where it left off.

Usage:
    python3 fix_all_missing_links.py
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
MLB_API       = "https://statsapi.mlb.com/api/v1"
REQUEST_DELAY = 0.3

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
def api_get_with_retry(session, url, params=None, retries=3):
    for attempt in range(retries):
        try:
            time.sleep(REQUEST_DELAY)
            resp = session.get(url, params=params, timeout=20)
            resp.raise_for_status()
            return resp
        except Exception as e:
            if attempt == retries - 1:
                print(f"    ✗ Failed after {retries} attempts: {e}")
                return None
            wait = 2 ** attempt
            print(f"    ⚠ Attempt {attempt + 1} failed, retrying in {wait}s...")
            time.sleep(wait)
    return None

def insert_link(db, mlb_id, criteria_id):
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
# MAIN
# ═══════════════════════════════════════════════════════════
def main():
    print("=" * 60)
    print("  ELITE NINE — FIX MISSING TEAM LINKS")
    print(f"  {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)

    db      = get_db()
    session = requests.Session()
    session.headers.update({"User-Agent": "EliteNine/1.0 (personal project)"})

    # Get all players with no team links
    # Safe to rerun — already fixed players won't appear here
    cursor = db.cursor()
    cursor.execute("""
        SELECT p.mlb_id, p.full_name
        FROM mlb_players p
        WHERE NOT EXISTS (
            SELECT 1 FROM player_criteria pc
            JOIN criteria c ON pc.criteria_id = c.id
            WHERE pc.mlb_id = p.mlb_id
            AND c.type = 'team'
        )
        ORDER BY p.full_name
    """)
    players = cursor.fetchall()
    cursor.close()

    if not players:
        print("\n✅ No players with missing team links — nothing to fix!")
        db.close()
        return

    # Build team criteria map: mlb_team_id → criteria_id
    cursor = db.cursor()
    cursor.execute("""
        SELECT mlb_team_id, id
        FROM criteria
        WHERE type = 'team' AND mlb_team_id IS NOT NULL
    """)
    team_criteria = {row[0]: row[1] for row in cursor.fetchall()}
    cursor.close()

    print(f"\n📋 {len(players)} players need team links")
    print(f"   Team criteria loaded: {len(team_criteria)} teams\n")

    fixed       = 0
    not_found   = 0
    total_links = 0

    for i, (mlb_id, full_name) in enumerate(players):
        if i % 50 == 0:
            print(f"  Progress: {i}/{len(players)} ({fixed} fixed, {not_found} not found)...")

        # Pull career stats with year-by-year team splits
        resp = api_get_with_retry(
            session,
            f"{MLB_API}/people/{mlb_id}/stats",
            params={"stats": "yearByYear", "sportId": 1},
        )

        if not resp:
            not_found += 1
            continue

        data       = resp.json()
        stats_list = data.get("stats", [])
        player_links = 0

        for stat_group in stats_list:
            for split in stat_group.get("splits", []):
                team    = split.get("team", {})
                team_id = team.get("id")

                if team_id and team_id in team_criteria:
                    criteria_id   = team_criteria[team_id]
                    player_links += insert_link(db, mlb_id, criteria_id)
                    total_links  += player_links

        if player_links > 0:
            fixed += 1

    db.close()

    print(f"\n{'=' * 60}")
    print(f"  ✅ COMPLETE")
    print(f"  Players fixed:       {fixed}")
    print(f"  Players not found:   {not_found}")
    print(f"  Remaining unfixed:   {len(players) - fixed - not_found}")
    print(f"  Total links added:   {total_links}")
    print(f"  Timestamp: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{'=' * 60}")

    if fixed > 0 or total_links > 0:
        print("\n⚡ Run rebuild_cell_answers.py to update grids")

if __name__ == "__main__":
    main()