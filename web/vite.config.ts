import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// The build lands in ../static, which the Go binary embeds: one artifact goes
// to the Pi. In dev the two servers run side by side and /api is proxied, so
// the frontend talks to the same relative URLs it will use in production.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "../static",
    // Wipes ../static on every build, including the .gitkeep that //go:embed
    // needs to compile against a clean checkout. public/.gitkeep is copied
    // back in as part of the same build, so any way of invoking Vite leaves
    // the Go build working.
    emptyOutDir: true,
  },
  server: {
    proxy: { "/api": "http://localhost:8080" },
  },
})
