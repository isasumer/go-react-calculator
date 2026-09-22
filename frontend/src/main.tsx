import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { App } from "@/app/App";
import "@/app/globals.css";
import { ErrorBoundary } from "@/components/providers/ErrorBoundary";
import { QueryProvider } from "@/components/providers/QueryProvider";

const root = document.getElementById("root");
if (!root) {
  throw new Error('Missing <div id="root"> in index.html');
}

// QueryProvider is outermost: the boundary's fallback must keep working after the tree below it
// has failed, and the query client is what a reload-free recovery would need first.
createRoot(root).render(
  <StrictMode>
    <QueryProvider>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </QueryProvider>
  </StrictMode>,
);
