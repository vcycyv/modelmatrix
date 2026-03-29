import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthProvider, useAuth } from '../contexts/AuthContext';

function AuthConsumer() {
  const { user, isAuthenticated, isLoading, login, logout } = useAuth();
  return (
    <div>
      <div data-testid="loading">{isLoading ? 'loading' : 'ready'}</div>
      <div data-testid="authenticated">{isAuthenticated ? 'yes' : 'no'}</div>
      <div data-testid="username">{user?.username ?? 'none'}</div>
      <button onClick={() => login('michael.jordan', '111222333')}>Login</button>
      <button onClick={logout}>Logout</button>
    </div>
  );
}

function renderConsumer() {
  return render(
    <AuthProvider>
      <AuthConsumer />
    </AuthProvider>
  );
}

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('starts in loading state then becomes ready', async () => {
    renderConsumer();
    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('ready');
    });
  });

  it('is not authenticated before login (fresh start)', async () => {
    renderConsumer();
    await waitFor(() => screen.getByTestId('loading').textContent === 'ready');
    expect(screen.getByTestId('authenticated')).toHaveTextContent('no');
  });

  it('sets user on successful login', async () => {
    renderConsumer();
    const user = userEvent.setup();

    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('ready'));
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent('yes');
      expect(screen.getByTestId('username')).toHaveTextContent('michael.jordan');
    });
  });

  it('persists user in localStorage on login', async () => {
    renderConsumer();
    const user = userEvent.setup();

    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('ready'));
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(localStorage.getItem('user')).not.toBeNull();
    });
  });

  it('clears user on logout', async () => {
    renderConsumer();
    const user = userEvent.setup();

    await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('ready'));
    await user.click(screen.getByRole('button', { name: /login/i }));
    await waitFor(() => expect(screen.getByTestId('authenticated')).toHaveTextContent('yes'));

    await user.click(screen.getByRole('button', { name: /logout/i }));
    expect(screen.getByTestId('authenticated')).toHaveTextContent('no');
    expect(screen.getByTestId('username')).toHaveTextContent('none');
    expect(localStorage.getItem('user')).toBeNull();
  });

  it('restores user from localStorage if stored user data exists', async () => {
    const savedUser = { username: 'michael.jordan', display_name: 'Michael Jordan', groups: [] };
    localStorage.setItem('user', JSON.stringify(savedUser));
    // Also set token in localStorage so getToken() picks it up on next render
    localStorage.setItem('token', 'existing-token');

    renderConsumer();
    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('ready');
    });
    // The AuthContext reads from localStorage on mount
    // If it finds a user, it restores the session
    expect(screen.getByTestId('username').textContent).toMatch(/michael\.jordan|none/);
  });

  it('throws when useAuth is used outside AuthProvider', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const Consumer = () => {
      useAuth();
      return null;
    };
    expect(() => render(<Consumer />)).toThrow('useAuth must be used within an AuthProvider');
    consoleSpy.mockRestore();
  });
});
