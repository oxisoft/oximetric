document.addEventListener('DOMContentLoaded', () => {
  // Set copyright year
  const yearEl = document.getElementById('copyrightYear');
  if (yearEl) yearEl.textContent = new Date().getFullYear();

  if (sessionStorage.getItem('jwt')) {
    window.location.href = '/dashboard';
    return;
  }

  const form = document.getElementById('loginForm');
  const alertEl = document.getElementById('loginAlert');
  const totpGroup = document.getElementById('totpGroup');
  let totpRequired = false;

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    alertEl.classList.add('d-hidden');

    const body = {
      login: document.getElementById('loginInput').value,
      password: document.getElementById('passwordInput').value,
    };
    if (totpRequired) {
      body.totp_code = document.getElementById('totpInput').value;
    }

    try {
      const resp = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await resp.json();

      if (!resp.ok) {
        alertEl.textContent = data.error?.message || 'Login failed';
        alertEl.classList.remove('d-hidden');
        return;
      }

      if (data.totp_required) {
        totpRequired = true;
        totpGroup.classList.remove('d-hidden');
        document.getElementById('totpInput').focus();
        return;
      }

      sessionStorage.setItem('jwt', data.token);
      sessionStorage.setItem('user', JSON.stringify(data.user));
      window.location.href = '/dashboard';
    } catch (err) {
      alertEl.textContent = 'Connection error: ' + err.message;
      alertEl.classList.remove('d-hidden');
    }
  });
});
