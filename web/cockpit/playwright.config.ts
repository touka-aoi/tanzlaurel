import { defineConfig } from "@playwright/test";

const TEST_BACKEND_PORT = 8081;
const TEST_FRONTEND_PORT = 5174;

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  use: {
    baseURL: `http://localhost:${TEST_FRONTEND_PORT}`,
  },
  webServer: [
    {
      command: `cd .. && ADMIN_USER=admin ADMIN_PASSWORD=pass ADDRESS=:${TEST_BACKEND_PORT} DATA_DIR=/tmp/e2e-test-data go run ./server/cmd/`,
      port: TEST_BACKEND_PORT,
      reuseExistingServer: false,
    },
    {
      command: `npx vite --port ${TEST_FRONTEND_PORT}`,
      port: TEST_FRONTEND_PORT,
      reuseExistingServer: false,
      env: {
        VITE_API_TARGET: `http://localhost:${TEST_BACKEND_PORT}`,
        VITE_WS_TARGET: `ws://localhost:${TEST_BACKEND_PORT}`,
      },
    },
  ],
});
