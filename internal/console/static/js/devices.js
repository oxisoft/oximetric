let platformChart = null;

function loadDevices(projectId) {
  if (!projectId) return;
  const qs = UI.queryString();
  API.get(`/analytics/${projectId}/devices?${qs}`).then(r => API.json(r)).then(data => {
    if (!data) return;
    const platforms = data.platforms || [];
    const tbody = document.getElementById('platformsTable');
    tbody.innerHTML = platforms.map(p =>
      `<tr><td>${p.platform}</td><td class="text-end">${UI.formatNumber(p.count)}</td></tr>`
    ).join('') || '<tr><td colspan="2" class="text-muted">No data</td></tr>';

    const ctx = document.getElementById('platformChart').getContext('2d');
    if (platformChart) platformChart.destroy();
    if (platforms.length > 0) {
      const colors = ['#667eea','#764ba2','#f093fb','#4facfe','#43e97b','#fa709a'];
      platformChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
          labels: platforms.map(p => p.platform),
          datasets: [{ data: platforms.map(p => p.count), backgroundColor: colors.slice(0, platforms.length) }]
        },
        options: { responsive: true, maintainAspectRatio: false }
      });
    }
  });
}

document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;
  UI.initProjectSelector(loadDevices);
  document.getElementById('applyDateFilter')?.addEventListener('click', () => loadDevices(UI.getSelectedProject()));
});
