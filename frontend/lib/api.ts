const BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type AuthResponse = {
  message?: string;
  token?: string;
  user?: string;
  error?: string;
};

async function postAuth(path: string, body: unknown): Promise<AuthResponse> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return (await res.json()) as AuthResponse;
}

export const api = {
  register: (username: string, password: string) =>
    postAuth("/api/register", { username, password }),
  login: (username: string, password: string) =>
    postAuth("/api/login", { username, password }),
  me: async (token: string): Promise<AuthResponse> => {
    const res = await fetch(`${BASE}/api/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    return (await res.json()) as AuthResponse;
  },
};
