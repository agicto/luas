import type {
  CreateExampleRequest,
  ExampleItem,
  ExampleListResponse,
  ExampleQuerySchema,
  UpdateExampleRequest,
} from '@/features/example/types';

const now = () => new Date().toISOString();

let exampleItems: ExampleItem[] = [
  {
    id: 'example-1',
    title: 'Translation pipeline',
    description: 'Routes multilingual content through the mock workflow.',
    status: 'active',
    createdAt: now(),
    updatedAt: now(),
  },
  {
    id: 'example-2',
    title: 'Incident summarizer',
    description: 'Produces daily internal summaries for the ops team.',
    status: 'active',
    createdAt: now(),
    updatedAt: now(),
  },
  {
    id: 'example-3',
    title: 'Lead enrichment job',
    description: 'Augments new signups with CRM attributes.',
    status: 'inactive',
    createdAt: now(),
    updatedAt: now(),
  },
];

export function listExamples(query: ExampleQuerySchema): ExampleListResponse {
  const page = query.page ?? 1;
  const pageSize = query.pageSize ?? 10;
  const keyword = query.keyword?.trim().toLowerCase();

  const filteredItems = exampleItems.filter((item) => {
    if (query.status && item.status !== query.status) {
      return false;
    }

    if (!keyword) {
      return true;
    }

    return `${item.title} ${item.description ?? ''}`.toLowerCase().includes(keyword);
  });

  const start = (page - 1) * pageSize;

  return {
    items: filteredItems.slice(start, start + pageSize),
    total: filteredItems.length,
    page,
    pageSize,
  };
}

export function getExampleById(id: string): ExampleItem | undefined {
  return exampleItems.find((item) => item.id === id);
}

export function createExample(data: CreateExampleRequest): ExampleItem {
  const item: ExampleItem = {
    id: crypto.randomUUID(),
    title: data.title,
    description: data.description,
    status: data.status ?? 'active',
    createdAt: now(),
    updatedAt: now(),
  };

  exampleItems = [item, ...exampleItems];
  return item;
}

export function updateExample(id: string, data: UpdateExampleRequest): ExampleItem | undefined {
  const existingItem = getExampleById(id);

  if (!existingItem) {
    return undefined;
  }

  const updatedItem: ExampleItem = {
    ...existingItem,
    ...data,
    updatedAt: now(),
  };

  exampleItems = exampleItems.map((item) => (item.id === id ? updatedItem : item));
  return updatedItem;
}

export function deleteExample(id: string): boolean {
  const nextItems = exampleItems.filter((item) => item.id !== id);
  const didDelete = nextItems.length !== exampleItems.length;
  exampleItems = nextItems;
  return didDelete;
}
