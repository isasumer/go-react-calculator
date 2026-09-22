import { setupServer } from "msw/node";

import { handlers } from "@/test/msw/handlers";

/** Shared MSW server for unit tests. Per-test overrides: `server.use(http.post(...))`. */
export const server = setupServer(...handlers);
