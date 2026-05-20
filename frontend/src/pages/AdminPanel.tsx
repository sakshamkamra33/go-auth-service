import React, { useEffect, useState } from 'react';
import { useAuth, type User } from '../context/AuthContext';
import { ArrowLeft, Trash2, Users } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { fetchWithAuth } from '../api/client';

export const AdminPanel = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  const fetchUsers = async () => {
    try {
      const res = await fetchWithAuth('/admin/users');
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || []);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this user?')) return;
    try {
      await fetchWithAuth(`/admin/users/${id}`, { method: 'DELETE' });
      setUsers(users.filter(u => u.id !== id));
    } catch (e) {
      alert('Failed to delete user');
    }
  };

  return (
    <div className="auth-container" style={{ maxWidth: '800px', padding: '2rem' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '2rem' }}>
        <button 
          onClick={() => navigate('/dashboard')} 
          style={{ background: 'rgba(255,255,255,0.05)', border: '1px solid var(--glass-border)', borderRadius: '8px', padding: '8px', color: 'var(--text-muted)', cursor: 'pointer', display: 'flex', alignItems: 'center' }}
        >
          <ArrowLeft size={20} />
        </button>
        <h1 style={{ margin: 0, fontSize: '1.5rem', textAlign: 'left', display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Users size={24} color="var(--accent)" /> Admin Panel
        </h1>
      </div>

      <div style={{ background: 'rgba(0,0,0,0.2)', borderRadius: '12px', border: '1px solid var(--glass-border)', overflowX: 'auto' }}>
        {loading ? (
          <div style={{ padding: '3rem', textAlign: 'center', color: 'var(--text-muted)' }}>Loading users...</div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '0.9rem' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid var(--glass-border)', background: 'rgba(255,255,255,0.03)' }}>
                <th style={{ padding: '1.25rem 1rem', color: 'var(--text-muted)', fontWeight: 500 }}>Username</th>
                <th style={{ padding: '1.25rem 1rem', color: 'var(--text-muted)', fontWeight: 500 }}>Email</th>
                <th style={{ padding: '1.25rem 1rem', color: 'var(--text-muted)', fontWeight: 500 }}>Role</th>
                <th style={{ padding: '1.25rem 1rem', color: 'var(--text-muted)', fontWeight: 500 }}>Status</th>
                <th style={{ padding: '1.25rem 1rem', color: 'var(--text-muted)', fontWeight: 500 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id} style={{ borderBottom: '1px solid rgba(255,255,255,0.02)' }}>
                  <td style={{ padding: '1rem' }}>{u.username}</td>
                  <td style={{ padding: '1rem', color: 'var(--text-muted)' }}>{u.email}</td>
                  <td style={{ padding: '1rem' }}>
                    <span style={{ padding: '4px 10px', borderRadius: '12px', fontSize: '0.8rem', background: u.role === 'admin' ? 'rgba(168, 85, 247, 0.1)' : 'rgba(255,255,255,0.05)', color: u.role === 'admin' ? '#d8b4fe' : 'inherit', border: u.role === 'admin' ? '1px solid rgba(168, 85, 247, 0.2)' : '1px solid transparent' }}>
                      {u.role}
                    </span>
                  </td>
                  <td style={{ padding: '1rem' }}>
                    <span style={{ color: u.email_verified ? '#6ee7b7' : '#fca5a5', fontSize: '0.85rem' }}>
                      {u.email_verified ? 'Verified' : 'Pending'}
                    </span>
                  </td>
                  <td style={{ padding: '1rem' }}>
                    <button 
                      onClick={() => handleDelete(u.id)}
                      style={{ background: 'none', border: 'none', color: '#fca5a5', cursor: 'pointer', padding: '4px' }}
                      disabled={u.role === 'admin'}
                      title={u.role === 'admin' ? "Cannot delete admin" : "Delete user"}
                    >
                      <Trash2 size={18} opacity={u.role === 'admin' ? 0.3 : 1} />
                    </button>
                  </td>
                </tr>
              ))}
              {users.length === 0 && (
                <tr>
                  <td colSpan={5} style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)' }}>
                    No users found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};
