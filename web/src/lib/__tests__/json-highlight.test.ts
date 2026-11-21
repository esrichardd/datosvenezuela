import { describe, expect, it } from "vitest";
import { highlightJson, type Token } from "../json-highlight";

/** Shorthand for the expected tokens. */
const t = (kind: Token["kind"], text: string): Token => ({ kind, text });

describe("highlightJson", () => {
  it.each([
    {
      name: "key and string",
      json: '  "name": "Mérida",',
      want: [
        t("punctuation", "  "),
        t("key", '"name"'),
        t("punctuation", ":"),
        t("punctuation", " "),
        t("string", '"Mérida"'),
        t("punctuation", ","),
      ],
    },
    {
      name: "numbers",
      json: '"total": -1.5e3',
      want: [
        t("key", '"total"'),
        t("punctuation", ":"),
        t("punctuation", " "),
        t("number", "-1.5e3"),
      ],
    },
    {
      name: "literals",
      json: "[true, false, null]",
      want: [
        t("punctuation", "["),
        t("literal", "true"),
        t("punctuation", ", "),
        t("literal", "false"),
        t("punctuation", ", "),
        t("literal", "null"),
        t("punctuation", "]"),
      ],
    },
    {
      name: "escaped quotes stay inside the string",
      json: '"Banco \\"X\\""',
      want: [t("string", '"Banco \\"X\\""')],
    },
    {
      name: "digits inside a string are not a number",
      json: '"VE14"',
      want: [t("string", '"VE14"')],
    },
    {
      name: "markup is kept as text",
      json: '"<script>alert(1)</script>"',
      want: [t("string", '"<script>alert(1)</script>"')],
    },
  ])("tokenizes $name", ({ json, want }) => {
    expect(highlightJson(json)).toEqual([want]);
  });

  it("returns one array of tokens per line", () => {
    const lines = highlightJson('{\n  "a": 1\n}');
    expect(lines).toHaveLength(3);
    expect(lines[0]).toEqual([t("punctuation", "{")]);
    expect(lines[2]).toEqual([t("punctuation", "}")]);
  });

  it("returns an empty line for empty text", () => {
    expect(highlightJson("")).toEqual([[]]);
  });
});
