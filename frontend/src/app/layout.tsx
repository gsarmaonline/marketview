import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "MarketView — Market Indicators",
  description: "Live Indian market indicators dashboard",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <nav className="nav">
          <a href="/" className="nav-brand">MarketView</a>
          <div className="nav-links">
            <a href="/">Indicators</a>
            <a href="/portfolio">Portfolio</a>
          </div>
        </nav>
        <main className="container">{children}</main>
      </body>
    </html>
  );
}
