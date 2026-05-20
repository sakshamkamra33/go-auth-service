import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Lock, Key, ArrowRight, Loader2, CheckCircle2, AlertCircle } from 'lucide-react';
import { Link, useSearchParams } from 'react-router-dom';

export const ResetPassword = () => {
  const [searchParams] = useSearchParams();
  const tokenParam = searchParams.get('token') || '';

  const [token, setToken] = useState(tokenParam);
  const [newPassword, setNewPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<{type: 'error' | 'success', msg: string} | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setStatus(null);

    const BASE_URL = import.meta.env.VITE_API_URL || 'https://go-auth-service-6j6d.onrender.com/api/v1';

    try {
      const response = await fetch(`${BASE_URL}/auth/reset-password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, new_password: newPassword })
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to reset password');
      }

      setStatus({
        type: 'success',
        msg: 'Password reset successfully! You can now log in.'
      });
      setNewPassword('');
    } catch (err: any) {
      setStatus({ type: 'error', msg: err.message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-container">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
      >
        <h1>Set New Password</h1>
        <p className="subtitle">
          Enter the token you received and your new password.
        </p>

        <AnimatePresence mode="wait">
          {status && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              className={`alert ${status.type}`}
            >
              {status.type === 'error' ? <AlertCircle size={18} /> : <CheckCircle2 size={18} />}
              {status.msg}
            </motion.div>
          )}
        </AnimatePresence>

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Reset Token</label>
            <div className="input-wrapper">
              <Key className="input-icon" />
              <input
                type="text"
                placeholder="Paste your reset token here"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                required
              />
            </div>
          </div>

          <div className="form-group">
            <label>New Password</label>
            <div className="input-wrapper">
              <Lock className="input-icon" />
              <input
                type="password"
                placeholder="••••••••"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
              />
            </div>
          </div>

          <motion.button
            whileTap={{ scale: 0.98 }}
            className="btn-submit"
            type="submit"
            disabled={loading}
          >
            {loading ? (
              <Loader2 className="spinner" />
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}>
                Reset Password
                <ArrowRight size={18} />
              </div>
            )}
          </motion.button>
        </form>

        <div className="toggle-text">
          <Link to="/login" className="toggle-link" style={{ textDecoration: 'none' }}>
            Back to login
          </Link>
        </div>
      </motion.div>
    </div>
  );
};
