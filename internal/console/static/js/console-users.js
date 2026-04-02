function loadUsers() {
  const currentUser = Auth.getUser();
  const isAdmin = currentUser?.role === 'admin';
  const myId = currentUser?.id;
  API.get('/users').then(r => API.json(r)).then(users => {
    const tbody = document.getElementById('usersTable');
    if (!users || users.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="text-muted">No users</td></tr>';
      return;
    }
    const isSelf = (id) => id === myId;
    tbody.innerHTML = users.map(u => `
      <tr>
        <td>${UI.escapeHtml(u.username)}${isSelf(u.id) ? ' <span class="text-muted">(you)</span>' : ''}</td>
        <td>${u.email ? UI.escapeHtml(u.email) : '—'}</td>
        <td><span class="badge bg-${u.role === 'admin' ? 'danger' : u.role === 'manager' ? 'warning' : 'info'}">${UI.escapeHtml(u.role)}</span></td>
        <td>${u.totp_enabled ? '<span class="badge bg-success">On</span>' : '<span class="badge bg-secondary">Off</span>'}</td>
        <td>${UI.formatDate(u.created_at)}</td>
        ${isAdmin ? `<td class="text-end">
          ${isSelf(u.id) ? '' : `<select class="form-select form-select-sm d-inline-block role-select" style="width:auto" data-id="${u.id}">
            <option value="viewer" ${u.role==='viewer'?'selected':''}>Viewer</option>
            <option value="manager" ${u.role==='manager'?'selected':''}>Manager</option>
            <option value="admin" ${u.role==='admin'?'selected':''}>Admin</option>
          </select>
          <button class="btn btn-sm btn-outline-danger ms-1 btn-delete-user" data-id="${u.id}" data-name="${UI.escapeHtml(u.username)}">Delete</button>`}
        </td>` : '<td></td>'}
      </tr>
    `).join('');

    tbody.querySelectorAll('.role-select').forEach(el => {
      el.addEventListener('change', () => changeRole(parseInt(el.dataset.id), el.value));
    });
    tbody.querySelectorAll('.btn-delete-user').forEach(el => {
      el.addEventListener('click', () => deleteUser(parseInt(el.dataset.id), el.dataset.name));
    });
  });

  // Hide create button for non-admins
  if (!isAdmin) {
    const btn = document.querySelector('[data-bs-target="#createUserModal"]');
    if (btn) btn.style.display = 'none';
  }
}

function createUser() {
  const username = document.getElementById('newUsername').value.trim();
  const password = document.getElementById('newPassword').value;
  const role = document.getElementById('newRole').value;
  const email = document.getElementById('newEmail').value.trim() || null;
  if (!username || !password) return;

  API.post('/users', { username, password, role, email }).then(r => {
    if (r.ok) {
      bootstrap.Modal.getInstance(document.getElementById('createUserModal')).hide();
      document.getElementById('newUsername').value = '';
      document.getElementById('newPassword').value = '';
      document.getElementById('newEmail').value = '';
      loadUsers();
      UI.showToast('User created');
    } else {
      r.json().then(d => UI.showToast(d.error?.message || 'Error', 'danger'));
    }
  });
}

function changeRole(id, role) {
  API.put(`/users/${id}`, { role }).then(r => {
    if (r.ok) UI.showToast('Role updated');
    else r.json().then(d => UI.showToast(d.error?.message || 'Error', 'danger'));
  });
}

function deleteUser(id, username) {
  if (!confirm(`Delete user "${username}"?`)) return;
  API.del(`/users/${id}`).then(() => { loadUsers(); UI.showToast('User deleted'); });
}

document.addEventListener('DOMContentLoaded', () => {
  if (!Auth.requireAuth()) return;
  loadUsers();

  document.getElementById('createUserModal')?.querySelector('.btn-primary')
    ?.addEventListener('click', createUser);
});
