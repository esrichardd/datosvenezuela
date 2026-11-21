/**
 * Splits JSON text into tokens so it can be shown with syntax colors.
 *
 * It only classifies the text: the caller renders each token as plain text
 * (never as HTML), so values such as "<script>" stay harmless.
 */

export type TokenKind = "key" | "string" | "number" | "literal" | "punctuation";

export interface Token {
  kind: TokenKind;
  text: string;
}

// A string (and the colon after it, if it is a key), a literal, or a number.
const tokenPattern =
  /("(?:[^"\\]|\\.)*")(\s*:)?|\b(?:true|false|null)\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?/g;

/** Tokenizes JSON text, one array of tokens per line. */
export function highlightJson(json: string): Token[][] {
  return json.split("\n").map((line) => {
    const tokens: Token[] = [];
    const push = (kind: TokenKind, text: string) => {
      if (text) tokens.push({ kind, text });
    };

    let last = 0;
    for (const match of line.matchAll(tokenPattern)) {
      const [whole, string, colon] = match;
      push("punctuation", line.slice(last, match.index));
      if (string !== undefined) {
        push(colon ? "key" : "string", string);
        push("punctuation", colon ?? "");
      } else {
        push(/^-?\d/.test(whole) ? "number" : "literal", whole);
      }
      last = match.index + whole.length;
    }
    push("punctuation", line.slice(last));
    return tokens;
  });
}
