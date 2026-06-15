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
  const container = document.getElementById('room-list');
 
  if (!rooms.length) {
    container.innerHTML = '<div class="empty-state">No rooms available. Create one!</div>';
    return;
  }
 
  container.innerHTML = rooms.map(room => {
    const statusClass = room.status === 'waiting' ? 'status-waiting'
                       : room.status === 'active'  ? 'status-active'
                       : 'status-closed';
 
    const difficultyClass = 'difficulty-' + (room.difficulty || 'regular');
    const difficultyLabel = (room.difficulty || 'regular').charAt(0).toUpperCase()
                           + (room.difficulty || 'regular').slice(1);
 
    const lockIcon = room.has_password ? '🔒 ' : '';
 
    return `
      <div class="room-item" onclick="handleJoinRoomClick('${room.id}', ${room.has_password})">
        <div class="room-info">
          <div class="room-name">${lockIcon}${room.name}</div>
          <div class="room-meta">
            <span class="badge ${statusClass}">${room.status}</span>
            <span class="badge ${difficultyClass}">${difficultyLabel}</span>
            <span class="room-players">${room.player_count}/${room.max_players} players</span>
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
  const roomName = document.getElementById('create-room-name').value.trim();
  const password = document.getElementById('create-room-password').value.trim();
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
