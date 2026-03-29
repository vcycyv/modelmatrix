import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import LoginPage from '../pages/LoginPage';
import { AuthProvider } from '../contexts/AuthContext';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return { ...actual, useNavigate: () => mockNavigate };
});

function renderLoginPage() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <LoginPage />
      </AuthProvider>
    </MemoryRouter>
  );
}

describe('LoginPage', () => {
  it('renders username and password inputs', () => {
    renderLoginPage();
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
  });

  it('renders the ModelMatrix heading', () => {
    renderLoginPage();
    expect(screen.getByText('ModelMatrix')).toBeInTheDocument();
  });

  it('renders LDAP credentials hint', () => {
    renderLoginPage();
    expect(screen.getByText(/ldap credentials/i)).toBeInTheDocument();
  });

  it('navigates to / after successful login', async () => {
    renderLoginPage();
    const user = userEvent.setup();

    await user.type(screen.getByLabelText(/username/i), 'michael.jordan');
    await user.type(screen.getByLabelText(/password/i), '111222333');
    await user.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/');
    });
  });

  it('displays an error message on invalid credentials', async () => {
    renderLoginPage();
    const user = userEvent.setup();

    await user.type(screen.getByLabelText(/username/i), 'wrong.user');
    await user.type(screen.getByLabelText(/password/i), 'wrongpass');
    await user.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => {
      // The API client throws Error('Unauthorized') on 401
      const errorElement = document.querySelector('[class*="red"]') || document.querySelector('[role="alert"]');
      expect(errorElement || screen.queryByText(/login failed|unauthorized|invalid/i)).toBeTruthy();
    });
  });
});
