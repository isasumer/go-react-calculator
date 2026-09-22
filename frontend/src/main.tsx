import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { App } from "@/app/App";
import "@/app/globals.css";

const root = document.getElementById("root");
if (!root) {
  throw new Error('Missing <div id="root"> in index.html');
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
