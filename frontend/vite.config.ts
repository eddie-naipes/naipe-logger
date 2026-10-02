import {fileURLToPath} from 'node:url';
import {defineConfig} from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
    plugins: [react()],
    resolve: {
        alias: {
            // Bindings gerados pelo Wails. O tsc usa a cópia de tipos em
            // .wails-types/ (ver tsconfig.json e scripts/wails-types.mjs).
            '@wailsjs': fileURLToPath(new URL('./wailsjs', import.meta.url)),
        },
    },
    test: {
        environment: 'jsdom',
        setupFiles: ['./src/test/setup.ts'],
        // Fuso fixo do Brasil (UTC-3): os bugs de data que os testes cobrem só
        // aparecem fora do UTC.
        env: {TZ: 'America/Sao_Paulo'},
        restoreMocks: true,
        coverage: {
            provider: 'v8',
            include: ['src/**/*.{ts,tsx}'],
            exclude: ['src/main.tsx', 'src/test/**', 'src/**/*.test.{ts,tsx}', 'src/types/**'],
            reporter: ['text', 'html'],
        },
    },
});
