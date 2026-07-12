// Metadata translations - English (US)
import type { MetadataMessages } from './zh-Hans';
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  title: 'Luas',
  description: 'Modern web application scaffold built with Next.js, TypeScript, and Tailwind CSS',
} as const satisfies LocaleMessageShape<MetadataMessages>;

export default messages;
