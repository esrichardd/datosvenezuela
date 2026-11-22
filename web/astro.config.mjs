// @ts-check
import { defineConfig, envField, fontProviders } from "astro/config";

// https://astro.build/config
export default defineConfig({
  build: {
    // Put the CSS inside each page instead of a separate file, so the browser
    // can paint without waiting for another request. Worth it while the CSS
    // stays small (a few KB compressed); revisit if it grows a lot.
    inlineStylesheets: "always",
  },
  // Typed environment variables, validated when the site is built.
  env: {
    schema: {
      // Base URL of the API, used by the API console in the browser.
      PUBLIC_API_URL: envField.string({
        context: "client",
        access: "public",
        url: true,
        default: "http://localhost:8080",
      }),
    },
  },
  // Fonts are downloaded at build time and served from this site
  // (no requests to Google from the visitor's browser).
  fonts: [
    {
      provider: fontProviders.google(),
      name: "Archivo",
      cssVariable: "--font-archivo",
      weights: ["400 900"],
      styles: ["normal"],
      subsets: ["latin"],
      fallbacks: ["sans-serif"],
      // Archivo is a variable font; keep its width axis for expanded headings.
      options: { experimental: { variableAxis: { wdth: [[62, 125]] } } },
    },
    {
      provider: fontProviders.google(),
      name: "IBM Plex Mono",
      cssVariable: "--font-ibm-plex-mono",
      weights: [400, 500, 600],
      styles: ["normal"],
      subsets: ["latin"],
      fallbacks: ["monospace"],
    },
  ],
});
