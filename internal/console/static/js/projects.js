let selectedProjectId = null;

function loadProjects() {
  const isAdmin = Auth.getUser()?.role === 'admin';
  API.get('/projects').then(r => API.json(r)).then(projects => {
    const tbody = document.getElementById('projectsTable');
    if (!projects || projects.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="text-muted">No projects yet</td></tr>';
      return;
    }
    tbody.innerHTML = projects.map(p => `
      <tr>
        <td><a href="#" class="project-link" data-id="${p.id}" data-name="${UI.escapeHtml(p.name)}">${UI.escapeHtml(p.name)}</a></td>
        <td>${p.retention_days === 0 ? 'Forever' : p.retention_days + ' days'}</td>
        <td>${UI.formatDate(p.created_at)}</td>
        <td class="text-end">
          ${isAdmin ? `<button class="btn btn-sm btn-outline-danger btn-delete-project" data-id="${p.id}">Delete</button>` : ''}
        </td>
      </tr>
    `).join('');

    tbody.querySelectorAll('.project-link').forEach(el => {
      el.addEventListener('click', (e) => {
        e.preventDefault();
        showTokens(parseInt(el.dataset.id), el.dataset.name);
      });
    });
    tbody.querySelectorAll('.btn-delete-project').forEach(el => {
      el.addEventListener('click', () => deleteProject(parseInt(el.dataset.id)));
    });
  });
}

function createProject() {
  const name = document.getElementById('newProjectName').value.trim();
  if (!name) return;
  API.post('/projects', { name }).then(r => {
    if (r.ok) {
      bootstrap.Modal.getInstance(document.getElementById('createProjectModal')).hide();
      document.getElementById('newProjectName').value = '';
      loadProjects();
      UI.showToast('Project created');
    } else {
      r.json().then(d => UI.showToast(d.error?.message || 'Error', 'danger'));
    }
  });
}

function deleteProject(id) {
  if (!confirm('Delete this project and all its data?')) return;
  API.del(`/projects/${id}`).then(() => {
    loadProjects();
    document.getElementById('tokensSection').style.display = 'none';
    UI.showToast('Project deleted');
  });
}

function showTokens(projectId, projectName) {
  selectedProjectId = projectId;
  document.getElementById('tokensSection').style.display = 'block';
  document.getElementById('tokenProjectName').textContent = projectName;
  loadTokens();
}

function loadTokens() {
  if (!selectedProjectId) return;
  const isAdmin = Auth.getUser()?.role === 'admin';
  API.get(`/projects/${selectedProjectId}/tokens`).then(r => API.json(r)).then(data => {
    const tokens = data?.tokens || [];
    const tbody = document.getElementById('tokensTable');
    if (tokens.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="text-muted">No tokens</td></tr>';
      return;
    }
    tbody.innerHTML = tokens.map(t => `
      <tr>
        <td>${UI.escapeHtml(t.label)}</td>
        <td><code>${t.token.substring(0,8)}...${t.token.substring(56)}</code></td>
        <td><span class="badge bg-${t.active ? 'success' : 'secondary'}">${t.active ? 'Active' : 'Disabled'}</span></td>
        <td>${UI.formatDate(t.created_at)}</td>
        <td class="text-end">
          ${t.active
            ? `<button class="btn btn-sm btn-outline-warning btn-disable-token" data-id="${t.id}">Disable</button>`
            : `<button class="btn btn-sm btn-outline-success btn-enable-token" data-id="${t.id}">Enable</button>`}
          ${isAdmin ? `<button class="btn btn-sm btn-outline-danger ms-1 btn-delete-token" data-id="${t.id}">Delete</button>` : ''}
        </td>
      </tr>
    `).join('');

    tbody.querySelectorAll('.btn-disable-token').forEach(el => {
      el.addEventListener('click', () => disableToken(parseInt(el.dataset.id)));
    });
    tbody.querySelectorAll('.btn-enable-token').forEach(el => {
      el.addEventListener('click', () => enableToken(parseInt(el.dataset.id)));
    });
    tbody.querySelectorAll('.btn-delete-token').forEach(el => {
      el.addEventListener('click', () => deleteToken(parseInt(el.dataset.id)));
    });
  });
}

function createToken() {
  const label = document.getElementById('newTokenLabel').value.trim();
  if (!label) return;
  API.post(`/projects/${selectedProjectId}/tokens`, { label }).then(r => API.json(r)).then(data => {
    if (!data) return;
    bootstrap.Modal.getInstance(document.getElementById('createTokenModal')).hide();
    document.getElementById('newTokenLabel').value = '';
    document.getElementById('newTokenValue').textContent = data.token;
    new bootstrap.Modal(document.getElementById('tokenCreatedModal')).show();
    loadTokens();
  });
}

function copyToken() {
  const token = document.getElementById('newTokenValue').textContent;
  navigator.clipboard.writeText(token).then(() => UI.showToast('Token copied'));
}

function disableToken(id) {
  API.put(`/projects/${selectedProjectId}/tokens/${id}/disable`).then(() => { loadTokens(); UI.showToast('Token disabled'); });
}

function enableToken(id) {
  API.put(`/projects/${selectedProjectId}/tokens/${id}/enable`).then(() => { loadTokens(); UI.showToast('Token enabled'); });
}

function deleteToken(id) {
  if (!confirm('Permanently delete this token?')) return;
  API.del(`/projects/${selectedProjectId}/tokens/${id}`).then(() => { loadTokens(); UI.showToast('Token deleted'); });
}

document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;
  loadProjects();

  // Modal button delegation (no inline onclick on modal buttons)
  document.getElementById('createProjectModal')?.querySelector('.btn-primary')
    ?.addEventListener('click', createProject);
  document.getElementById('createTokenModal')?.querySelector('.btn-primary')
    ?.addEventListener('click', createToken);
  document.getElementById('tokenCreatedModal')?.querySelector('.btn-primary')
    ?.addEventListener('click', copyToken);
});
