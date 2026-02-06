import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  timeout: 30000,
  retries: 0,
  use: {
    baseURL: "http://localhost:5174",
    trace: "on-first-retry",
  },
  webServer: [
    {
      command: "sh -c 'PORT=9091 go run server/cmd/main.go'",
      stdout: "pipe",
      reuseExistingServer: !process.env.CI,
      timeout: 30000,
      cwd: "..",
    },
    {
      command: "sh -c 'VITE_SERVER_URL=ws://localhost:9091/ws npm run dev -- --port 5174'",
      url: "http://localhost:5174",
      reuseExistingServer: !process.env.CI,
      timeout: 30000,
      cwd: "../client",
    },
  ],
});
