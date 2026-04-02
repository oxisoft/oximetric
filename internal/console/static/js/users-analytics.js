let usersChart = null;

function loadUsersAnalytics(projectId) {
  if (!projectId) return;
  const qs = UI.queryString();

  API.get(`/analytics/${projectId}/users?${qs}&interval=day`).then(r => API.json(r)).then(data => {
    if (!data) return;
    document.getElementById('statTotalUsers').textContent = UI.formatNumber(data.total || 0);

    const ctx = document.getElementById('usersChart').getContext('2d');
    if (usersChart) usersChart.destroy();
    if (data.time_series && data.time_series.length > 0) {
      usersChart = new Chart(ctx, {
        type: 'line',
        data: {
          labels: data.time_series.map(p => p.timestamp),
          datasets: [{ label: 'New Users', data: data.time_series.map(p => p.count), borderColor: '#43e97b', backgroundColor: 'rgba(67,233,123,0.1)', fill: true, tension: 0.3 }]
        },
        options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true } } }
      });
    }
  });

  API.get(`/analytics/${projectId}/retention?${qs}`).then(r => API.json(r)).then(data => {
    const el = document.getElementById('retentionContent');
    if (!data || !data.cohorts || data.cohorts.length === 0) {
      el.innerHTML = '<p class="text-muted">No retention data yet</p>';
      return;
    }
    el.innerHTML = '<table class="table table-sm"><thead><tr><th>Cohort</th><th>Users</th><th>Retention</th></tr></thead><tbody>' +
      data.cohorts.map(c => `<tr><td>${c.period}</td><td>${c.users}</td><td>${(c.retention||[]).map(r=>(r*100).toFixed(0)+'%').join(', ')}</td></tr>`).join('') +
      '</tbody></table>';
  });
}

document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;
  UI.initProjectSelector(loadUsersAnalytics);
  document.getElementById('applyDateFilter')?.addEventListener('click', () => loadUsersAnalytics(UI.getSelectedProject()));
});
