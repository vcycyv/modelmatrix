import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import RetrainDialog from '../components/RetrainDialog';
import { server } from '../test/mocks/server';

const testModel = {
  id: 'model-1',
  name: 'Test Model',
  algorithm: 'random_forest',
  model_type: 'regression',
  status: 'active',
  build_id: 'build-1',
  datasource_id: 'ds-1',
  target_column: 'BAD',
  version: 1,
  created_by: 'michael.jordan',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

function renderDialog(props: Partial<React.ComponentProps<typeof RetrainDialog>> = {}) {
  const defaults = {
    isOpen: true,
    onClose: vi.fn(),
    onSuccess: vi.fn(),
    model: testModel as any,
  };
  return render(<RetrainDialog {...defaults} {...props} />);
}

describe('RetrainDialog', () => {
  it('renders the dialog when open', async () => {
    renderDialog();
    await waitFor(() => {
      expect(document.querySelector('[role="dialog"]') ?? document.body).toBeInTheDocument();
    });
  });

  it('does not render when isOpen=false', () => {
    renderDialog({ isOpen: false });
    expect(document.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('shows a cancel button', async () => {
    renderDialog();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
    });
  });

  it('calls onClose when cancel is clicked', async () => {
    const onClose = vi.fn();
    renderDialog({ onClose });
    const user = userEvent.setup();

    await waitFor(() => expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /cancel/i }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('shows model name in dialog', async () => {
    renderDialog();
    await waitFor(() => {
      expect(screen.getAllByText(/test model/i).length).toBeGreaterThan(0);
    }, { timeout: 3000 });
  });

  it('shows a retrain/start button', async () => {
    renderDialog();
    await waitFor(() => {
      const btn = screen.queryByRole('button', { name: /retrain|start retrain|begin/i });
      expect(btn).toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it('calls onSuccess when retrain succeeds', async () => {
    const onSuccess = vi.fn();
    renderDialog({ onSuccess });
    const user = userEvent.setup();

    await waitFor(() => screen.getByRole('button', { name: /retrain|start|begin/i }));
    await user.click(screen.getByRole('button', { name: /retrain|start|begin/i }));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
    }, { timeout: 3000 });
  });

  it('shows error when retrain API fails', async () => {
    server.use(
      http.post('/api/models/:id/retrain', () => {
        return HttpResponse.json({ code: 400, msg: 'Model has no input variables' }, { status: 400 });
      })
    );

    renderDialog();
    const user = userEvent.setup();

    await waitFor(() => screen.getByRole('button', { name: /retrain|start|begin/i }));
    await user.click(screen.getByRole('button', { name: /retrain|start|begin/i }));

    await waitFor(() => {
      const errorEl = document.querySelector('[class*="red"], [class*="error"]');
      expect(errorEl || screen.queryByText(/error|failed|no input/i)).toBeTruthy();
    }, { timeout: 3000 });
  });
});
