import { render, screen } from '@testing-library/react';
import { Eye } from 'lucide-react';
import { describe, expect, it } from 'vitest';

import { Button } from './button';

describe('Button layout contract', () => {
  it('keeps icon and label content in one horizontal line', () => {
    render(
      <Button className="flex-col flex-wrap whitespace-normal">
        <Eye aria-hidden="true" />
        Open preview
      </Button>
    );

    expect(screen.getByRole('button', { name: 'Open preview' })).toHaveClass(
      'flex-row!',
      'flex-nowrap!',
      'whitespace-nowrap!'
    );
  });
});
