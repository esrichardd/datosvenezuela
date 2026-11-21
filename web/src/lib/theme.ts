/**
 * Color theme: light or dark.
 *
 * The page starts with the theme the visitor chose (saved in localStorage)
 * or, if they never chose, the one of their operating system. The theme is
 * set as <html data-theme>, which the design tokens read.
 */

export type Theme = "light" | "dark";

export const THEME_STORAGE_KEY = "theme";

const darkQuery = "(prefers-color-scheme: dark)";

/** The theme saved by the visitor, if any. Storage can be unavailable. */
export function getStoredTheme(): Theme | null {
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY);
    return value === "light" || value === "dark" ? value : null;
  } catch {
    return null;
  }
}

export function getSystemTheme(): Theme {
  return window.matchMedia(darkQuery).matches ? "dark" : "light";
}

export function getCurrentTheme(): Theme {
  return document.documentElement.dataset.theme === "dark" ? "dark" : "light";
}

export function applyTheme(theme: Theme): void {
  document.documentElement.dataset.theme = theme;
}

export function saveTheme(theme: Theme): void {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme);
  } catch {
    // Private mode or storage disabled: the choice lasts until reload.
  }
}

/** Follows operating-system changes while the visitor has not chosen. */
export function onSystemThemeChange(callback: (theme: Theme) => void): void {
  window.matchMedia(darkQuery).addEventListener("change", (event) => {
    if (getStoredTheme() === null) callback(event.matches ? "dark" : "light");
  });
}
