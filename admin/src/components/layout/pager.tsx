import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import type { PageMeta } from '@/http/pagination';

interface PagerProps {
  meta: PageMeta | undefined;
  label: string;
  onPageChange: (page: number) => void;
}

/** Previous/next navigation for a paginated list; renders nothing for a single page. */
export function Pager({ meta, label, onPageChange }: PagerProps) {
  const { t } = useTranslation();
  if (!meta || meta.last_page <= 1) {
    return null;
  }
  return (
    <nav
      className="flex items-center justify-between border-t px-4 py-3 text-sm"
      aria-label={label}
    >
      <span className="text-muted-foreground">
        {t('common.page', { page: meta.current_page, pages: meta.last_page })}
      </span>
      <div className="flex gap-2">
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={meta.current_page <= 1}
          onClick={() => onPageChange(meta.current_page - 1)}
        >
          {t('common.previous')}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={meta.current_page >= meta.last_page}
          onClick={() => onPageChange(meta.current_page + 1)}
        >
          {t('common.next')}
        </Button>
      </div>
    </nav>
  );
}
