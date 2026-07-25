// Test translations with deep nesting - English (US)
const messages = {
  title: 'i18n Multi-layer Nesting Test',
  level1: {
    title: 'Level 1',
    level2: {
      title: 'Level 2',
      message: 'This is a message from Level 2',
      level3: {
        title: 'Level 3',
        message: 'This is a message from Level 3',
        level4: {
          title: 'Level 4',
          message: 'This is a message from Level 4',
          deepValue: 'Deep Value: {value}',
        },
      },
    },
  },
} as const;

export default messages;
export type TestMessages = typeof messages;
