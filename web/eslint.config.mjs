import nextCoreWebVitals from 'eslint-config-next/core-web-vitals';
import nextTypescript from 'eslint-config-next/typescript';

const eslintConfig = [
  {
    ignores: ['.next/**', 'coverage/**', '.registry-check/**'],
  },
  ...nextCoreWebVitals,
  ...nextTypescript,
  {
    rules: {
      'react-hooks/set-state-in-effect': 'off',
    },
  },
];

export default eslintConfig;
