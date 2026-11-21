/// <reference types="vitest/config" />
// Unit tests for the TypeScript in src/lib. getViteConfig() applies the
// project's Astro settings, so tests resolve modules the same way as the site.
import { getViteConfig } from "astro/config";

export default getViteConfig({
  test: {
    // Tests live in a __tests__ folder next to the code they cover.
    include: ["src/**/__tests__/**/*.test.ts"],
    // The code under test only needs a few browser globals, which each test
    // replaces with small fakes; no simulated DOM is needed.
    environment: "node",
    restoreMocks: true,
    unstubGlobals: true,
  },
});
