import React from 'react';
import { useAuth } from '../context/AuthContext';
import { LogOut, User as UserIcon, Shield } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

export const Dashboard = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  return (
    <div className="auth-container" style={{ maxWidth: '600px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem' }}>
        <h1 style={{ fontSize: '1.8rem', margin: 0, textAlign: 'left' }}>Dashboard</h1>
        <button 
          onClick={logout} 
          style={{ background: 'transparent', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.9rem', fontWeight: 500 }}
        >
          <LogOut size={18} /> Logout
        </button>
      </div>

      <div style={{ background: 'rgba(0,0,0,0.2)', padding: '2rem', borderRadius: '16px', border: '1px solid var(--glass-border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '1.25rem', marginBottom: '1.5rem' }}>
          <div style={{ width: '60px', height: '60px', borderRadius: '30px', background: 'var(--accent)', display: 'flex', justifyContent: 'center', alignItems: 'center', boxShadow: '0 4px 12px rgba(99, 102, 241, 0.3)' }}>
            <UserIcon size={28} color="white" />
          </div>
          <div>
            <h3 style={{ margin: 0, fontSize: '1.25rem', fontWeight: 600 }}>{user?.username}</h3>
            <p style={{ color: 'var(--text-muted)', margin: 0, fontSize: '0.95rem', marginTop: '4px' }}>{user?.email}</p>
          </div>
        </div>

        <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
          <span style={{ padding: '6px 14px', borderRadius: '20px', fontSize: '0.85rem', background: 'rgba(255,255,255,0.05)', border: '1px solid var(--glass-border)' }}>
            Role: <strong style={{ color: user?.role === 'admin' ? '#d8b4fe' : 'inherit' }}>{user?.role}</strong>
          </span>
          <span style={{ padding: '6px 14px', borderRadius: '20px', fontSize: '0.85rem', background: user?.email_verified ? 'rgba(16, 185, 129, 0.1)' : 'rgba(239, 68, 68, 0.1)', color: user?.email_verified ? '#6ee7b7' : '#fca5a5', border: `1px solid ${user?.email_verified ? 'rgba(16, 185, 129, 0.2)' : 'rgba(239, 68, 68, 0.2)'}` }}>
            {user?.email_verified ? 'Email Verified' : 'Email Unverified'}
          </span>
          <span style={{ padding: '6px 14px', borderRadius: '20px', fontSize: '0.85rem', background: 'rgba(255,255,255,0.05)', border: '1px solid var(--glass-border)' }}>
            Joined: {new Date(user?.created_at || '').toLocaleDateString()}
          </span>
        </div>
      </div>

      {user?.role === 'admin' && (
        <div style={{ marginTop: '2rem' }}>
          <button 
            onClick={() => navigate('/admin')}
            className="btn-submit" 
            style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', background: 'transparent', border: '1px solid var(--accent)', color: 'var(--text-main)' }}
          >
            <Shield size={18} color="var(--accent)" /> Go to Admin Panel
          </button>
        </div>
      )}
    </div>
  );
};
