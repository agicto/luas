import 'server-only';

import { env } from './env';
import { serverEnv } from './server-env';

export interface MockBffEnvironment {
  nodeEnv: typeof env.NODE_ENV;
  enabled: boolean;
}

function currentMockBffEnvironment(): MockBffEnvironment {
  return {
    nodeEnv: env.NODE_ENV,
    enabled: serverEnv.MOCK_BFF_ENABLED,
  };
}

export function isMockBffEnabled(
  environment: MockBffEnvironment = currentMockBffEnvironment()
): boolean {
  return environment.nodeEnv !== 'production' || environment.enabled;
}
