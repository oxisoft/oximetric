// Oximetric Console — Core Application

const API = {
  async request(method, path, body) {
    const opts = {
      method,
      headers: { 'Content-Type': 'application/json' },
    };
    const token = sessionStorage.getItem('jwt');
    if (token) opts.headers['Authorization'] = 'Bearer ' + token;
    if (body) opts.body = JSON.stringify(body);

    const resp = await fetch('/api/v1' + path, opts);
    if (resp.status === 401) {
      sessionStorage.removeItem('jwt');
      window.location.href = '/';
      return null;
    }
    return resp;
  },

  async get(path) { return this.request('GET', path); },
  async post(path, body) { return this.request('POST', path, body); },
  async put(path, body) { return this.request('PUT', path, body); },
  async del(path) { return this.request('DELETE', path); },

  async json(resp) {
    if (!resp || !resp.ok) return null;
    return resp.json();
  }
};

const Auth = {
  isLoggedIn() { return !!sessionStorage.getItem('jwt'); },

  getUser() {
    const u = sessionStorage.getItem('user');
    return u ? JSON.parse(u) : null;
  },

  setSession(token, user) {
    sessionStorage.setItem('jwt', token);
    sessionStorage.setItem('user', JSON.stringify(user));
  },

  logout() {
    API.post('/auth/logout');
    sessionStorage.removeItem('jwt');
    sessionStorage.removeItem('user');
    window.location.href = '/';
  },

  requireAuth() {
    if (!this.isLoggedIn()) {
      window.location.href = '/';
      return false;
    }
    return true;
  }
};

const UI = {
  showToast(message, type = 'success') {
    let container = document.querySelector('.toast-container');
    if (!container) {
      container = document.createElement('div');
      container.className = 'toast-container';
      document.body.appendChild(container);
    }
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-bg-${type} border-0 show`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
      <div class="d-flex">
        <div class="toast-body">${message}</div>
        <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
      </div>`;
    container.appendChild(toast);
    setTimeout(() => toast.remove(), 4000);
  },

  escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  },

  formatNumber(n) {
    if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
    if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
    return String(n);
  },

  formatDate(d) {
    return new Date(d).toLocaleDateString();
  },

  setActiveNav(path) {
    document.querySelectorAll('.sidebar-link').forEach(el => {
      el.classList.toggle('active', el.getAttribute('href') === path);
    });
  },

  initProjectSelector(onChange) {
    const sel = document.getElementById('projectSelect');
    if (!sel) return;

    API.get('/projects').then(r => API.json(r)).then(projects => {
      if (!projects || projects.length === 0) {
        sel.innerHTML = '<option>No projects</option>';
        return;
      }
      sel.innerHTML = projects.map(p =>
        `<option value="${p.id}">${p.name}</option>`
      ).join('');

      const saved = sessionStorage.getItem('selectedProject');
      if (saved && projects.find(p => p.id == saved)) {
        sel.value = saved;
      }
      sel.addEventListener('change', () => {
        sessionStorage.setItem('selectedProject', sel.value);
        if (onChange) onChange(parseInt(sel.value));
      });
      if (onChange) onChange(parseInt(sel.value));
    });
  },

  getSelectedProject() {
    const sel = document.getElementById('projectSelect');
    return sel ? parseInt(sel.value) : null;
  },

  dateRange() {
    const from = document.getElementById('dateFrom');
    const to = document.getElementById('dateTo');
    const f = from ? from.value : '';
    const t = to ? to.value : '';
    const fromISO = f ? new Date(f).toISOString() : new Date(Date.now() - 30*86400000).toISOString();
    const toISO = t ? new Date(t + 'T23:59:59').toISOString() : new Date().toISOString();
    return { from: fromISO, to: toISO };
  },

  queryString() {
    const { from, to } = this.dateRange();
    return `from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`;
  }
};

// Init page
document.addEventListener('DOMContentLoaded', () => {
  const user = Auth.getUser();

  // Sidebar user info
  const usernameEl = document.getElementById('sidebarUsername');
  if (usernameEl && user) {
    usernameEl.textContent = user.username;
  }
  const roleEl = document.getElementById('sidebarRole');
  if (roleEl && user) {
    const colors = { admin: 'danger', manager: 'warning', viewer: 'info' };
    roleEl.className = `badge bg-${colors[user.role] || 'secondary'}`;
    roleEl.textContent = user.role;
  }

  const logoutBtn = document.getElementById('logoutBtn');
  if (logoutBtn) {
    logoutBtn.addEventListener('click', (e) => { e.preventDefault(); Auth.logout(); });
  }

  // Mobile sidebar toggle
  const sidebar = document.getElementById('sidebar');
  const sidebarToggle = document.getElementById('sidebarToggle');
  const sidebarOverlay = document.getElementById('sidebarOverlay');
  if (sidebarToggle && sidebar) {
    sidebarToggle.addEventListener('click', () => {
      sidebar.classList.toggle('show');
      sidebarOverlay.classList.toggle('show');
    });
    if (sidebarOverlay) {
      sidebarOverlay.addEventListener('click', () => {
        sidebar.classList.remove('show');
        sidebarOverlay.classList.remove('show');
      });
    }
    // Close sidebar on nav click (mobile)
    sidebar.querySelectorAll('.sidebar-link').forEach(link => {
      link.addEventListener('click', () => {
        if (window.innerWidth <= 768) {
          sidebar.classList.remove('show');
          sidebarOverlay.classList.remove('show');
        }
      });
    });
  }

  // Role-based sidebar visibility
  if (user) {
    if (user.role === 'viewer') {
      // Viewers: hide projects and console users
      document.querySelectorAll('.manager-only').forEach(el => el.classList.add('d-hidden'));
      document.querySelectorAll('.admin-only').forEach(el => el.classList.add('d-hidden'));
    } else if (user.role === 'manager') {
      // Managers: hide admin-only items
      document.querySelectorAll('.admin-only').forEach(el => el.classList.add('d-hidden'));
    }
  }

  UI.setActiveNav(window.location.pathname);

  // Hide analytics controls on management pages
  const managementPages = ['/projects', '/console-users', '/account', '/help', '/about'];
  const controls = document.getElementById('analyticsControls');
  if (controls && managementPages.includes(window.location.pathname)) {
    controls.style.display = 'none';
  }

  // Date filter change
  const dateFrom = document.getElementById('dateFrom');
  const dateTo = document.getElementById('dateTo');
  if (dateFrom) {
    if (!dateFrom.value) {
      const d = new Date(Date.now() - 30*86400000);
      dateFrom.value = d.toISOString().split('T')[0];
    }
    if (!dateTo.value) {
      dateTo.value = new Date().toISOString().split('T')[0];
    }
  }
});
