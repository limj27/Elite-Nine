// ═══════════════════════════════════════════════════════════
// AUTH
// Handles login, register, logout, and tab switching
// ═══════════════════════════════════════════════════════════

function switchTab(tab) {
  document.querySelectorAll('.auth-tab').forEach((t, i) => {
    t.classList.toggle('active', (tab === 'login' && i === 0) || (tab === 'register' && i === 1));
  });
  document.getElementById('form-login').classList.toggle('active', tab === 'login');
  document.getElementById('form-register').classList.toggle('active', tab === 'register');
}

async function handleLogin(e) {
  e.preventDefault();
  const btn   = document.getElementById('login-btn');
  const errEl = document.getElementById('login-error');
  errEl.classList.remove('show');
  btn.disabled    = true;
  btn.textContent = 'Signing in...';

  try {
    const resp = await fetch('/login', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: document.getElementById('login-username').value.trim(),
        password: document.getElementById('login-password').value,
      }),
    });

    const body = await resp.json();
    if (!resp.ok) {
      errEl.textContent = body.message || 'Login failed';
      errEl.classList.add('show');
      return;
    }

    onAuthSuccess(body.token, document.getElementById('login-username').value.trim());
  } catch {
    errEl.textContent = 'Network error';
    errEl.classList.add('show');
  } finally {
    btn.disabled    = false;
    btn.textContent = 'Sign In';
  }
}

async function handleRegister(e) {
  e.preventDefault();
  const btn   = document.getElementById('register-btn');
  const errEl = document.getElementById('register-error');
  errEl.classList.remove('show');
  btn.disabled    = true;
  btn.textContent = 'Creating account...';

  try {
    const resp = await fetch('/register', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: document.getElementById('reg-username').value.trim(),
        email:    document.getElementById('reg-email').value.trim(),
        password: document.getElementById('reg-password').value,
      }),
    });

    const body = await resp.json();
    if (!resp.ok) {
      errEl.textContent = body.message || 'Registration failed';
      errEl.classList.add('show');
      return;
    }

    onAuthSuccess(body.token, document.getElementById('reg-username').value.trim());
  } catch {
    errEl.textContent = 'Network error';
    errEl.classList.add('show');
  } finally {
    btn.disabled    = false;
    btn.textContent = 'Create Account';
  }
}

function onAuthSuccess(token, username) {
  State.token      = token;
  State.myUsername = username;
  localStorage.setItem('elite9_token',    token);
  localStorage.setItem('elite9_username', username);
  document.getElementById('lobby-username').textContent = username;
  showScreen('lobby');
  connectWebSocket();
}

function handleLogout() {
  if (window.ws) window.ws.close();
  State.token      = null;
  State.myUsername = null;
  localStorage.removeItem('elite9_token');
  localStorage.removeItem('elite9_username');
  showScreen('auth');
}
