export type ValidationResult = { ok: true } | { ok: false; error: string };

/**
 * Password rule (mirrors backend):
 *  - At least 8 characters
 *  - At least 1 number
 */
export function validatePassword(pw: string): ValidationResult {
  if (pw.length < 8) {
    return { ok: false, error: "Password must be at least 8 characters" };
  }
  if (!/\d/.test(pw)) {
    return { ok: false, error: "Password must contain at least 1 number" };
  }
  return { ok: true };
}

export function validateUsername(u: string): ValidationResult {
  if (u.trim().length === 0) {
    return { ok: false, error: "Username must not be empty" };
  }
  return { ok: true };
}
