import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ConfirmDialog from '../components/ConfirmDialog';

function renderDialog(props: Partial<Parameters<typeof ConfirmDialog>[0]> = {}) {
  const defaults = {
    isOpen: true,
    onClose: vi.fn(),
    onConfirm: vi.fn(),
    title: 'Delete Item',
    message: 'Are you sure you want to delete this item?',
  };
  return render(<ConfirmDialog {...defaults} {...props} />);
}

describe('ConfirmDialog', () => {
  it('renders title and message when open', () => {
    renderDialog();
    expect(screen.getByText('Delete Item')).toBeInTheDocument();
    expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
  });

  it('does not render when isOpen=false', () => {
    renderDialog({ isOpen: false });
    expect(screen.queryByText('Delete Item')).not.toBeInTheDocument();
  });

  it('calls onConfirm when confirm button is clicked', async () => {
    const onConfirm = vi.fn();
    renderDialog({ onConfirm });
    const user = userEvent.setup();

    await user.click(screen.getByRole('button', { name: /confirm/i }));
    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it('calls onClose when cancel button is clicked', async () => {
    const onClose = vi.fn();
    renderDialog({ onClose });
    const user = userEvent.setup();

    await user.click(screen.getByRole('button', { name: /cancel/i }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('renders custom confirmText', () => {
    renderDialog({ confirmText: 'Delete Forever' });
    expect(screen.getByRole('button', { name: /delete forever/i })).toBeInTheDocument();
  });

  it('disables the cancel button while loading', () => {
    renderDialog({ isLoading: true });
    // The Cancel button explicitly has disabled={isLoading}
    expect(screen.getByRole('button', { name: /cancel/i })).toBeDisabled();
  });

  it('shows both Cancel and Confirm buttons', () => {
    renderDialog();
    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /confirm/i })).toBeInTheDocument();
  });
});
