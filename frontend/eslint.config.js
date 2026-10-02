// Configuração do ESLint (flat config). `npm run lint` roda com --max-warnings 0.
import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import jsxA11y from 'eslint-plugin-jsx-a11y';
import globals from 'globals';

export default tseslint.config(
    {
        // wailsjs/ é gerado pelo Wails.
        ignores: ['dist/', 'coverage/', 'wailsjs/', 'node_modules/'],
    },
    js.configs.recommended,
    {
        files: ['**/*.{ts,tsx}'],
        extends: [
            ...tseslint.configs.recommendedTypeChecked,
            react.configs.flat.recommended,
            react.configs.flat['jsx-runtime'],
            reactHooks.configs.flat.recommended,
            jsxA11y.flatConfigs.recommended,
        ],
        languageOptions: {
            parserOptions: {
                project: ['./tsconfig.json', './tsconfig.node.json'],
                tsconfigRootDir: import.meta.dirname,
            },
            globals: globals.browser,
        },
        settings: {
            react: {version: 'detect'},
        },
        rules: {
            // Closures velhas em efeitos já causaram bug real aqui: as duas
            // regras clássicas dos hooks são erro, não aviso.
            'react-hooks/rules-of-hooks': 'error',
            'react-hooks/exhaustive-deps': 'error',
            // Os tipos das props vêm do TypeScript.
            'react/prop-types': 'off',
        },
    },
    {
        // Arquivos de configuração e scripts rodam no Node, fora do projeto TS.
        files: ['**/*.{js,mjs,cjs}'],
        languageOptions: {
            globals: globals.node,
        },
    },
);
