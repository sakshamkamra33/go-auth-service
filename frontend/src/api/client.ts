const BASE_URL = import.meta.env.VITE_API_URL || 'https://go-auth-service-6j6d.onrender.com/api/v1';

export const getAccessToken = () => localStorage.getItem('access_token');
export const getRefreshToken = () => localStorage.getItem('refresh_token');

export const setTokens = (access: string, refresh: string) => {
  localStorage.setItem('access_token', access);
  localStorage.setItem('refresh_token', refresh);
};

export const clearTokens = () => {
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
};

let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];

const subscribeTokenRefresh = (cb: (token: string) => void) => {
  refreshSubscribers.push(cb);
};

const onRefreshed = (token: string) => {
  refreshSubscribers.map(cb => cb(token));
  refreshSubscribers = [];
};

export async function fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
  const token = getAccessToken();
  
  const headers = new Headers(options.headers || {});
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const config: RequestInit = {
    ...options,
    headers,
  };

  let response = await fetch(`${BASE_URL}${url}`, config);

  if (response.status === 401) {
    const refreshToken = getRefreshToken();
    if (!refreshToken) {
      clearTokens();
      window.location.href = '/login';
      return response;
    }

    if (!isRefreshing) {
      isRefreshing = true;
      try {
        const refreshResponse = await fetch(`${BASE_URL}/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });

        if (!refreshResponse.ok) {
          clearTokens();
          window.location.href = '/login';
          throw new Error('Refresh failed');
        }

        const data = await refreshResponse.json();
        setTokens(data.access_token, data.refresh_token);
        isRefreshing = false;
        onRefreshed(data.access_token);
      } catch (err) {
        isRefreshing = false;
        clearTokens();
        window.location.href = '/login';
        throw err;
      }
    }

    // Wait for the refresh to finish and retry the original request
    return new Promise((resolve) => {
      subscribeTokenRefresh((newToken: string) => {
        headers.set('Authorization', `Bearer ${newToken}`);
        resolve(fetch(`${BASE_URL}${url}`, { ...config, headers }));
      });
    });
  }

  return response;
}
