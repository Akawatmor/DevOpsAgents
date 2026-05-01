import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "DevOpsAgents",
  description: "Sprint 1 - Authentication",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
