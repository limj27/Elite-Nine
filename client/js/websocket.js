// ═══════════════════════════════════════════════════════════
// WEBSOCKET
// Manages the connection and routes incoming messages
// to the appropriate handler in lobby.js or game.js
// ═══════════════════════════════════════════════════════════

function connectWebSocket() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const port  = location.protocol === 'https:' ? '' : ':8080';
  const addr  = `${proto}//${location.hostname}${port}/ws?token=${encodeURIComponent(State.token)}`;

  window.ws = new WebSocket(addr);

  window.ws.onopen  = () => { requestRoomList(); };
  window.ws.onclose = () => { showToast('Disconnected from server', 'error'); };
  window.ws.onerror = () => { showToast('Connection error', 'error'); };

  window.ws.onmessage = (e) => {
    // Backend may batch multiple messages in one frame separated by \n
    const frames = e.data.split('\n').filter(s => s.trim());
    frames.forEach(raw => {
      try {
        handleServerMessage(JSON.parse(raw));
      } catch (err) {
        console.warn('WebSocket parse error:', err, raw);
      }
    });
  };
}

function wsSend(type, payload) {
  if (!window.ws || window.ws.readyState !== WebSocket.OPEN) return;
  window.ws.send(JSON.stringify({ type, payload: payload || {} }));
}

// ═══════════════════════════════════════════════════════════
// MESSAGE ROUTER
// Each case delegates to the relevant module
// ═══════════════════════════════════════════════════════════
function handleServerMessage(msg) {
  switch (msg.type) {

    // ── Connection ──────────────────────────────────────────
    case 'connected':
      State.myClientId = msg.payload?.clientId;
      break;

    // ── Lobby ───────────────────────────────────────────────
    case 'rooms_list':
      renderRoomList(msg.payload?.rooms || []);
      break;

    // ── Room join / create ───────────────────────────────────
    case 'room_created':
      State.currentRoom = msg.payload;
      State.isCreator   = true;
      enterGameScreen(msg.payload.room_name);
      break;

    case 'joined_room':
      // Only enter game screen if not already there
      if (!State.gameStarted && !State.currentRoom) {
        State.currentRoom = msg.payload;
        State.isCreator   = false;
        enterGameScreen(msg.payload.room_name);
      }
      break;

    // ── In-room events ───────────────────────────────────────
    case 'player_joined': onPlayerJoined(msg.payload);  break;
    case 'player_left':   onPlayerLeft(msg.payload);    break;
    case 'player_ready':  onPlayerReady(msg.payload);   break;

    case 'room_ready':
      console.log('room_ready received, isCreator:', State.isCreator);
      if (State.isCreator) {
        document.getElementById('start-btn').disabled = false;
        showToast('Both players ready! You can start the game.', 'success');
      }
      break;
    case 'game_ended':
        onGameEnded(msg.payload);
        break;

    case 'rematch': {
        // Reset game state and go back to ready screen
        State.gameStarted  = false;
        State.myReady      = false;
        State.oppReady     = false;
        State.playerIndex  = 0;
        State.gridTemplate = null;

        const overlay = document.getElementById('win-overlay');
        if (overlay) overlay.remove();
        const historySection = document.getElementById('cell-history-section');
        if (historySection) historySection.remove();

        // Show waiting state again
        document.getElementById('waiting-state').style.display = 'block';
        document.getElementById('grid-wrap').style.display     = 'none';
        document.getElementById('ready-section').style.display = 'flex';
        document.getElementById('start-btn').disabled          = true;
        document.getElementById('ready-btn').textContent       = 'Mark Ready';

        updateReadyUI();
        showToast('Rematch started — mark ready to play again!', 'success');
        break;
    }
    // ── Game ─────────────────────────────────────────────────
    case 'game_started':
      if (msg.payload?.playerIndex !== undefined) {
          State.playerIndex    = msg.payload.playerIndex;
          State.gridTemplate   = {
              rowCriteria: msg.payload.rowCriteria,
              colCriteria: msg.payload.colCriteria,
          };
          updatePlayerColors();
          renderGridHeaders();  // new function
      }
      if (!State.gameStarted) {
          State.gameStarted = true;
          document.getElementById('waiting-state').style.display = 'none';
          document.getElementById('grid-wrap').style.display     = 'flex';
          document.getElementById('ready-section').style.display = 'none';
          buildGrid();
      }
      break;

    case 'game_state':
      if (msg.payload?.players) {
        State.players = msg.payload.players;
      }
      if (msg.payload?.cell_history) {
        State.cellHistory = msg.payload.cell_history;
      }
      onGameState(msg.payload);
      break;

    case 'move_made':
      onMoveMade(msg.payload);
      break;

    case 'invalid_move':
      showToast(msg.payload?.message || 'Wrong answer — turn lost!', 'error');
      break;
    
    case 'cell_overtaken':
      showToast(
          msg.payload?.newPlayer + ' overtook ' + msg.payload?.oldPlayer + '!',
          'success'
      );
      break;

    case 'overtake_failed':
      showToast(
          'Not rare enough to overtake! (yours: ' +
          (msg.payload?.yourRarity * 100).toFixed(1) + '% vs existing: ' +
          (msg.payload?.existingRarity * 100).toFixed(1) + '%)',
          'error'
      );
      break;
    // ── Errors ───────────────────────────────────────────────
    case 'error':
      showToast(msg.message || 'An error occurred', 'error');
      break;

    default:
      console.log('Unhandled message type:', msg.type, msg);
  }
}
