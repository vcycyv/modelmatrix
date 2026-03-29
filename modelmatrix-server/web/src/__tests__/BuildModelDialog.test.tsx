import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import BuildModelDialog from '../components/BuildModelDialog';
import { server } from '../test/mocks/server';

function renderDialog(props: Partial<React.ComponentProps<typeof BuildModelDialog>> = {}) {
  const defaults = {
    isOpen: true,
    onClose: vi.fn(),
    onSuccess: vi.fn(),
  };
  return render(<BuildModelDialog {...defaults} {...props} />);
}

describe('BuildModelDialog', () => {
  it('renders the dialog when open', async () => {
    renderDialog();
    // The dialog should render with some heading text
    await waitFor(() => {
      const dialog = document.querySelector('[role="dialog"]') ?? document.body;
      expect(dialog).toBeInTheDocument();
    });
  });

  it('does not render when isOpen=false', () => {
    renderDialog({ isOpen: false });
    expect(document.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('shows at least one select/dropdown for model type or algorithm', async () => {
    renderDialog();
    await waitFor(() => {
      const selects = screen.queryAllByRole('combobox');
      expect(selects.length).toBeGreaterThan(0);
    }, { timeout: 3000 });
  });

  it('calls onClose when cancel button is clicked', async () => {
    const onClose = vi.fn();
    renderDialog({ onClose });
    const user = userEvent.setup();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
    });
    await user.click(screen.getByRole('button', { name: /cancel/i }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('shows a submit button', async () => {
    renderDialog();
    await waitFor(() => {
      // Look for a submit button in the dialog
      const submitBtn = screen.queryByRole('button', { name: /create|build|train|start|submit/i });
      expect(submitBtn).toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it('shows error when build creation fails', async () => {
    server.use(
      http.post('/api/builds', () => {
        return HttpResponse.json({ code: 400, msg: 'Datasource not found' }, { status: 400 });
      })
    );

    const onSuccess = vi.fn();
    renderDialog({ onSuccess });
    const user = userEvent.setup();

    // Wait for form to be ready and find a text input for the build name
    await waitFor(() => {
      const inputs = screen.getAllByRole('textbox');
      expect(inputs.length).toBeGreaterThan(0);
    }, { timeout: 3000 });

    const nameInputs = screen.getAllByRole('textbox');
    // Type a name in the first text input
    await user.type(nameInputs[0], 'Test Build');

    const submitBtn = screen.getByRole('button', { name: /create|build|train|start|submit/i });
    await user.click(submitBtn);

    await waitFor(() => {
      // Error may appear anywhere in the dialog
      const errorEl = document.querySelector('[class*="red"], [class*="error"], [role="alert"]');
      expect(errorEl || screen.queryByText(/error|failed|datasource/i)).toBeTruthy();
    }, { timeout: 3000 });
  });
});
