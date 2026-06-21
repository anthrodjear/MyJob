/**
 * ThemeProvider — dark/light/system theme management.
 *
 * Manages theme state with localStorage persistence and system preference
 * detection. Applies theme classes to `<html>` element for Tailwind dark mode.
 *
 * @example
 *   <ThemeProvider>
 *     <QueryProvider>...</QueryProvider>
 *   </ThemeProvider>
 */

"use client";

import { createContext, useContext, useEffect, useLayoutEffect, useState } from "react";

type Theme = "light" | "dark" | "system";

const ThemeContext = createContext<{
  theme: Theme;
  setTheme: (t: Theme) => void;
}>({ theme: "system", setTheme: (t: Theme) => {} });

/**
 * Hook to access theme context.
 * Must be used within ThemeProvider.
 */
export function useTheme() {
  return useContext(ThemeContext);
}

/**
 * Apply theme to document.documentElement.
 * Called synchronously via useLayoutEffect to avoid FOUC.
 */
function applyTheme(root: HTMLElement, theme: Theme) {
  root.classList.remove("light", "dark");

  if (theme === "system") {
    const preferred = window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
    root.classList.add(preferred);
  } else {
    root.classList.add(theme);
  }
}

/**
 * ThemeProvider — dark/light/system theme management.
 *
 * Behavior:
 * - Reads initial theme from localStorage (defaults to "system")
 * - On theme change: updates localStorage and applies class to document.documentElement
 * - "system" mode follows OS prefers-color-scheme media query
 * - Listens for system preference changes when in "system" mode
 *
 * Accessibility:
 * - Respects `prefers-color-scheme` for system mode
 * - localStorage persistence across sessions
 * - No FOUC (uses useLayoutEffect for synchronous application)
 */
export function ThemeProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  // Lazy initialization — reads localStorage once on client, no SSR mismatch
  const [theme, setTheme] = useState<Theme>(() => {
    if (typeof window === "undefined") return "system";
    return (localStorage.getItem("theme") as Theme | null) ?? "system";
  });

  // Synchronous theme application (runs before paint) — prevents FOUC
  useLayoutEffect(() => {
    const root = document.documentElement;
    applyTheme(root, theme);
    localStorage.setItem("theme", theme);
  }, [theme]);

  // Listen for system preference changes when in "system" mode
  useEffect(() => {
    if (theme !== "system") return;

    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => {
      const root = document.documentElement;
      applyTheme(root, "system");
    };

    media.addEventListener("change", handler);
    return () => media.removeEventListener("change", handler);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}
