// @ts-check
import js from "@eslint/js";
import { defineConfig, globalIgnores } from "eslint/config";
import astro from "eslint-plugin-astro";
import tseslint from "typescript-eslint";

export default defineConfig([
  // ESLint already skips node_modules; build output and generated types too.
  globalIgnores(["dist/", ".astro/"]),
  js.configs.recommended,
  tseslint.configs.recommended,
  astro.configs.recommended,
  // Accessibility rules for Astro templates (backed by eslint-plugin-jsx-a11y-x).
  astro.configs["jsx-a11y-recommended"],
]);
