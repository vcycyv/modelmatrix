import '@testing-library/jest-dom';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { server } from './mocks/server';

// Start MSW server before all tests
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }));

// Reset handlers between tests so one test's overrides don't bleed into another
afterEach(() => server.resetHandlers());

// Tear down the server after all tests
afterAll(() => server.close());
