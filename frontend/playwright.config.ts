import {defineConfig, devices} from '@playwright/test';

export default defineConfig({
    testDir: './e2e/integration',
    fullyParallel: false,
    workers: 3,
    forbidOnly: !!process.env.CI,
    retries: process.env.CI ? 2 : 0,
    timeout: 30000,
    expect: {
        timeout: 10000,
    },
    reporter: 'html',
    use: {
        baseURL: 'https://discom.spbgu.localhost:8443',
        ignoreHTTPSErrors: true,
        trace: 'on-first-retry',
        screenshot: 'only-on-failure',
    },

    projects: [
        {
            name: 'chromium',
            use: {...devices['Desktop Chrome']},
        },
    ],
});
