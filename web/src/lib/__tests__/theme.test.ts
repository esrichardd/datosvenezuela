import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  applyTheme,
  getCurrentTheme,
  getStoredTheme,
  getSystemTheme,
  onSystemThemeChange,
  saveTheme,
  THEME_STORAGE_KEY,
} from "../theme";

/** A localStorage replacement backed by a Map. */
function fakeStorage(initial: Record<string, string> = {}) {
  const data = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => void data.set(key, value),
    data,
  };
}

/** A localStorage that fails, as in some private browsing modes. */
const brokenStorage = {
  getItem: () => {
    throw new Error("storage disabled");
  },
  setItem: () => {
    throw new Error("storage disabled");
  },
};

/** A matchMedia replacement whose result and change events tests control. */
function fakeMatchMedia(dark: boolean) {
  const listeners: ((event: { matches: boolean }) => void)[] = [];
  const matchMedia = () => ({
    matches: dark,
    addEventListener: (_: string, listener: (typeof listeners)[number]) =>
      listeners.push(listener),
  });
  const change = (matches: boolean) =>
    listeners.forEach((listener) => listener({ matches }));
  return { matchMedia, change };
}

beforeEach(() => {
  vi.stubGlobal("document", { documentElement: { dataset: {} } });
  vi.stubGlobal("window", { matchMedia: fakeMatchMedia(false).matchMedia });
  vi.stubGlobal("localStorage", fakeStorage());
});

describe("getStoredTheme", () => {
  it.each([
    { stored: "light", want: "light" },
    { stored: "dark", want: "dark" },
    { stored: "blue", want: null },
    { stored: undefined, want: null },
  ])("returns $want when storage holds $stored", ({ stored, want }) => {
    const initial: Record<string, string> = stored
      ? { [THEME_STORAGE_KEY]: stored }
      : {};
    vi.stubGlobal("localStorage", fakeStorage(initial));
    expect(getStoredTheme()).toBe(want);
  });

  it("returns null when storage is unavailable", () => {
    vi.stubGlobal("localStorage", brokenStorage);
    expect(getStoredTheme()).toBeNull();
  });
});

describe("saveTheme", () => {
  it("stores the theme", () => {
    const storage = fakeStorage();
    vi.stubGlobal("localStorage", storage);
    saveTheme("dark");
    expect(storage.data.get(THEME_STORAGE_KEY)).toBe("dark");
  });

  it("does not throw when storage is unavailable", () => {
    vi.stubGlobal("localStorage", brokenStorage);
    expect(() => saveTheme("dark")).not.toThrow();
  });
});

describe("getSystemTheme", () => {
  it.each([
    { dark: true, want: "dark" },
    { dark: false, want: "light" },
  ])("returns $want when the system prefers dark: $dark", ({ dark, want }) => {
    vi.stubGlobal("window", { matchMedia: fakeMatchMedia(dark).matchMedia });
    expect(getSystemTheme()).toBe(want);
  });
});

describe("applyTheme and getCurrentTheme", () => {
  it("sets the theme on the document and reads it back", () => {
    applyTheme("dark");
    expect(getCurrentTheme()).toBe("dark");
    applyTheme("light");
    expect(getCurrentTheme()).toBe("light");
  });

  it("treats a document without a theme as light", () => {
    expect(getCurrentTheme()).toBe("light");
  });
});

describe("onSystemThemeChange", () => {
  it("follows the system while the visitor has not chosen", () => {
    const media = fakeMatchMedia(false);
    vi.stubGlobal("window", { matchMedia: media.matchMedia });
    const callback = vi.fn();

    onSystemThemeChange(callback);
    media.change(true);

    expect(callback).toHaveBeenCalledWith("dark");
  });

  it("ignores the system once the visitor has chosen", () => {
    const media = fakeMatchMedia(false);
    vi.stubGlobal("window", { matchMedia: media.matchMedia });
    vi.stubGlobal(
      "localStorage",
      fakeStorage({ [THEME_STORAGE_KEY]: "light" }),
    );
    const callback = vi.fn();

    onSystemThemeChange(callback);
    media.change(true);

    expect(callback).not.toHaveBeenCalled();
  });
});
