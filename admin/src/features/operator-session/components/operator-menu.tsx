import { useNavigate } from '@tanstack/react-router';
import { LogOut } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { isFeatureEnabled } from '@/config/env';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  useOperatorSession,
  useSignOut,
} from '@/features/operator-session/hooks/use-operator-session';

export function OperatorMenu() {
  return isFeatureEnabled('operator') ? <SignedInOperator /> : null;
}

function SignedInOperator() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const session = useOperatorSession();
  const signOut = useSignOut();
  const operator = session.data?.operator;
  if (!operator) {
    return null;
  }
  const label = t('auth.signOut');

  return (
    <div className="flex items-center gap-2">
      <span className="hidden max-w-40 truncate text-sm text-muted-foreground md:inline">
        {operator.nickname || operator.username}
      </span>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="size-8"
            aria-label={label}
            disabled={signOut.isPending}
            onClick={() =>
              signOut.mutate(undefined, {
                onSettled: () => void navigate({ to: '/login', replace: true }),
              })
            }
          >
            <LogOut aria-hidden="true" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom">{label}</TooltipContent>
      </Tooltip>
    </div>
  );
}
