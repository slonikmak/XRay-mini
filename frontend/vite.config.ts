import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import wails from '@wailsio/runtime/plugins/vite';
export default defineConfig({ server: { host: "127.0.0.1", port: 9245, strictPort: true }, plugins: [react(), wails("./bindings")], base: './', clearScreen: false });
