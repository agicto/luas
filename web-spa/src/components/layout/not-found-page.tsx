import { Link } from '@tanstack/react-router';
import { ArrowLeft } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';

export function NotFoundPage() {
  const { t } = useTranslation();

  return (
    <main className="not-found">
      <span className="not-found-code">404</span>
      <h1>{t('errors.notFoundTitle')}</h1>
      <p>{t('errors.notFoundDescription')}</p>
      <Button asChild>
        <Link to="/console">
          <ArrowLeft className="size-4" />
          {t('errors.backToConsole')}
        </Link>
      </Button>
    </main>
  );
}
