// ═══════════════════════════════════════════════════════════
// WEBSOCKET
// Manages the connection and routes incoming messages
// to the appropriate handler in lobby.js or game.js
// ═══════════════════════════════════════════════════════════

function connectWebSocket() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const addr  = `${proto}//${location.hostname}:8080/ws?token=${encodeURIComponent(State.token)}`;

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
    case 'joined_room':
      State.currentRoom = msg.payload;
      State.isCreator   = msg.type === 'room_created';
      enterGameScreen(msg.payload.room_name);
      break;

    // ── In-room events ───────────────────────────────────────
    case 'player_joined': onPlayerJoined(msg.payload);  break;
    case 'player_left':   onPlayerLeft(msg.payload);    break;
    case 'player_ready':  onPlayerReady(msg.payload);   break;

    case 'room_ready':
      if (State.isCreator) {
        document.getElementById('start-btn').disabled = false;
        showToast('Both players ready! You can start the game.', 'success');
      }
      break;

    // ── Game ─────────────────────────────────────────────────
    case 'game_started':
    case 'game_state':
      onGameState(msg.payload);
      break;

    case 'move_made':
      onMoveMade(msg.payload);
      break;

    // ── Errors ───────────────────────────────────────────────
    case 'error':
      showToast(msg.message || 'An error occurred', 'error');
      break;

    default:
      console.log('Unhandled message type:', msg.type, msg);
  }
}
