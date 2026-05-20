import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Mail, Lock, User, ArrowRight, Loader2, CheckCircle2, AlertCircle } from 'lucide-react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export const AuthPage = () => {
  const [isLogin, setIsLogin] = useState(true);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<{type: 'error' | 'success', msg: string} | null>(null);
  
  const { login } = useAuth();
  const navigate = useNavigate();

  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: ''
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setStatus(null);

    const endpoint = isLogin ? '/auth/login' : '/auth/register';
    const BASE_URL = import.meta.env.VITE_API_URL || 'https://go-auth-service-6j6d.onrender.com/api/v1';
    
    try {
      const response = await fetch(`${BASE_URL}${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(isLogin ? {
          username: formData.username,
          password: formData.password
        } : formData)
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Something went wrong');
      }

      if (isLogin) {
        await login(data.access_token, data.refresh_token);
        navigate('/dashboard');
      } else {
        setStatus({
          type: 'success',
          msg: 'Account created! Check the backend terminal logs for your verification link.'
        });
        setIsLogin(true); 
      }
      
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
        <h1>{isLogin ? 'Welcome Back' : 'Create Account'}</h1>
        <p className="subtitle">
          {isLogin ? 'Enter your details to access your account' : 'Join us to get started'}
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
            <label>Username</label>
            <div className="input-wrapper">
              <User className="input-icon" />
              <input
                type="text"
                name="username"
                placeholder="Enter your username"
                value={formData.username}
                onChange={handleChange}
                required
              />
            </div>
          </div>

          <AnimatePresence mode="wait">
            {!isLogin && (
              <motion.div
                initial={{ opacity: 0, height: 0, overflow: 'hidden' }}
                animate={{ opacity: 1, height: 'auto', overflow: 'visible' }}
                exit={{ opacity: 0, height: 0, overflow: 'hidden' }}
                className="form-group"
              >
                <label>Email</label>
                <div className="input-wrapper">
                  <Mail className="input-icon" />
                  <input
                    type="email"
                    name="email"
                    placeholder="Enter your email"
                    value={formData.email}
                    onChange={handleChange}
                    required={!isLogin}
                  />
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          <div className="form-group">
            <label>Password</label>
            <div className="input-wrapper">
              <Lock className="input-icon" />
              <input
                type="password"
                name="password"
                placeholder="••••••••"
                value={formData.password}
                onChange={handleChange}
                required
              />
            </div>
          </div>

          {isLogin && (
            <div style={{ textAlign: 'right', marginBottom: '1rem', marginTop: '-0.5rem' }}>
              <Link to="/forgot-password" style={{ color: 'var(--text-muted)', fontSize: '0.85rem', textDecoration: 'none' }}>
                Forgot Password?
              </Link>
            </div>
          )}

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
                {isLogin ? 'Sign In' : 'Create Account'}
                <ArrowRight size={18} />
              </div>
            )}
          </motion.button>
        </form>

        <div className="toggle-text">
          {isLogin ? "Don't have an account? " : "Already have an account? "}
          <span 
            className="toggle-link" 
            onClick={() => {
              setIsLogin(!isLogin);
              setStatus(null);
            }}
          >
            {isLogin ? 'Register here' : 'Sign in here'}
          </span>
        </div>
      </motion.div>
    </div>
  );
};
