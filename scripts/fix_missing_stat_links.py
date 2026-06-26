#!/usr/bin/env python3
"""
Elite Nine — Fix Missing Stat Links
=====================================
For every player in mlb_players, pulls their career stats directly
from the MLB Stats API and links them to any stat criteria they qualify for.

Safe to rerun — INSERT IGNORE skips existing links.

Usage:
    python3 fix_missing_stat_links.py
"""

import os
import time
import requests
import mysql.connector
from datetime import datetime
from dotenv import load_dotenv

load_dotenv()

MLB_API       = "https://statsapi.mlb.com/api/v1"
REQUEST_DELAY = 0.3

STAT_CRITERIA = [
    {"label": ".300+ Career AVG",    "stat_field": "avg",         "stat_value": 0.300, "stat_group": "hitting",  "higher_is_better": True},
    {"label": ".400+ Career OBP",    "stat_field": "obp",         "stat_value": 0.400, "stat_group": "hitting",  "higher_is_better": True},
    {"label": "500+ Career HR",      "stat_field": "homeRuns",    "stat_value": 500,   "stat_group": "hitting",  "higher_is_better": True},
    {"label": "400+ Career HR",      "stat_field": "homeRuns",    "stat_value": 400,   "stat_group": "hitting",  "higher_is_better": True},
    {"label": "300+ Career HR",      "stat_field": "homeRuns",    "stat_value": 300,   "stat_group": "hitting",  "higher_is_better": True},
    {"label": "3000+ Career Hits",   "stat_field": "hits",        "stat_value": 3000,  "stat_group": "hitting",  "higher_is_better": True},
    {"label": "2000+ Career Hits",   "stat_field": "hits",        "stat_value": 2000,  "stat_group": "hitting",  "higher_is_better": True},
    {"label": "300+ Career SB",      "stat_field": "stolenBases", "stat_value": 300,   "stat_group": "hitting",  "higher_is_better": True},
    {"label": "1500+ Career RBI",    "stat_field": "rbi",         "stat_value": 1500,  "stat_group": "hitting",  "higher_is_better": True},
    {"label": "1000+ Career RBI",    "stat_field": "rbi",         "stat_value": 1000,  "stat_group": "hitting",  "higher_is_better": True},
    {"label": "200+ Career Wins",    "stat_field": "wins",        "stat_value": 200,   "stat_group": "pitching", "higher_is_better": True},
    {"label": "150+ Career Wins",    "stat_field": "wins",        "stat_value": 150,   "stat_group": "pitching", "higher_is_better": True},
    {"label": "3000+ Career K",      "stat_field": "strikeOuts",  "stat_value": 3000,  "stat_group": "pitching", "higher_is_better": True},
    {"label": "2000+ Career K",      "stat_field": "strikeOuts",  "stat_value": 2000,  "stat_group": "pitching", "higher_is_better": True},
    {"label": "Sub-3.00 Career ERA", "stat_field": "era",         "stat_value": 3.00,  "stat_group": "pitching", "higher_is_better": False},
    {"label": "300+ Career Saves",   "stat_field": "saves",       "stat_value": 300,   "stat_group": "pitching", "higher_is_better": True},
]

def get_db():
    return mysql.connector.connect(
        host=os.getenv("DB_HOST", "localhost"),
        port=int(os.getenv("DB_PORT", 3306)),
        user=os.getenv("DB_USER", "gameuser"),
        password=os.getenv("DB_PASSWORD", "gamepassword"),
        database=os.getenv("DB_NAME", "baseball_game"),
        autocommit=False,
    )

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

def main():
    print("=" * 60)
    print("  ELITE NINE — FIX MISSING STAT LINKS")
    print(f"  {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)

    db      = get_db()
    session = requests.Session()
    session.headers.update({"User-Agent": "EliteNine/1.0"})

    # Load criteria IDs
    cursor = db.cursor()
    cursor.execute("SELECT id, label FROM criteria WHERE type = 'stat'")
    criteria_map = {row[1]: row[0] for row in cursor.fetchall()}
    cursor.close()

    print(f"\n📊 Loaded {len(criteria_map)} stat criteria")

    # Get all players
    cursor = db.cursor()
    cursor.execute("SELECT mlb_id, full_name FROM mlb_players ORDER BY full_name")
    players = cursor.fetchall()
    cursor.close()

    print(f"⚾ Processing {len(players)} players...\n")

    total_new_links = 0
    errors          = 0

    for i, (mlb_id, full_name) in enumerate(players):
        if i % 100 == 0:
            print(f"  Progress: {i}/{len(players)} ({total_new_links} new links so far)...")

        time.sleep(REQUEST_DELAY)

        # Pull career hitting stats
        hitting_links = 0
        pitching_links = 0

        try:
            resp = session.get(
                f"{MLB_API}/people/{mlb_id}/stats",
                params={"stats": "career", "group": "hitting", "sportId": 1},
                timeout=15,
            )
            if resp.ok:
                for stat_group in resp.json().get("stats", []):
                    for split in stat_group.get("splits", []):
                        stats = split.get("stat", {})
                        for crit in STAT_CRITERIA:
                            if crit["stat_group"] != "hitting":
                                continue
                            crit_id = criteria_map.get(crit["label"])
                            if not crit_id:
                                continue
                            raw = stats.get(crit["stat_field"])
                            if raw is None:
                                continue
                            try:
                                value = float(raw)
                            except (ValueError, TypeError):
                                continue
                            qualifies = (value >= crit["stat_value"]) if crit["higher_is_better"] \
                                       else (value <= crit["stat_value"])
                            if qualifies:
                                hitting_links += insert_link(db, mlb_id, crit_id)
        except requests.RequestException:
            errors += 1

        time.sleep(REQUEST_DELAY)

        try:
            resp = session.get(
                f"{MLB_API}/people/{mlb_id}/stats",
                params={"stats": "career", "group": "pitching", "sportId": 1},
                timeout=15,
            )
            if resp.ok:
                for stat_group in resp.json().get("stats", []):
                    for split in stat_group.get("splits", []):
                        stats = split.get("stat", {})
                        for crit in STAT_CRITERIA:
                            if crit["stat_group"] != "pitching":
                                continue
                            crit_id = criteria_map.get(crit["label"])
                            if not crit_id:
                                continue
                            raw = stats.get(crit["stat_field"])
                            if raw is None:
                                continue
                            try:
                                value = float(raw)
                            except (ValueError, TypeError):
                                continue
                            qualifies = (value >= crit["stat_value"]) if crit["higher_is_better"] \
                                       else (value <= crit["stat_value"])
                            if qualifies:
                                pitching_links += insert_link(db, mlb_id, crit_id)
        except requests.RequestException:
            errors += 1

        new_links = hitting_links + pitching_links
        if new_links > 0:
            print(f"  ✓ {full_name}: +{new_links} new stat links")
        total_new_links += new_links

    db.close()

    print(f"\n{'=' * 60}")
    print(f"  ✅ COMPLETE")
    print(f"  Players processed:  {len(players)}")
    print(f"  New stat links:     {total_new_links}")
    print(f"  API errors:         {errors}")
    print(f"  Timestamp: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{'=' * 60}")
    print("\n⚡ Run rebuild_cell_answers.py to update grids")

if __name__ == "__main__":
    main()