import { expect } from 'vitest';

import type { ServiceParityContext } from '../plugin-web-features';

/**
 * Runs every knowledge service method against the mock BFF, so a mock response that drifts from
 * the service's schema fails as an invalid response. Loaded by `mock-bff-service-parity.test.ts`.
 */
export async function runServiceParity({ organizationId }: ServiceParityContext): Promise<void> {
  const { knowledgeService } = await import('@/features/knowledge/services/knowledge-service');

  const created = await knowledgeService.create(organizationId, {
    title: 'Parity article',
    body: '# Parity\n\n知识库 parity body',
  });
  const second = await knowledgeService.create(organizationId, { title: 'Second article' });
  const firstPage = await knowledgeService.list(organizationId, { perPage: 1 });
  expect(firstPage.items.map(item => item.id)).toEqual([second.id]);
  expect(firstPage.meta.has_more).toBe(true);
  const nextPage = await knowledgeService.list(organizationId, {
    cursor: firstPage.meta.next_cursor,
    perPage: 1,
  });
  expect(nextPage.items.map(item => item.id)).toContain(created.id);

  const hits = await knowledgeService.search(organizationId, '知识库');
  expect(hits.map(hit => hit.id)).toEqual([created.id]);

  const read = await knowledgeService.get(organizationId, created.id);
  const updated = await knowledgeService.update(
    organizationId,
    created.id,
    { body: 'Updated body' },
    read.version
  );
  expect(updated.version).toBe(read.version + 1);

  await knowledgeService.remove(organizationId, created.id, updated.version);
  await knowledgeService.remove(organizationId, second.id, second.version);
}
