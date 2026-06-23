// ═══════════════════════════════════════════════════════════
// SETTINGS
// Handles profile updates, favorite team, game history,
// and account deletion
// ═══════════════════════════════════════════════════════════

// ── Authenticated fetch helper ───────────────────────────────
async function authFetch(url, options = {}) {
  return fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${State.token}`,
      ...(options.headers || {}),
    },
  });
}

// ── Open settings screen ─────────────────────────────────────
async function openSettings() {
  showScreen('settings');
  loadGameHistory();
  loadCurrentProfile();
}

// ── Load current profile ─────────────────────────────────────
async function loadCurrentProfile() {
  try {
    const resp = await authFetch('/api/profile/full');
    if (!resp.ok) return;
    const user = await resp.json();

    // Pre-fill username field
    document.getElementById('settings-username').value = user.username || '';

    // Show current favorite team
    updateFavTeamDisplay(user.favorite_team_id, user.favorite_team_name);
  } catch {
    // non-fatal
  }
}

function updateFavTeamDisplay(teamID, teamName) {
  const nameEl = document.getElementById('fav-team-name');
  const logoEl = document.getElementById('fav-team-logo');

  if (teamID && teamName) {
    nameEl.textContent = teamName;
    logoEl.innerHTML = `
      <img src="https://www.mlbstatic.com/team-logos/${teamID}.svg"
           style="width:32px;height:32px;object-fit:contain;margin-right:10px;"
           onerror="this.style.display='none'">`;
  } else {
    nameEl.textContent = 'No favorite team set';
    logoEl.innerHTML = '';
  }
}

// ── Username availability check ──────────────────────────────
let usernameCheckTimer = null;

async function checkUsernameAvailability() {
  clearTimeout(usernameCheckTimer);
  const input  = document.getElementById('settings-username').value.trim();
  const msgEl  = document.getElementById('username-availability');

  if (input.length < 3) {
    msgEl.textContent = '';
    msgEl.className   = 'availability-msg';
    return;
  }

  msgEl.textContent = 'Checking...';
  msgEl.className   = 'availability-msg';

  usernameCheckTimer = setTimeout(async () => {
    try {
      const resp = await authFetch('/api/profile/check-username', {
        method: 'POST',
        body:   JSON.stringify({ username: input }),
      });
      const data = await resp.json();

      if (data.available) {
        msgEl.textContent = '✓ Available';
        msgEl.className   = 'availability-msg available';
      } else {
        msgEl.textContent = '✗ Already taken';
        msgEl.className   = 'availability-msg taken';
      }
    } catch {
      msgEl.textContent = '';
    }
  }, 400);
}

// ── Update username ───────────────────────────────────────────
async function handleUpdateUsername() {
  const username = document.getElementById('settings-username').value.trim();
  const btn      = document.getElementById('update-username-btn');

  if (username.length < 3) {
    showToast('Username must be at least 3 characters', 'error');
    return;
  }

  btn.disabled    = true;
  btn.textContent = 'Updating...';

  try {
    const resp = await authFetch('/api/profile/username', {
      method: 'PUT',
      body:   JSON.stringify({ username }),
    });

    if (resp.status === 409) {
      showToast('Username already taken', 'error');
      return;
    }
    if (!resp.ok) {
      showToast('Failed to update username', 'error');
      return;
    }

    const data = await resp.json();

    // Update token and username in state + localStorage
    State.token      = data.token;
    State.myUsername = username;
    localStorage.setItem('elite9_token',    data.token);
    localStorage.setItem('elite9_username', username);
    document.getElementById('lobby-username').textContent = username;

    showToast('Username updated!', 'success');
  } catch {
    showToast('Network error', 'error');
  } finally {
    btn.disabled    = false;
    btn.textContent = 'Update Username';
  }
}

// ── Update password ───────────────────────────────────────────
async function handleUpdatePassword() {
  const currentPw = document.getElementById('settings-current-password').value;
  const newPw     = document.getElementById('settings-new-password').value;

  if (!currentPw || !newPw) {
    showToast('Both password fields are required', 'error');
    return;
  }
  if (newPw.length < 8) {
    showToast('New password must be at least 8 characters', 'error');
    return;
  }

  try {
    const resp = await authFetch('/api/profile/password', {
      method: 'PUT',
      body:   JSON.stringify({
        current_password: currentPw,
        new_password:     newPw,
      }),
    });

    if (resp.status === 401) {
      showToast('Current password is incorrect', 'error');
      return;
    }
    if (!resp.ok) {
      showToast('Failed to update password', 'error');
      return;
    }

    document.getElementById('settings-current-password').value = '';
    document.getElementById('settings-new-password').value     = '';
    showToast('Password updated!', 'success');
  } catch {
    showToast('Network error', 'error');
  }
}

// ── Team search ───────────────────────────────────────────────
let allTeams = null;
let teamSearchTimer = null;

async function loadTeams() {
  if (allTeams) return allTeams;
  try {
    const resp = await fetch('https://statsapi.mlb.com/api/v1/teams?sportId=1&activeStatus=Yes');
    const data = await resp.json();
    allTeams = data.teams || [];
    return allTeams;
  } catch {
    return [];
  }
}

async function handleTeamSearch(query) {
  clearTimeout(teamSearchTimer);
  const resultsEl = document.getElementById('team-search-results');

  if (query.length < 1) {
    resultsEl.innerHTML = '';
    return;
  }

  teamSearchTimer = setTimeout(async () => {
    const teams = await loadTeams();
    const q     = query.toLowerCase();
    const matches = teams.filter(t =>
      t.name.toLowerCase().includes(q) ||
      t.abbreviation?.toLowerCase().includes(q) ||
      t.teamName?.toLowerCase().includes(q)
    ).slice(0, 8);

    if (!matches.length) {
      resultsEl.innerHTML = '<div class="team-result-empty">No teams found</div>';
      return;
    }

    resultsEl.innerHTML = matches.map(t => `
      <div class="team-result-item" onclick="selectFavoriteTeam(${t.id}, '${t.name.replace(/'/g, "\\'")}')">
        <img src="https://www.mlbstatic.com/team-logos/${t.id}.svg"
             style="width:28px;height:28px;object-fit:contain;"
             onerror="this.style.display='none'">
        <span>${t.name}</span>
      </div>
    `).join('');
  }, 300);
}

async function selectFavoriteTeam(teamID, teamName) {
  try {
    const resp = await authFetch('/api/profile/team', {
      method: 'PUT',
      body:   JSON.stringify({ team_id: teamID, team_name: teamName }),
    });

    if (!resp.ok) {
      showToast('Failed to update favorite team', 'error');
      return;
    }

    updateFavTeamDisplay(teamID, teamName);
    document.getElementById('team-search-input').value = '';
    document.getElementById('team-search-results').innerHTML = '';
    showToast(`${teamName} set as your favorite team!`, 'success');
  } catch {
    showToast('Network error', 'error');
  }
}

// ── Game history ──────────────────────────────────────────────
async function loadGameHistory() {
  const listEl = document.getElementById('game-history-list');
  listEl.innerHTML = '<div class="empty-state">Loading...</div>';

  try {
    const resp = await authFetch('/api/profile/history');
    if (!resp.ok) {
      listEl.innerHTML = '<div class="empty-state">Failed to load history</div>';
      return;
    }

    const data    = await resp.json();
    const history = data.history || [];

    if (!history.length) {
      listEl.innerHTML = '<div class="empty-state">No games played yet</div>';
      return;
    }

    listEl.innerHTML = history.map(entry => {
      const resultClass = entry.result === 'win'  ? 'result-win'
                        : entry.result === 'loss' ? 'result-loss'
                        : 'result-draw';
      const resultLabel = entry.result === 'win'  ? 'W'
                        : entry.result === 'loss' ? 'L'
                        : 'D';
      const diffClass = 'difficulty-' + (entry.difficulty || 'regular');
      const diffLabel = (entry.difficulty || 'regular').charAt(0).toUpperCase()
                       + (entry.difficulty || 'regular').slice(1);
      const date = entry.played_at
        ? new Date(entry.played_at).toLocaleDateString()
        : '—';

      return `
        <div class="history-item">
          <div class="history-result ${resultClass}">${resultLabel}</div>
          <div class="history-info">
            <div class="history-opponent">vs ${entry.opponent_name}</div>
            <div class="history-meta">
              <span class="badge ${diffClass}">${diffLabel}</span>
              <span class="history-date">${date}</span>
            </div>
          </div>
        </div>`;
    }).join('');
  } catch {
    listEl.innerHTML = '<div class="empty-state">Failed to load history</div>';
  }
}

// ── Delete account ────────────────────────────────────────────
function showDeleteConfirm() {
  document.getElementById('delete-confirm-input').value = '';
  document.getElementById('delete-confirm-modal').classList.add('show');
}

function closeDeleteConfirm() {
  document.getElementById('delete-confirm-modal').classList.remove('show');
}

async function handleDeleteAccount() {
  const input = document.getElementById('delete-confirm-input').value.trim();
  if (input !== 'DELETE') {
    showToast('Type DELETE to confirm', 'error');
    return;
  }

  try {
    const resp = await authFetch('/api/profile', { method: 'DELETE' });
    if (!resp.ok) {
      showToast('Failed to delete account', 'error');
      return;
    }

    closeDeleteConfirm();
    handleLogout();
    showToast('Account deleted', '');
  } catch {
    showToast('Network error', 'error');
  }
}