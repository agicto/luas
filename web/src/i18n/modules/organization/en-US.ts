import type { LocaleMessageShape } from '../../locale-message-shape';
import type { OrganizationMessages } from './zh-Hans';

const messages = {
  title: 'Organizations',
  description: 'Manage the organizations you belong to and their core settings.',
  list: 'Organization list',
  create: 'Create organization',
  createDescription: 'Create a new organization boundary. The name can change; the slug remains fixed.',
  createSuccess: 'Organization created',
  updateSuccess: 'Organization settings updated',
  name: 'Organization name',
  namePlaceholder: 'For example, Acme Europe',
  nameInvalid: 'Enter an organization name between 2 and 100 characters',
  slug: 'Organization slug',
  slugPlaceholder: 'Optional, for example acme-europe',
  slugHint: 'Use lowercase letters, numbers, and hyphens, or leave blank to generate one.',
  slugInvalid: 'Enter a canonical organization slug between 3 and 50 characters',
  role: 'Role',
  open: 'Open',
  retry: 'Retry',
  emptyTitle: 'No organizations yet',
  emptyDescription: 'Create the first organization to start managing members and organization-owned resources.',
  back: 'Back to organizations',
  contextVerified: 'Context verified',
  profile: 'Organization profile',
  profileDescription: 'The slug is immutable. Owners and administrators can update the display name.',
  roles: {
    owner: 'Owner',
    admin: 'Administrator',
    member: 'Member',
  },
  errors: {
    unavailable: 'The organization service is temporarily unavailable. Try again shortly.',
    forbidden: 'You do not have permission to perform this operation.',
    notFound: 'The organization does not exist, or you are no longer a member.',
    slugConflict: 'That organization slug is already in use.',
    invalidResponse: 'The organization service returned data this client cannot recognize.',
    generic: 'The organization operation failed. Try again.',
  },
} as const satisfies LocaleMessageShape<OrganizationMessages>;

export default messages;
