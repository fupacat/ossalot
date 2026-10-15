import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

// The Go server (cmd/ossalot) answers /api on :8080; Vite serves the UI with hot reload on :5173.
export default defineConfig({
  plugins: [tailwindcss()],
  server: {
    host: "127.0.0.1",
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": "http://127.0.0.1:8080",
    },
  },
  build: {
    outDir: "dist",
    // Keep the committed placeholder that lets go:embed compile without a build (ADR-029)
    emptyOutDir: false,
  },
});
