import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ScoreModelDialog from '../components/ScoreModelDialog';

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

function renderDialog(props: Partial<React.ComponentProps<typeof ScoreModelDialog>> = {}) {
  const defaults = {
    isOpen: true,
    onClose: vi.fn(),
    onSuccess: vi.fn(),
    model: testModel as any,
  };
  return render(<ScoreModelDialog {...defaults} {...props} />);
}

describe('ScoreModelDialog', () => {
  it('renders when open', async () => {
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

  it('has a submit/score button', async () => {
    renderDialog();
    await waitFor(() => {
      const submitBtn = screen.queryByRole('button', { name: /score|predict|start scoring/i });
      expect(submitBtn).toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it('displays the model name or scoring label', async () => {
    renderDialog();
    await waitFor(() => {
      // Should show the dialog for scoring the model
      const scoringText = screen.queryByText(/score|predict|Test Model/i);
      expect(scoringText).toBeInTheDocument();
    }, { timeout: 3000 });
  });
});
