let timeChart = null;

function loadEvents(projectId) {
  if (!projectId) return;
  const qs = UI.queryString();

  API.get(`/analytics/${projectId}/events?${qs}&interval=day`).then(r => API.json(r)).then(data => {
    if (!data) return;
    const tbody = document.getElementById('eventsTable');
    tbody.innerHTML = (data.events || []).map(e =>
      `<tr class="cursor-pointer" data-name="${UI.escapeHtml(e.name)}" data-pid="${projectId}"><td>${UI.escapeHtml(e.name)}</td><td class="text-end">${UI.formatNumber(e.count)}</td></tr>`
    ).join('') || '<tr><td colspan="2" class="text-muted">No events</td></tr>';

    tbody.querySelectorAll('tr[data-name]').forEach(row => {
      row.addEventListener('click', () => loadProps(parseInt(row.dataset.pid), row.dataset.name));
    });

    const ctx = document.getElementById('eventsTimeChart').getContext('2d');
    if (timeChart) timeChart.destroy();
    if (data.time_series && data.time_series.length > 0) {
      timeChart = new Chart(ctx, {
        type: 'bar',
        data: {
          labels: data.time_series.map(p => p.timestamp),
          datasets: [{ label: 'Events', data: data.time_series.map(p => p.count), backgroundColor: '#667eea' }]
        },
        options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true } } }
      });
    }
  });
}

function loadProps(projectId, eventName) {
  const qs = UI.queryString();
  document.getElementById('propsCard').style.display = 'block';
  document.getElementById('propsEventName').textContent = eventName;

  API.get(`/analytics/${projectId}/events/${encodeURIComponent(eventName)}/properties?${qs}`).then(r => API.json(r)).then(data => {
    const container = document.getElementById('propsContent');
    if (!data || !data.properties || data.properties.length === 0) {
      container.innerHTML = '<p class="text-muted">No properties</p>';
      return;
    }
    container.innerHTML = data.properties.map(p => `
      <div class="mb-3">
        <strong>${p.key}</strong> <span class="badge bg-secondary">${p.type}</span>
        <table class="table table-sm mt-1">
          <thead><tr><th>Value</th><th class="text-end">Count</th></tr></thead>
          <tbody>${(p.values || []).map(v => `<tr><td>${v.value}</td><td class="text-end">${v.count}</td></tr>`).join('')}</tbody>
        </table>
      </div>
    `).join('');
  });
}

document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;
  UI.initProjectSelector(loadEvents);
  document.getElementById('applyDateFilter')?.addEventListener('click', () => loadEvents(UI.getSelectedProject()));
});
