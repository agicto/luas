import type { ComponentProps } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, expectTypeOf, it } from 'vitest';

import { ColorPicker, Input, PasswordInput } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';

describe('form control accessibility contracts', () => {
  it('preserves native input semantics for specialized HTML types', () => {
    render(
      <>
        <Label htmlFor="birthday">Birthday</Label>
        <Input id="birthday" name="birthday" type="date" required />
      </>
    );

    const input = screen.getByLabelText('Birthday');

    expect(input).toHaveAttribute('type', 'date');
    expect(input).toHaveAttribute('name', 'birthday');
    expect(input).toBeRequired();
  });

  it('associates input errors without discarding existing descriptions', () => {
    render(
      <>
        <Label htmlFor="email">Email</Label>
        <Input id="email" aria-describedby="email-hint" errorText="Enter a valid email" />
        <p id="email-hint">Use your work address.</p>
      </>
    );

    const input = screen.getByLabelText('Email');
    const error = screen.getByText('Enter a valid email');

    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(input).toHaveAttribute('aria-describedby', `email-hint ${error.id}`);
    expect(error).toHaveAttribute('aria-live', 'polite');
  });

  it('preserves caller-provided aria-invalid semantics without an error message', () => {
    render(<Input aria-label="Grammar sample" aria-invalid="grammar" />);

    expect(screen.getByLabelText('Grammar sample')).toHaveAttribute('aria-invalid', 'grammar');
  });

  it('applies the same error contract to textareas', () => {
    render(
      <>
        <Label htmlFor="summary">Summary</Label>
        <Textarea id="summary" errorText="Summary is required" />
      </>
    );

    const textarea = screen.getByLabelText('Summary');
    const error = screen.getByText('Summary is required');

    expect(textarea).toHaveAttribute('aria-invalid', 'true');
    expect(textarea).toHaveAttribute('aria-describedby', error.id);
    expect(error).toHaveAttribute('aria-live', 'polite');
  });

  it('requires and updates password visibility labels', () => {
    type PasswordInputProps = ComponentProps<typeof PasswordInput>;

    expectTypeOf<PasswordInputProps>().toMatchTypeOf<{
      showPasswordLabel: string;
      hidePasswordLabel: string;
    }>();

    const labels = {
      showPasswordLabel: 'Show password',
      hidePasswordLabel: 'Hide password',
    } satisfies Pick<PasswordInputProps, 'showPasswordLabel' | 'hidePasswordLabel'>;

    render(
      <>
        <Label htmlFor="password">Password</Label>
        <PasswordInput id="password" {...labels} />
      </>
    );

    const input = screen.getByLabelText('Password');
    const toggle = screen.getByRole('button', { name: 'Show password' });

    expect(input).toHaveAttribute('type', 'password');
    expect(toggle).toHaveAttribute('aria-controls', 'password');
    expect(toggle).not.toHaveAttribute('aria-pressed');

    fireEvent.click(toggle);

    expect(input).toHaveAttribute('type', 'text');
    expect(toggle).toHaveAccessibleName('Hide password');
    expect(toggle).toHaveAttribute('aria-controls', 'password');
  });

  it('keeps color input labeling and error state on the native control', () => {
    render(
      <>
        <Label htmlFor="brand-color">Brand color</Label>
        <ColorPicker
          id="brand-color"
          value="#112233"
          errorText="Choose an approved color"
          onChange={() => {}}
        />
      </>
    );

    const input = screen.getByLabelText('Brand color');
    const error = screen.getByText('Choose an approved color');

    expect(input).toHaveAttribute('type', 'color');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(input).toHaveAttribute('aria-describedby', error.id);
    expect(error).toHaveAttribute('aria-live', 'polite');
  });
});
