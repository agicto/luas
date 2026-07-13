import type { ComponentProps } from 'react';
import { render, screen } from '@testing-library/react';
import {
  NextIntlClientProvider,
  type AbstractIntlMessages,
} from 'next-intl';
import { describe, expect, expectTypeOf, it } from 'vitest';

import { FormShowcase } from '@/components/features/styleguide/form-showcase';
import { Alert, AlertTitle } from '@/components/ui/alert';
import { AvatarImage } from '@/components/ui/avatar';
import { messages } from '@/i18n/modules';

describe('shared UI accessibility contracts', () => {
  it('keeps alert titles out of the document heading hierarchy', () => {
    render(
      <Alert>
        <AlertTitle>Connection restored</AlertTitle>
      </Alert>
    );

    const title = screen.getByText('Connection restored');
    expect(title.tagName).toBe('DIV');
    expect(screen.queryByRole('heading')).not.toBeInTheDocument();
  });

  it('requires every avatar image caller to choose alt semantics', () => {
    type AvatarImageProps = ComponentProps<typeof AvatarImage>;

    expectTypeOf<AvatarImageProps['alt']>().toEqualTypeOf<string>();
  });

  it('associates every visible styleguide form label with its control', () => {
    render(
      <NextIntlClientProvider
        locale="en-US"
        messages={messages as unknown as AbstractIntlMessages}
      >
        <FormShowcase />
      </NextIntlClientProvider>
    );

    for (const label of [
      'Premium Date Picker (with Year)',
      'DateTime Picker (HMS)',
      'Color Picker',
      'File Upload',
      'Search Input (Pill)',
      'Password Input',
      'Outline Variant (Default)',
      'Filled Variant',
      'Textarea Outline',
      'Textarea Filled',
      'Checkbox Primary',
      'Switch Feedback',
      'Error State',
      'Textarea Error State',
    ]) {
      expect(screen.getByLabelText(label)).toBeInTheDocument();
    }

    expect(screen.queryByLabelText('Selection Toggles')).not.toBeInTheDocument();
  });
});
