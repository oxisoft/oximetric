function loadGeo(projectId) {
  if (!projectId) return;
  const qs = UI.queryString();
  API.get(`/analytics/${projectId}/geo?${qs}`).then(r => API.json(r)).then(data => {
    if (!data) return;
    document.getElementById('countriesTable').innerHTML = (data.countries || []).map(c =>
      `<tr><td>${UI.escapeHtml(c.country)}</td><td class="text-end">${UI.formatNumber(c.count)}</td></tr>`
    ).join('') || '<tr><td colspan="2" class="text-muted">No data</td></tr>';

    document.getElementById('citiesTable').innerHTML = (data.cities || []).map(c =>
      `<tr><td>${UI.escapeHtml(c.city)}</td><td>${UI.escapeHtml(c.country)}</td><td class="text-end">${UI.formatNumber(c.count)}</td></tr>`
    ).join('') || '<tr><td colspan="3" class="text-muted">No data</td></tr>';
  });
}

document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;
  UI.initProjectSelector(loadGeo);
  document.getElementById('applyDateFilter')?.addEventListener('click', () => loadGeo(UI.getSelectedProject()));
});
