import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Mail, ArrowRight, Loader2, CheckCircle2, AlertCircle, ArrowLeft } from 'lucide-react';
import { Link } from 'react-router-dom';

export const ForgotPassword = () => {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<{type: 'error' | 'success', msg: string} | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setStatus(null);

    const BASE_URL = import.meta.env.VITE_API_URL || 'https://go-auth-service-6j6d.onrender.com/api/v1';

    try {
      const response = await fetch(`${BASE_URL}/auth/forgot-password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email })
      });

      if (!response.ok) {
        throw new Error('Something went wrong. Please try again.');
      }

      setStatus({
        type: 'success',
        msg: 'If that email is registered, a reset link has been sent. Check the backend terminal logs.'
      });
      setEmail('');
    } catch (err: any) {
      setStatus({ type: 'error', msg: err.message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-container">
      <Link to="/login" style={{ position: 'absolute', top: '1.5rem', left: '1.5rem', color: 'var(--text-muted)', textDecoration: 'none', display: 'flex', alignItems: 'center', gap: '4px', fontSize: '0.9rem' }}>
        <ArrowLeft size={16} /> Back
      </Link>
      
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        style={{ marginTop: '2rem' }}
      >
        <h1>Reset Password</h1>
        <p className="subtitle">
          Enter your email address and we'll send you a link to reset your password.
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
            <label>Email</label>
            <div className="input-wrapper">
              <Mail className="input-icon" />
              <input
                type="email"
                placeholder="Enter your email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
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
                Send Reset Link
                <ArrowRight size={18} />
              </div>
            )}
          </motion.button>
        </form>
      </motion.div>
    </div>
  );
};
