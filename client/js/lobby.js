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
  const container = document.getElementById('rooms-list');  // ← add the 's'
  if (!container) return;  // ← also add this safety check

  if (!rooms || !rooms.length) {
    container.innerHTML = `
      <div class="empty-rooms">
        <div class="big">⚾</div>
        <p>No open rooms yet.<br>Be the first to create one.</p>
      </div>`;
    return;
  }

  container.innerHTML = rooms.map(room => {
    const difficultyClass = 'difficulty-' + (room.difficulty || 'regular');
    const difficultyLabel = (room.difficulty || 'regular').charAt(0).toUpperCase()
                           + (room.difficulty || 'regular').slice(1);
    const lockIcon = room.has_password ? '🔒 ' : '';
    const pip = (filled) => `<div class="pip${filled ? ' filled' : ''}"></div>`;
    const pips = Array.from({length: room.max_players}, (_, i) =>
      pip(i < room.player_count)).join('');

    return `
      <div class="room-card" onclick="handleJoinRoomClick('${room.id}', ${room.has_password})">
        <div class="room-card-left">
          <div class="room-card-name">
            ${lockIcon}${room.name}
          </div>
          <div class="room-card-meta">
            <span class="room-status status-${room.status}">${room.status}</span>
            <span class="badge ${difficultyClass}">${difficultyLabel}</span>
            <div class="players-pip">${pips}</div>
            <span>${room.player_count}/${room.max_players}</span>
          </div>
        </div>
      </div>`;
  }).join('');
}

// ── Create room ─────────────────────────────────────────────

function toggleCreateForm() {
  document.getElementById('create-form').classList.toggle('show');
}

function handleCreateRoom() {
  const roomName = document.getElementById('new-room-name').value.trim();
  const password = document.getElementById('new-room-pass').value.trim();
  const difficulty = document.getElementById('create-room-difficulty').value; // ADD THIS
 
  if (!roomName) {
    showToast('Room name is required', 'error');
    return;
  }
 
  wsSend('create_room', {
    room_name: roomName,
    password:  password,
    max_players: 2,
    difficulty: difficulty, // ADD THIS
  });
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
