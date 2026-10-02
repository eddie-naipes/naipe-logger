import {fileURLToPath} from 'node:url';
import {defineConfig} from 'vite';
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
});
