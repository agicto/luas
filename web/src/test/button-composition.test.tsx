import { fireEvent, render, screen } from '@testing-library/react';
import { ArrowRight } from 'lucide-react';
import { describe, expect, it, vi } from 'vitest';

import { Button } from '@/components/ui/button';

describe('Button composition contract', () => {
  it('keeps icon and label content in one horizontal line', () => {
    render(
      <Button
        icon={<ArrowRight aria-hidden="true" />}
        className="flex-col flex-wrap whitespace-normal"
      >
        Open preview
      </Button>
    );

    expect(screen.getByRole('button', { name: 'Open preview' })).toHaveClass(
      'flex-row!',
      'flex-nowrap!',
      'whitespace-nowrap!'
    );
  });

  it('applies the button contract directly to the composed link', () => {
    const { container } = render(
      <Button asChild className="contract-marker">
        <a href="/target">Open target</a>
      </Button>
    );

    const link = screen.getByRole('link', { name: 'Open target' });

    expect(container.firstElementChild).toBe(link);
    expect(link).toHaveAttribute('data-slot', 'button');
    expect(link).toHaveClass('inline-flex', 'px-4', 'contract-marker');
  });

  it('keeps leading and trailing content inside the composed link hit area', () => {
    render(
      <Button asChild icon={<ArrowRight data-testid="leading-icon" aria-hidden="true" />}>
        <a href="/continue">Continue</a>
      </Button>
    );

    const link = screen.getByRole('link', { name: 'Continue' });
    const icon = screen.getByTestId('leading-icon');

    expect(link).toContainElement(icon);
    expect(link).toHaveAccessibleName('Continue');
    expect(icon.nextSibling).toHaveTextContent('Continue');
  });

  it('preserves native disabled and loading semantics for a button', () => {
    const onClick = vi.fn();

    render(
      <Button loading onClick={onClick}>
        Save changes
      </Button>
    );

    const button = screen.getByRole('button', { name: 'Save changes' });
    const spinner = button.querySelector('.animate-spin');

    expect(button).toBeDisabled();
    expect(button).toHaveAttribute('aria-busy', 'true');
    expect(spinner).toHaveAttribute('aria-hidden', 'true');

    fireEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('makes a disabled composed link inert without invalid HTML attributes', () => {
    const childClick = vi.fn();
    const childClickCapture = vi.fn();
    const childKeyDown = vi.fn();
    const childKeyDownCapture = vi.fn();
    const buttonClick = vi.fn();
    const buttonKeyDown = vi.fn();

    render(
      <Button asChild disabled onClick={buttonClick} onKeyDown={buttonKeyDown}>
        <a
          href="/billing"
          aria-disabled="false"
          tabIndex={0}
          onClick={childClick}
          onClickCapture={childClickCapture}
          onKeyDown={childKeyDown}
          onKeyDownCapture={childKeyDownCapture}
        >
          Billing
        </a>
      </Button>
    );

    const link = screen.getByRole('link', { name: 'Billing' });

    expect(link).toHaveAttribute('aria-disabled', 'true');
    expect(link).toHaveAttribute('tabindex', '-1');
    expect(link).not.toHaveAttribute('disabled');
    expect(fireEvent.click(link)).toBe(false);
    expect(fireEvent.keyDown(link, { key: 'Enter' })).toBe(false);
    expect(fireEvent.keyDown(link, { key: ' ' })).toBe(false);
    expect(childClick).not.toHaveBeenCalled();
    expect(childClickCapture).not.toHaveBeenCalled();
    expect(childKeyDown).not.toHaveBeenCalled();
    expect(childKeyDownCapture).not.toHaveBeenCalled();
    expect(buttonClick).not.toHaveBeenCalled();
    expect(buttonKeyDown).not.toHaveBeenCalled();
  });

  it('keeps loading feedback inside an inert composed link', () => {
    render(
      <Button asChild loading iconPosition="right">
        <a href="/next">Continue setup</a>
      </Button>
    );

    const link = screen.getByRole('link', { name: 'Continue setup' });
    const spinner = link.querySelector('.animate-spin');

    expect(link).toHaveAttribute('aria-busy', 'true');
    expect(link).toHaveAttribute('aria-disabled', 'true');
    expect(link).toHaveAttribute('tabindex', '-1');
    expect(spinner).toHaveAttribute('aria-hidden', 'true');
    expect(link.lastElementChild).toBe(spinner);
    expect(fireEvent.click(link)).toBe(false);
  });
});
