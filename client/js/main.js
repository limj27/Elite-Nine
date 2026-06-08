// ═══════════════════════════════════════════════════════════
// GLOBAL STATE
// Shared across all modules — do not duplicate these elsewhere
// ═══════════════════════════════════════════════════════════
const State = {
  token: null,
  myUsername: null,
  myClientId: null,
  currentRoom: null,
  isCreator: false,
  myReady: false,
  oppReady: false,
  gameStarted: false,
  myTurn: false,
  playerIndex: 0,
  players: [],
  gridTemplate: null, 
  cellHistrory: null,
};

// ═══════════════════════════════════════════════════════════
// SCREEN MANAGEMENT
// Uses History API so browser back/forward buttons work
// ═══════════════════════════════════════════════════════════
function showScreen(id) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.getElementById('screen-' + id).classList.add('active');
  const path = id === 'auth' ? '/game/' : `/game/${id}`;
  history.pushState({ screen: id }, '', path);
}

// Handle browser back/forward buttons
window.addEventListener('popstate', (e) => {
  const screen = e.state?.screen;
  if (!screen || screen === 'auth') {
    handleLogout();
    return;
  }
  if (screen === 'lobby') {
    if (State.gameStarted || State.currentRoom) {
      wsSend('leave_room', {});
      State.currentRoom = null;
      State.gameStarted = false;
      State.myReady     = false;
      State.oppReady    = false;
    }
    document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
    document.getElementById('screen-lobby').classList.add('active');
    requestRoomList();
    return;
  }
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.getElementById('screen-' + screen).classList.add('active');
});

// ═══════════════════════════════════════════════════════════
// TOAST
// ═══════════════════════════════════════════════════════════
let toastTimer;

function showToast(msg, type) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast show' + (type ? ' ' + type : '');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.classList.remove('show'), 3000);
}

// ═══════════════════════════════════════════════════════════
// AUTO LOGIN ON PAGE LOAD
// ═══════════════════════════════════════════════════════════
window.addEventListener('load', () => {
  const savedToken    = localStorage.getItem('elite9_token');
  const savedUsername = localStorage.getItem('elite9_username');

  if (savedToken && savedUsername) {
    State.token      = savedToken;
    State.myUsername = savedUsername;
    document.getElementById('lobby-username').textContent = savedUsername;
    showScreen('lobby');
    connectWebSocket();
  } else {
    history.replaceState({ screen: 'auth' }, '', '/game/');
  }
});