document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;

  document.getElementById('aboutYear').textContent = new Date().getFullYear();

  fetch('/api/v1/health').then(r => r.json()).then(data => {
    document.getElementById('aboutVersion').textContent = data.version || '—';
    document.getElementById('aboutDatabase').textContent = data.database || '—';
    document.getElementById('aboutGeoIP').textContent = data.geoip || '—';
  });
});
