#!/usr/bin/env python3
"""
Elite Nine — Rebuild Cell Answers Script
==========================================
Rebuilds cell_answers for all existing grid templates based on
current player_criteria data. Run this after updating rosters
or adding new players to ensure grids reflect the latest data.

Usage:
    python3 rebuild_cell_answers.py
"""

import os
import mysql.connector
from datetime import datetime
from dotenv import load_dotenv

load_dotenv()

# ═══════════════════════════════════════════════════════════
# CONFIG
# ═══════════════════════════════════════════════════════════
MIN_ANSWERS_PER_CELL = 3

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
def get_valid_answers_for_cell(cursor, row_criteria_id, col_criteria_id):
    """Find all players who satisfy BOTH row and column criteria."""
    cursor.execute("""
        SELECT p.mlb_id, p.full_name, COALESCE(p.headshot_url, '')
        FROM mlb_players p
        JOIN player_criteria pc1 ON p.mlb_id = pc1.mlb_id AND pc1.criteria_id = %s
        JOIN player_criteria pc2 ON p.mlb_id = pc2.mlb_id AND pc2.criteria_id = %s
    """, (row_criteria_id, col_criteria_id))
    return cursor.fetchall()

def calculate_rarity(db):
    """
    Recalculate rarity scores based on player accomplishments.
    More accomplished = lower rarity score (common/easy to guess)
    Less accomplished = higher rarity score (rare/hard to guess)
    """
    cursor = db.cursor()
    try:
        # Score each player by weighted accomplishments
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

        max_score = max(r[1] for r in rows)

        # rarity = 1 - (weighted_score / max_score)
        updated = 0
        for mlb_id, weighted_score in rows:
            rarity = round(1.0 - (weighted_score / max_score), 4)
            cursor.execute("""
                UPDATE cell_answers SET rarity_score = %s WHERE mlb_id = %s
            """, (rarity, mlb_id))
            updated += cursor.rowcount

        db.commit()
        print(f"  → Updated rarity for {len(rows)} players ({updated} cell rows)")

        # Show sample for verification
        cursor.execute("""
            SELECT p.full_name, ca.rarity_score
            FROM cell_answers ca
            JOIN mlb_players p ON ca.mlb_id = p.mlb_id
            GROUP BY ca.mlb_id, p.full_name, ca.rarity_score
            ORDER BY ca.rarity_score ASC LIMIT 5
        """)
        print("\n  Most common (easiest to guess):")
        for row in cursor.fetchall():
            print(f"    {row[0]:<30} rarity={row[1]:.3f}")

        cursor.execute("""
            SELECT p.full_name, ca.rarity_score
            FROM cell_answers ca
            JOIN mlb_players p ON ca.mlb_id = p.mlb_id
            GROUP BY ca.mlb_id, p.full_name, ca.rarity_score
            ORDER BY ca.rarity_score DESC LIMIT 5
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
    print("  ELITE NINE — REBUILD CELL ANSWERS")
    print(f"  {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)

    db = get_db()
    cursor = db.cursor()

    # Get all grid templates
    cursor.execute("""
        SELECT id, row_criteria_1, row_criteria_2, row_criteria_3,
               col_criteria_1, col_criteria_2, col_criteria_3, difficulty
        FROM grid_templates
        WHERE active = TRUE
        ORDER BY id
    """)
    grids = cursor.fetchall()
    print(f"\n📋 Found {len(grids)} grid templates to rebuild")

    # Clear existing cell answers
    print("🗑️  Clearing existing cell_answers...")
    cursor.execute("DELETE FROM cell_answers")
    db.commit()

    total_answers  = 0
    grids_ok       = 0
    grids_degraded = 0  # grids where some cells now have fewer than MIN answers

    for grid in grids:
        grid_id   = grid[0]
        row_ids   = [grid[1], grid[2], grid[3]]
        col_ids   = [grid[4], grid[5], grid[6]]
        difficulty = grid[7]

        grid_answers    = 0
        grid_min_cell   = float('inf')
        cell_data       = {}

        for ri, row_c in enumerate(row_ids):
            for ci, col_c in enumerate(col_ids):
                answers = get_valid_answers_for_cell(cursor, row_c, col_c)
                cell_data[(ri, ci)] = answers
                grid_answers += len(answers)
                grid_min_cell = min(grid_min_cell, len(answers))

        # Insert new cell answers
        for (ri, ci), answers in cell_data.items():
            total = len(answers)
            for rank, (mlb_id, player_name, hs_url) in enumerate(answers):
                rarity = (rank + 1) / total if total > 0 else 0.5
                try:
                    cursor.execute("""
                        INSERT IGNORE INTO cell_answers
                        (grid_template_id, row_index, col_index, mlb_id,
                         player_name, headshot_url, rarity_score)
                        VALUES (%s, %s, %s, %s, %s, %s, %s)
                    """, (grid_id, ri, ci, mlb_id, player_name, hs_url, rarity))
                except Exception as e:
                    print(f"  ✗ Error inserting cell answer: {e}")

        db.commit()
        total_answers += grid_answers

        if grid_min_cell < MIN_ANSWERS_PER_CELL:
            grids_degraded += 1
            print(f"  ⚠️  Grid {grid_id} ({difficulty}): min cell has {grid_min_cell} answers — consider regenerating")
        else:
            grids_ok += 1

        avg = grid_answers / 9 if grid_answers > 0 else 0
        print(f"  ✓ Grid {grid_id} ({difficulty}): {grid_answers} answers, avg {avg:.1f}/cell, min {grid_min_cell}/cell")

    # Recalculate rarity scores
    print("\n🎯 Recalculating rarity scores...")
    calculate_rarity(db)

    # Update min_answers on grid_templates
    print("📊 Updating grid template stats...")
    cursor.execute("""
        UPDATE grid_templates gt
        JOIN (
            SELECT grid_template_id, COUNT(*) as total
            FROM cell_answers
            GROUP BY grid_template_id
        ) counts ON gt.id = counts.grid_template_id
        SET gt.min_answers = counts.total
    """)
    db.commit()

    cursor.close()
    db.close()

    print(f"\n{'=' * 60}")
    print(f"  ✅ REBUILD COMPLETE")
    print(f"  Grids rebuilt:          {len(grids)}")
    print(f"  Grids healthy:          {grids_ok}")
    print(f"  Grids degraded:         {grids_degraded}")
    print(f"  Total cell answers:     {total_answers}")
    print(f"  Avg answers per grid:   {total_answers / len(grids):.1f}" if grids else "")
    print(f"  Timestamp: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{'=' * 60}")

    if grids_degraded > 0:
        print(f"\n⚠️  {grids_degraded} grids have cells with fewer than {MIN_ANSWERS_PER_CELL} valid answers.")
        print("   These grids may be unsolvable. Consider running populate.py")
        print("   to regenerate grid templates with the updated player data.")

if __name__ == "__main__":
    main()