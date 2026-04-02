function loadAccountStatus() {
  API.get('/auth/me').then(r => API.json(r)).then(user => {
    if (!user) return;

    // Profile info
    document.getElementById('profileUsername').innerHTML = '<strong>' + UI.escapeHtml(user.username) + '</strong>';
    document.getElementById('profileEmail').textContent = user.email || '—';
    const colors = { admin: 'danger', manager: 'warning', viewer: 'info' };
    document.getElementById('profileRole').innerHTML = '<span class="badge bg-' + (colors[user.role] || 'secondary') + '">' + UI.escapeHtml(user.role) + '</span>';
    document.getElementById('profileTOTP').innerHTML = user.totp_enabled
      ? '<span class="badge bg-success">Enabled</span>'
      : '<span class="badge bg-secondary">Disabled</span>';
    document.getElementById('profileCreated').textContent = user.created_at ? UI.formatDate(user.created_at) : '—';

    // TOTP status
    const status = document.getElementById('totpStatus');
    if (user.totp_enabled) {
      status.innerHTML = `
        <p><span class="badge bg-success">2FA Enabled</span></p>
        <p class="text-muted">Enter your password to disable 2FA.</p>
        <div class="mb-3"><input type="password" id="disablePassword" class="form-control" placeholder="Password"></div>
        <button class="btn btn-outline-danger" id="btnDisableTOTP">Disable 2FA</button>`;
      status.querySelector('#btnDisableTOTP').addEventListener('click', disableTOTP);
    } else {
      status.innerHTML = `
        <p><span class="badge bg-secondary">2FA Disabled</span></p>
        <button class="btn btn-outline-primary" id="btnSetupTOTP">Setup 2FA</button>`;
      status.querySelector('#btnSetupTOTP').addEventListener('click', setupTOTP);
    }
  });
}

function setupTOTP() {
  API.post('/auth/totp/setup').then(r => API.json(r)).then(data => {
    if (!data) return;
    document.getElementById('totpURI').textContent = data.uri;
    document.getElementById('totpSetup').style.display = 'block';
  });
}

function enableTOTP() {
  const code = document.getElementById('totpCode').value.trim();
  const password = document.getElementById('totpPassword').value;
  if (!code || !password) return;
  API.post('/auth/totp/enable', { code, password }).then(r => {
    if (r.ok) {
      UI.showToast('2FA enabled');
      document.getElementById('totpSetup').style.display = 'none';
      loadAccountStatus();
    } else {
      r.json().then(d => UI.showToast(d.error?.message || 'Invalid code', 'danger'));
    }
  });
}

function disableTOTP() {
  const password = document.getElementById('disablePassword').value;
  if (!password) return;
  API.post('/auth/totp/disable', { password }).then(r => {
    if (r.ok) {
      UI.showToast('2FA disabled');
      loadAccountStatus();
    } else {
      r.json().then(d => UI.showToast(d.error?.message || 'Error', 'danger'));
    }
  });
}

document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;
  loadAccountStatus();

  document.getElementById('passwordForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const newPw = document.getElementById('newPassword').value;
    const confirmPw = document.getElementById('confirmPassword').value;
    if (newPw !== confirmPw) {
      UI.showToast('Passwords do not match', 'danger');
      return;
    }
    if (newPw.length < 8) {
      UI.showToast('Password must be at least 8 characters', 'danger');
      return;
    }
    const resp = await API.put('/auth/password', {
      current_password: document.getElementById('currentPassword').value,
      new_password: newPw,
    });
    if (resp && resp.ok) {
      UI.showToast('Password updated');
      document.getElementById('passwordForm').reset();
    } else if (resp) {
      const d = await resp.json();
      UI.showToast(d.error?.message || 'Error', 'danger');
    }
  });

  document.getElementById('btnEnableTOTP')?.addEventListener('click', enableTOTP);
});
