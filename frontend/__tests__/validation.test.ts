import { describe, expect, test } from "bun:test";
import { validatePassword, validateUsername } from "../lib/validation";

describe("validatePassword", () => {
  test("rejects passwords shorter than 8 chars", () => {
    const r = validatePassword("ab12");
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error).toMatch(/8 characters/);
  });

  test("rejects passwords without a number", () => {
    const r = validatePassword("abcdefgh");
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error).toMatch(/1 number/);
  });

  test("accepts valid password", () => {
    expect(validatePassword("abcdefg1").ok).toBe(true);
    expect(validatePassword("P@ssw0rd!").ok).toBe(true);
  });

  test("rejects empty password", () => {
    expect(validatePassword("").ok).toBe(false);
  });
});

describe("validateUsername", () => {
  test("rejects empty / whitespace", () => {
    expect(validateUsername("").ok).toBe(false);
    expect(validateUsername("   ").ok).toBe(false);
  });
  test("accepts non-empty", () => {
    expect(validateUsername("kong").ok).toBe(true);
  });
});
