"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function Dashboard() {
  const router = useRouter();
  const [user, setUser] = useState<string | null>(null);

  useEffect(() => {
    const token = localStorage.getItem("token");
    const u = localStorage.getItem("user");
    if (!token) {
      router.replace("/");
      return;
    }
    setUser(u);
  }, [router]);

  function logout() {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    router.replace("/");
  }

  return (
    <main className="container">
      <div className="card">
        <h1>Welcome, {user ?? "..."} 👋</h1>
        <p>You are now authenticated.</p>
        <button className="primary" onClick={logout}>
          Logout
        </button>
      </div>
    </main>
  );
}
