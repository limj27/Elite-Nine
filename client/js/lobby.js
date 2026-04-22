// ═══════════════════════════════════════════════════════════
// LOBBY
// Room list rendering, create room, join room (with password)
// ═══════════════════════════════════════════════════════════

let pendingJoinRoom = null;

// ── Room list ───────────────────────────────────────────────

function requestRoomList() {
  wsSend('list_rooms');
}

function renderRoomList(rooms) {
  const el = document.getElementById('rooms-list');

  if (!rooms.length) {
    el.innerHTML = `
      <div class="empty-rooms">
        <div class="big">⚾</div>
        <p>No open rooms yet.<br>Be the first to create one.</p>
      </div>`;
    return;
  }

  el.innerHTML = rooms.map(r => {
    const locked     = r.has_password ? `<span class="lock-icon">🔒</span>` : '';
    const pips       = [0, 1].map(i => `<div class="pip ${i < (r.player_count || 0) ? 'filled' : ''}"></div>`).join('');
    const statusClass = r.status === 'ready'  ? 'status-ready'
                      : r.status === 'active' ? 'status-active'
                      : 'status-waiting';
    return `
      <div class="room-card" onclick="handleJoinClick('${r.id}', '${r.name}', ${!!r.has_password})">
        <div class="room-card-left">
          <div class="room-card-name">${r.name} ${locked}</div>
          <div class="room-card-meta">
            <div class="players-pip">${pips}</div>
            <span>${r.player_count || 0}/${r.max_players || 2} players</span>
          </div>
        </div>
        <span class="room-status ${statusClass}">${r.status || 'waiting'}</span>
      </div>`;
  }).join('');
}

// ── Create room ─────────────────────────────────────────────

function toggleCreateForm() {
  document.getElementById('create-form').classList.toggle('show');
}

function handleCreateRoom() {
  const name = document.getElementById('new-room-name').value.trim();
  const pass = document.getElementById('new-room-pass').value;

  if (!name) {
    showToast('Room name is required', 'error');
    return;
  }

  wsSend('create_room', { room_name: name, password: pass, max_players: 2 });

  // Reset and hide the form
  document.getElementById('create-form').classList.remove('show');
  document.getElementById('new-room-name').value = '';
  document.getElementById('new-room-pass').value = '';
}

// ── Join room ───────────────────────────────────────────────

function handleJoinClick(id, name, hasPassword) {
  if (hasPassword) {
    pendingJoinRoom = { id, name };
    document.getElementById('modal-password').value = '';
    document.getElementById('password-modal').classList.add('show');
  } else {
    wsSend('join_room', { room_id: id, room_name: name });
  }
}

function closePasswordModal() {
  document.getElementById('password-modal').classList.remove('show');
  pendingJoinRoom = null;
}

function submitPasswordJoin() {
  if (!pendingJoinRoom) return;
  const pass = document.getElementById('modal-password').value;
  wsSend('join_room', {
    room_id:   pendingJoinRoom.id,
    room_name: pendingJoinRoom.name,
    password:  pass,
  });
  closePasswordModal();
}
