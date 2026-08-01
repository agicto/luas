// Metadata translations - Simplified Chinese
import type { LocaleMessageShape } from '../../locale-message-shape';
import type { MetadataMessages } from './en-US';

const messages = {
  title: 'Luas Web',
  description: '基于 Next.js、TypeScript 和 Tailwind CSS 构建的用户端 Web 应用',
} as const satisfies LocaleMessageShape<MetadataMessages>;

export default messages;
