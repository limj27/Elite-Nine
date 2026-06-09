// ═══════════════════════════════════════════════════════════
// GAME
// Game screen setup, grid rendering, ready system,
// MLB player search, and move submission
// ═══════════════════════════════════════════════════════════

// 9 cells: { owner: null|'p1'|'p2', player: null, rarity: 0 }
let gridState = [];
let selectedCell = null;
let searchTimeout = null;

// ── Screen setup ────────────────────────────────────────────

function enterGameScreen(roomName) {
  document.getElementById('game-room-name').textContent = roomName || '';
  document.getElementById('my-name').textContent        = State.myUsername;
  document.getElementById('my-avatar').textContent      = State.myUsername?.[0]?.toUpperCase() || '?';

  // Reset to waiting state
  document.getElementById('waiting-state').style.display = 'block';
  document.getElementById('grid-wrap').style.display     = 'none';
  document.getElementById('ready-section').style.display = 'flex';
  document.getElementById('opp-name').textContent        = 'Waiting...';
  document.getElementById('opp-avatar').textContent      = '?';
  document.getElementById('start-btn').disabled          = true;
  document.getElementById('ready-btn').textContent       = 'Mark Ready';

  State.myReady     = false;
  State.oppReady    = false;
  State.gameStarted = false;

  updateReadyUI();
  showScreen('game');
}

function updatePlayerColors() {
  const myClass  = State.playerIndex === 0 ? 'p1' : 'p2';
  const oppClass = State.playerIndex === 0 ? 'p2' : 'p1';

  const myAvatar  = document.getElementById('my-avatar');
  const oppAvatar = document.getElementById('opp-avatar');

  // Remove existing color classes
  myAvatar.classList.remove('p1', 'p2');
  oppAvatar.classList.remove('p1', 'p2');

  // Apply correct colors
  myAvatar.classList.add(myClass);
  oppAvatar.classList.add(oppClass);
}

// ── Player joined / left ────────────────────────────────────

function onPlayerJoined(payload) {
  if (payload.playerId === State.myClientId) return;  // skip if it's yourself

  const name = payload.username || ('Player ' + payload.userId);
  document.getElementById('opp-name').textContent   = name;
  document.getElementById('opp-avatar').textContent = name[0]?.toUpperCase() || '?';
  showToast(name + ' joined the room', 'success');
}

function onPlayerLeft(payload) {
  if (payload.playerId === State.myClientId) return;

  document.getElementById('opp-name').textContent   = 'Waiting...';
  document.getElementById('opp-avatar').textContent = '?';
  State.oppReady = false;
  document.getElementById('start-btn').disabled = true;
  updateReadyUI();
  showToast('Opponent left the room', 'error');
}

// ── Ready system ─────────────────────────────────────────────

function handleReady() {
  State.myReady = !State.myReady;
  wsSend('player_ready', { ready: State.myReady });
  document.getElementById('ready-btn').textContent = State.myReady ? 'Cancel Ready' : 'Mark Ready';
  updateReadyUI();
}

function onPlayerReady(payload) {
  if (payload.playerId === State.myClientId) return;

  State.oppReady = payload.ready;
  updateReadyUI();
  showToast('Opponent is ' + (State.oppReady ? 'ready!' : 'not ready'), State.oppReady ? 'success' : '');
}

function updateReadyUI() {
  document.getElementById('my-ready-dot').classList.toggle('ready', State.myReady);
  document.getElementById('my-ready-label').textContent  = 'You: '      + (State.myReady  ? 'ready ✓' : 'not ready');
  document.getElementById('opp-ready-dot').classList.toggle('ready', State.oppReady);
  document.getElementById('opp-ready-label').textContent = 'Opponent: ' + (State.oppReady ? 'ready ✓' : 'waiting');
}

// ── Start game ───────────────────────────────────────────────

function handleStartGame() {
  if (!State.isCreator) return;
  wsSend('start_game', {});
}

// ── Game state updates ───────────────────────────────────────

function onGameState(payload) {
  if (!State.gameStarted) {
    State.gameStarted = true;
    document.getElementById('waiting-state').style.display = 'none';
    document.getElementById('grid-wrap').style.display     = 'flex';
    document.getElementById('ready-section').style.display = 'none';
    buildGrid();
  }

  // Check for game over — must come before turn update
  if (payload?.game?.status === 'completed' && payload?.game?.winner_id) {
    if (payload?.grid) updateGridFromState(payload.grid);
    setTimeout(() => showWinScreen(payload.game.winner_id), 500);
    return;
  }

  if (payload?.game?.current_turn !== undefined) {
    updateTurnBar(payload.game.current_turn);
  }

  if (payload?.grid) {
    updateGridFromState(payload.grid);
  }
}


function onMoveMade(payload) {
  if (payload?.cell_index === undefined) return;

  gridState[payload.cell_index] = {
    owner:  payload.owner,
    player: payload.player,
    rarity: payload.rarity || 0,
  };
  renderCell(payload.cell_index);
}

function showTurnNotification() {
  // Remove any existing notification
  const existing = document.getElementById('turn-notification');
  if (existing) existing.remove();

  const notif = document.createElement('div');
  notif.id = 'turn-notification';
  notif.style.cssText = `
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%) scale(0.8);
    background: var(--blue);
    color: #fff;
    font-family: var(--font-display);
    font-size: 48px;
    letter-spacing: 3px;
    padding: 24px 48px;
    border-radius: 16px;
    z-index: 400;
    opacity: 0;
    transition: all 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
    pointer-events: none;
    text-align: center;
  `;
  notif.textContent = "YOUR TURN";
  document.body.appendChild(notif);

  // Animate in
  requestAnimationFrame(() => {
    notif.style.opacity = '1';
    notif.style.transform = 'translate(-50%, -50%) scale(1)';
  });

  // Animate out after 1.5 seconds
  setTimeout(() => {
    notif.style.opacity = '0';
    notif.style.transform = 'translate(-50%, -50%) scale(0.8)';
    setTimeout(() => notif.remove(), 200);
  }, 1500);
}

function updateTurnBar(currentTurn) {
  const wasMyTurn = State.myTurn;
  State.myTurn = currentTurn === State.playerIndex;
  const text = State.myTurn ? 'Your turn' : "Opponent's turn";
  document.getElementById('turn-text').textContent = text;
  document.getElementById('turn-bar').style.borderColor = State.myTurn
    ? 'rgba(59,130,246,0.5)'
    : 'rgba(239,68,68,0.3)';

  // Show notification only when turn switches TO you (not on initial load)
  if (State.myTurn && !wasMyTurn && State.gameStarted) {
    showTurnNotification();
  }
}

function updateGridFromState(grid) {
  for (let row = 0; row < 3; row++) {
    for (let col = 0; col < 3; col++) {
      const idx  = row * 3 + col;
      const move = grid[row][col];

      if (!move) {
        gridState[idx] = { owner: null, player: null, rarity: 0 };
      } else {
        // Find which player index made this move
        const movePlayerIndex = State.players?.findIndex(p => p.user_id === move.user_id);
        gridState[idx] = {
          owner:  movePlayerIndex === 0 ? 'p1' : 'p2',
          player: {
            fullName: move.player_name || move.player_answer,
            headshot: move.headshot || '',
          },
          rarity: 0,
        };
      }
      renderCell(idx);
    }
  }
}

// ── Leave room ───────────────────────────────────────────────

function handleLeaveRoom() {
  wsSend('leave_room', {});
  State.currentRoom  = null;
  State.gameStarted  = false;
  State.myReady      = false;
  State.oppReady     = false;
  State.playerIndex  = 0;
  const overlay = document.getElementById('win-overlay');
  if (overlay) overlay.remove();
  const historySection = document.getElementById('cell-history-section');
  if (historySection) historySection.remove();
  showScreen('lobby');
  requestRoomList();
}

// ═══════════════════════════════════════════════════════════
// GRID
// ═══════════════════════════════════════════════════════════

function buildGrid() {
  const grid = document.getElementById('the-grid');
  grid.innerHTML = '';
  gridState = Array(9).fill(null).map(() => ({ owner: null, player: null, rarity: 0 }));

  for (let row = 0; row < 3; row++) {
    const rowEl = document.createElement('div');
    rowEl.className = 'grid-row';

    for (let col = 0; col < 3; col++) {
      const idx  = row * 3 + col;
      const cell = document.createElement('div');
      cell.className = 'grid-cell';
      cell.id        = 'cell-' + idx;
      cell.onclick   = () => onCellClick(idx);
      cell.innerHTML = `<span class="cell-empty-icon">+</span>`;
      rowEl.appendChild(cell);
    }

    grid.appendChild(rowEl);
  }
}

function renderCell(idx) {
  const el    = document.getElementById('cell-' + idx);
  if (!el) return;

  const state = gridState[idx];
  el.className = 'grid-cell' + (state.owner ? ' ' + state.owner : '');

  if (!state.player) {
    el.innerHTML = `<span class="cell-empty-icon">+</span>`;
    return;
  }

  const imgSrc   = state.player.headshot || '';
  const imgEl    = imgSrc
    ? `<img class="cell-player-img" src="${imgSrc}" onerror="this.style.display='none'">`
    : `<div style="width:72px;height:72px;border-radius:50%;background:var(--surface2);display:flex;align-items:center;justify-content:center;font-size:24px;">⚾</div>`;
  const ownerBar = state.owner ? `<div class="cell-owner-bar ${state.owner}"></div>` : '';

  el.innerHTML = `
    <div class="cell-content">
      ${imgEl}
      <div class="cell-player-name">${state.player.fullName || state.player.name || ''}</div>
      <div class="cell-rarity">${state.rarity ? (state.rarity * 100).toFixed(1) + '% rare' : ''}</div>
    </div>
    ${ownerBar}`;
}

function onCellClick(idx) {
  if (!State.gameStarted) return;

  // Show history first
  showCellHistory(idx);

  if (!State.myTurn) {
    showToast("It's not your turn", 'error');
    return;
  }

  selectedCell = idx;

  // Close history panel before opening search
  openSearchModal();
}

function renderGridHeaders() {
    if (!State.gridTemplate) return;

    const { rowCriteria, colCriteria } = State.gridTemplate;

    colCriteria.forEach((crit, i) => {
        const el = document.getElementById('col-header-' + i);
        if (!el) return;
        if (crit.type === 'team' && crit.mlb_team_id) {
            el.innerHTML = `
                <img 
                    src="https://www.mlbstatic.com/team-logos/${crit.mlb_team_id}.svg"
                    alt="${crit.short_label}"
                    style="width:48px;height:48px;object-fit:contain;"
                    onerror="this.outerHTML='<span>${crit.short_label}</span>'"
                >`;
        } else {
            el.textContent = crit.short_label || crit.label;
        }
    });

    rowCriteria.forEach((crit, i) => {
        const el = document.getElementById('row-header-' + i);
        if (!el) return;
        if (crit.type === 'team' && crit.mlb_team_id) {
            el.innerHTML = `
                <img
                    src="https://www.mlbstatic.com/team-logos/${crit.mlb_team_id}.svg"
                    alt="${crit.short_label}"
                    style="width:48px;height:48px;object-fit:contain;"
                    onerror="this.outerHTML='<span>${crit.short_label}</span>'"
                >`;
        } else {
            el.textContent = crit.short_label || crit.label;
        }
    });
}

function showCellHistory(idx) {
  const row = Math.floor(idx / 3);
  const col = idx % 3;
  const history = State.cellHistory?.[row]?.[col];

  // Remove existing panel
  const existing = document.getElementById('cell-history-panel');
  if (existing) existing.remove();

  const rowLabel = State.gridTemplate?.rowCriteria?.[row]?.short_label || `Row ${row + 1}`;
  const colLabel = State.gridTemplate?.colCriteria?.[col]?.short_label || `Col ${col + 1}`;

  const panel = document.createElement('div');
  panel.id = 'cell-history-panel';
  panel.style.cssText = `
    position: fixed;
    top: 25%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: var(--surface);
    border: 1px solid var(--border2);
    border-radius: 14px;
    padding: 20px;
    z-index: 1000;
    min-width: 320px;
    max-width: 420px;
    max-height: 320px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
  `;

  const hasHistory = history && history.length > 0;

  panel.innerHTML = `
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px;flex-shrink:0;">
      <div>
        <div style="font-family:var(--font-display);font-size:18px;letter-spacing:1px;">
          Cell History
        </div>
        <div style="font-size:12px;color:var(--text2);margin-top:2px;">
          ${rowLabel} × ${colLabel}
        </div>
      </div>
      <button onclick="document.getElementById('cell-history-panel').remove()"
        style="background:none;border:none;color:var(--text2);font-size:20px;cursor:pointer;line-height:1;">✕</button>
    </div>
    <div style="overflow-y:auto;flex:1;display:flex;flex-direction:column;gap:8px;">
      ${hasHistory ? history.map(attempt => `
        <div style="
          padding:10px 12px;
          background:${attempt.valid ? 'rgba(34,197,94,0.08)' : 'rgba(239,68,68,0.08)'};
          border:1px solid ${attempt.valid ? 'rgba(34,197,94,0.2)' : 'rgba(239,68,68,0.2)'};
          border-radius:8px;
          flex-shrink:0;
        ">
          <div style="display:flex;align-items:center;justify-content:space-between;">
            <div style="font-size:14px;font-weight:600;color:var(--text);">
              ${attempt.player_name}
            </div>
            <div style="font-size:12px;font-weight:600;color:${attempt.valid ? 'var(--green)' : 'var(--red)'};">
              ${attempt.valid ? '✓ Valid' : '✗ Invalid'}
            </div>
          </div>
          <div style="font-size:11px;color:var(--text2);margin-top:3px;">
            by ${attempt.username}
          </div>
        </div>
      `).join('') : `
        <div style="text-align:center;padding:24px;color:var(--text3);font-size:14px;">
          No attempts yet for this cell
        </div>
      `}
    </div>
  `;

  document.body.appendChild(panel);

  // Close when clicking outside
  setTimeout(() => {
    document.addEventListener('click', function closePanel(e) {
      if (!panel.contains(e.target)) {
        panel.remove();
        document.removeEventListener('click', closePanel);
      }
    });
  }, 100);
}

// ═══════════════════════════════════════════════════════════
// MLB PLAYER SEARCH
// Uses the free MLB Stats API — no key required
// ═══════════════════════════════════════════════════════════

function openSearchModal() {
  document.getElementById('player-search-input').value = '';
  document.getElementById('search-results').innerHTML  = '<div class="search-empty">Type a player name to search</div>';
  document.getElementById('search-modal').classList.add('show');
  setTimeout(() => document.getElementById('player-search-input').focus(), 100);
}

function closeSearchModal() {
  document.getElementById('search-modal').classList.remove('show');
  selectedCell = null;
}

function handlePlayerSearch(query) {
  clearTimeout(searchTimeout);

  if (query.length < 2) {
    document.getElementById('search-results').innerHTML = '<div class="search-empty">Type a player name to search</div>';
    return;
  }

  document.getElementById('search-results').innerHTML = '<div class="search-loading"><div class="spinner"></div></div>';
  searchTimeout = setTimeout(() => searchMLBPlayers(query), 350);
}

async function searchMLBPlayers(query) {
  try {
    const resp = await fetch(`https://statsapi.mlb.com/api/v1/people/search?names=${encodeURIComponent(query)}&sportId=1`);
    const data = await resp.json();
    renderSearchResults(data.people || []);
  } catch {
    document.getElementById('search-results').innerHTML = '<div class="search-empty">Search failed. Try again.</div>';
  }
}

function renderSearchResults(players) {
  const el = document.getElementById('search-results');

  if (!players.length) {
    el.innerHTML = '<div class="search-empty">No players found</div>';
    return;
  }

  el.innerHTML = players.slice(0, 15).map(p => {
    const headshot = `https://img.mlbstatic.com/mlb-photos/image/upload/d_people:generic:headshot:67:current.png/w_213,q_auto:best/v1/people/${p.id}/headshot/67/current`;
    const pos      = p.primaryPosition?.abbreviation || '';
    const team     = p.currentTeam?.name || '';
    const meta     = [pos, team].filter(Boolean).join(' · ');

    // Encode player data safely as a data attribute to avoid inline JSON escaping issues
    const encoded = encodeURIComponent(JSON.stringify({ id: p.id, fullName: p.fullName, headshot }));

    return `
      <div class="search-result-item" onclick="selectPlayer('${encoded}')">
        <img class="search-result-img" src="${headshot}"
          onerror="this.src='data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 40 40%22><circle cx=%2220%22 cy=%2220%22 r=%2220%22 fill=%22%231a2235%22/><text x=%2220%22 y=%2226%22 text-anchor=%22middle%22 fill=%22%2394a3b8%22 font-size=%2218%22>⚾</text></svg>'">
        <div class="search-result-info">
          <div class="search-result-name">${p.fullName}</div>
          <div class="search-result-meta">${meta}</div>
        </div>
      </div>`;
  }).join('');
}

function selectPlayer(encoded) {
  let player;
  try { player = JSON.parse(decodeURIComponent(encoded)); }
  catch { return; }

  console.log('sending make_move with room_id:', State.currentRoom?.room_id);  // add this

  wsSend('make_move', {
    room_id:         State.currentRoom?.room_id,
    row:             Math.floor(selectedCell / 3),
    col:             selectedCell % 3,
    answer:          player.fullName,
    player_id:       player.id,
    player_name:     player.fullName,
    player_headshot: player.headshot,
  });

  closeSearchModal();
}

// ═══════════════════════════════════════════════════════════
// Win Screen
// ═══════════════════════════════════════════════════════════
function showWinScreen(winnerId) {
  const isWinner = State.players?.[State.playerIndex]?.user_id === winnerId;
  const message  = isWinner ? '🏆 You Win!' : '😔 You Lose!';
  const color    = isWinner ? 'var(--green)' : 'var(--red)';

  const overlay = document.createElement('div');
  overlay.id = 'win-overlay';
  overlay.style.cssText = `
    position: fixed; inset: 0; background: rgba(0,0,0,0.85);
    display: flex; flex-direction: column; align-items: center;
    justify-content: center; z-index: 500; gap: 20px;
  `;
  overlay.innerHTML = `
    <div style="font-family:var(--font-display);font-size:72px;color:${color};letter-spacing:4px;">
      ${message}
    </div>
    <div style="font-size:16px;color:var(--text2);">Game over</div>
    <div style="display:flex;gap:12px;">
      <button class="btn btn-green" style="width:160px;" onclick="handleRematch()">
        Rematch
      </button>
      <button class="btn btn-primary" style="width:160px;" onclick="handleLeaveRoom()">
        Back to Lobby
      </button>
    </div>
  `;
  document.body.appendChild(overlay);
}

// ═══════════════════════════════════════════════════════════
// GAME END & REMATCH
// ═══════════════════════════════════════════════════════════

function onGameEnded(payload) {
    // Update grid one final time
    if (payload?.final_state?.grid) {
        updateGridFromState(payload.final_state.grid);
    }

    setTimeout(() => {
        if (payload?.is_draw) {
            showDrawScreen();
        } else {
            showWinScreen(payload?.winner_id);
        }
    }, 500);
}

function handleRematch() {
  wsSend('rematch', {});
  // Remove win overlay
  const overlay = document.getElementById('win-overlay');
  if (overlay) overlay.remove();
}
