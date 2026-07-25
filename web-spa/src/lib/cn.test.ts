import { cn } from '@/lib/cn';

describe('cn', () => {
  it('merges conditional and conflicting Tailwind classes', () => {
    const hidden = false;

    expect(cn('px-2', hidden ? 'hidden' : undefined, 'px-4')).toBe('px-4');
  });
});
