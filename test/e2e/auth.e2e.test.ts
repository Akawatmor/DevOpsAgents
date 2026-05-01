/**
 * E2E test against a running backend at http://localhost:8080
 * Run:
 *   1) cd backend && go run .
 *   2) cd test && bun test
 */
import { describe, expect, test } from "bun:test";

const BASE = process.env.API_URL ?? "http://localhost:8080";

function uniqueUser() {
  return `user_${Date.now()}_${Math.floor(Math.random() * 1000)}`;
}

describe("auth e2e", () => {
  test("health endpoint", async () => {
    const res = await fetch(`${BASE}/api/health`);
    expect(res.status).toBe(200);
  });

  test("register -> login -> me", async () => {
    const username = uniqueUser();
    const password = "passw0rd";

    const reg = await fetch(`${BASE}/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    expect(reg.status).toBe(201);
    const regBody = await reg.json();
    expect(regBody.token).toBeTruthy();

    const login = await fetch(`${BASE}/api/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    expect(login.status).toBe(200);
    const loginBody = await login.json();
    expect(loginBody.token).toBeTruthy();

    const me = await fetch(`${BASE}/api/me`, {
      headers: { Authorization: `Bearer ${loginBody.token}` },
    });
    expect(me.status).toBe(200);
    const meBody = await me.json();
    expect(meBody.user).toBe(username);
  });

  test("rejects weak password", async () => {
    const res = await fetch(`${BASE}/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: uniqueUser(), password: "short" }),
    });
    expect(res.status).toBe(400);
  });

  test("rejects wrong password", async () => {
    const username = uniqueUser();
    await fetch(`${BASE}/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password: "passw0rd" }),
    });
    const res = await fetch(`${BASE}/api/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password: "wrongpw1" }),
    });
    expect(res.status).toBe(401);
  });
});
