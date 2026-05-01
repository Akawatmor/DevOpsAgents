"use client";

import { useState, FormEvent } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { validatePassword, validateUsername } from "@/lib/validation";

type Status = { kind: "success" | "error"; text: string } | null;

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [status, setStatus] = useState<Status>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(mode: "login" | "register", e: FormEvent) {
    e.preventDefault();
    setStatus(null);

    const u = validateUsername(username);
    if (!u.ok) return setStatus({ kind: "error", text: u.error });

    if (mode === "register") {
      const p = validatePassword(password);
      if (!p.ok) return setStatus({ kind: "error", text: p.error });
    }

    setLoading(true);
    try {
      const res =
        mode === "register"
          ? await api.register(username, password)
          : await api.login(username, password);

      if (res.error || !res.token) {
        setStatus({ kind: "error", text: res.error ?? "Unknown error" });
        return;
      }

      localStorage.setItem("token", res.token);
      localStorage.setItem("user", res.user ?? username);
      setStatus({
        kind: "success",
        text: mode === "register" ? "Registered successfully!" : "Logged in!",
      });
      setTimeout(() => router.push("/dashboard"), 600);
    } catch (err) {
      setStatus({ kind: "error", text: "Network error. Is backend running?" });
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="container">
      <form className="card" onSubmit={(e) => e.preventDefault()}>
        <h1>DevOpsAgents - Login</h1>

        <div className="field">
          <label htmlFor="username">Username</label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
          />
        </div>

        <div className="field">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
          <small style={{ color: "#94a3b8" }}>
            Min 8 characters, at least 1 number
          </small>
        </div>

        <div className="actions">
          <button
            className="secondary"
            disabled={loading}
            onClick={(e) => handleSubmit("register", e)}
          >
            Register
          </button>
          <button
            className="primary"
            type="submit"
            disabled={loading}
            onClick={(e) => handleSubmit("login", e)}
          >
            Login
          </button>
        </div>

        {status && (
          <div className={`message ${status.kind}`} role="alert">
            {status.text}
          </div>
        )}
      </form>
    </main>
  );
}
