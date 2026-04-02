let eventsChart = null;

function loadDashboard(projectId) {
  if (!projectId) return;
  const qs = UI.queryString();

  API.get(`/analytics/${projectId}/overview?${qs}`).then(r => API.json(r)).then(data => {
    if (!data) return;
    document.getElementById('statDevices').textContent = UI.formatNumber(data.active_devices || 0);
    document.getElementById('statEvents').textContent = UI.formatNumber(data.total_events || 0);

    const topEvt = data.top_events && data.top_events.length > 0 ? data.top_events[0].name : '—';
    document.getElementById('statTopEvent').textContent = topEvt;

    const topCountry = data.top_countries && data.top_countries.length > 0 ? data.top_countries[0].country : '—';
    document.getElementById('statTopCountry').textContent = topCountry;

    const tbody = document.getElementById('topEventsTable');
    tbody.innerHTML = (data.top_events || []).map(e =>
      `<tr><td>${UI.escapeHtml(e.name)}</td><td class="text-end">${UI.formatNumber(e.count)}</td></tr>`
    ).join('') || '<tr><td colspan="2" class="text-muted">No data</td></tr>';

    const ctbody = document.getElementById('topCountriesTable');
    ctbody.innerHTML = (data.top_countries || []).map(c =>
      `<tr><td>${UI.escapeHtml(c.country)}</td><td class="text-end">${UI.formatNumber(c.count)}</td></tr>`
    ).join('') || '<tr><td colspan="2" class="text-muted">No data</td></tr>';
  });

  API.get(`/analytics/${projectId}/events?${qs}&interval=day`).then(r => API.json(r)).then(data => {
    if (!data || !data.time_series) return;
    const ctx = document.getElementById('eventsChart').getContext('2d');
    if (eventsChart) eventsChart.destroy();
    eventsChart = new Chart(ctx, {
      type: 'line',
      data: {
        labels: data.time_series.map(p => p.timestamp),
        datasets: [{
          label: 'Events',
          data: data.time_series.map(p => p.count),
          borderColor: '#667eea',
          backgroundColor: 'rgba(102,126,234,0.1)',
          fill: true, tension: 0.3,
        }]
      },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { legend: { display: false } },
        scales: { y: { beginAtZero: true } },
      }
    });
  });
}

document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;
  UI.initProjectSelector(loadDashboard);
  document.getElementById('applyDateFilter')?.addEventListener('click', () => {
    loadDashboard(UI.getSelectedProject());
  });
});
