// Metadata translations - Simplified Chinese
import type { LocaleMessageShape } from '../../locale-message-shape';
import type { MetadataMessages } from './en-US';

const messages = {
  title: 'Luas AI 脚手架',
  description: '基于 Next.js、TypeScript 和 Tailwind CSS 构建的现代化 Web 应用脚手架',
} as const satisfies LocaleMessageShape<MetadataMessages>;

export default messages;
