import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { initI18n } from "./i18n";
import "./styles/globals.css";

// Mark fonts as loaded once Material Symbols are available
if (document.fonts && document.fonts.ready) {
  document.fonts.ready.then(() => {
    document.documentElement.classList.add("fonts-loaded");
  });
} else {
  // Fallback: mark loaded after a short delay
  setTimeout(() => document.documentElement.classList.add("fonts-loaded"), 500);
}

// Boot i18n, then render
initI18n().then(() => {
  createRoot(document.getElementById("root")).render(
    <StrictMode>
      <App />
    </StrictMode>
  );
});
