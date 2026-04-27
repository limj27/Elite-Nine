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
  // Build grid if not already built (fallback)
  if (!State.gameStarted) {
    State.gameStarted = true;
    document.getElementById('waiting-state').style.display = 'none';
    document.getElementById('grid-wrap').style.display     = 'flex';
    document.getElementById('ready-section').style.display = 'none';
    buildGrid();
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

function updateTurnBar(currentTurn) {
  console.log('updateTurnBar — currentTurn:', currentTurn, 'playerIndex:', State.playerIndex, 'myTurn:', currentTurn === State.playerIndex);
  State.myTurn = currentTurn === State.playerIndex;
  const text = State.myTurn ? 'Your turn' : "Opponent's turn";
  console.log('setting turn text to:', text);
  document.getElementById('turn-text').textContent = text;
  document.getElementById('turn-bar').style.borderColor = State.myTurn
    ? 'rgba(59,130,246,0.5)'
    : 'rgba(239,68,68,0.3)';
}

function updateGridFromState(grid) {
  // grid is [3][3] — iterate rows and cols
  for (let row = 0; row < 3; row++) {
    for (let col = 0; col < 3; col++) {
      const idx  = row * 3 + col;
      const move = grid[row][col]; // null if no move yet

      if (!move) {
        gridState[idx] = { owner: null, player: null, rarity: 0 };
      } else {
        // Determine owner based on which player made the move
        const playerIndex = State.players?.findIndex(p => p.user_id === move.user_id);
        gridState[idx] = {
          owner:  playerIndex === 0 ? 'p1' : 'p2',
          player: { fullName: move.player_name || move.player_answer },
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
  if (!State.myTurn) {
    showToast("It's not your turn", 'error');
    return;
  }
  selectedCell = idx;
  openSearchModal();
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
