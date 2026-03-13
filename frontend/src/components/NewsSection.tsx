"use client";

import { useState, useCallback } from "react";

export interface NewsItem {
  title: string;
  description: string;
  link: string;
  publishedAt: string;
  source: string;
  symbol?: string;
}

const sourceColors: Record<string, string> = {
  "Economic Times": "#ff6b35",
  "Moneycontrol": "#0077cc",
  "Business Standard": "#c0392b",
};

function timeAgo(iso: string): string {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

function NewsCard({ item }: { item: NewsItem }) {
  const color = sourceColors[item.source] ?? "var(--accent)";

  return (
    <a
      href={item.link}
      target="_blank"
      rel="noopener noreferrer"
      style={{
        display: "block",
        background: "var(--surface)",
        border: "1px solid var(--border)",
        borderRadius: "var(--radius)",
        padding: "16px 20px",
        textDecoration: "none",
        color: "inherit",
        transition: "background 0.15s",
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLElement).style.background = "var(--surface-hover)";
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLElement).style.background = "var(--surface)";
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: "8px" }}>
        <span style={{
          background: `${color}22`,
          color: color,
          border: `1px solid ${color}44`,
          borderRadius: "5px",
          padding: "1px 8px",
          fontSize: "11px",
          fontWeight: 600,
          letterSpacing: "0.04em",
          textTransform: "uppercase",
        }}>
          {item.source}
        </span>
        {item.symbol && (
          <span style={{
            background: "rgba(99,102,241,0.15)",
            color: "#818cf8",
            border: "1px solid rgba(99,102,241,0.3)",
            borderRadius: "5px",
            padding: "1px 8px",
            fontSize: "11px",
            fontWeight: 600,
            letterSpacing: "0.04em",
          }}>
            {item.symbol}
          </span>
        )}
        {item.publishedAt && (
          <span style={{ color: "var(--text-muted)", fontSize: "12px" }}>
            {timeAgo(item.publishedAt)}
          </span>
        )}
      </div>

      <h3 style={{
        fontSize: "14px",
        fontWeight: 600,
        lineHeight: 1.5,
        color: "var(--text)",
        marginBottom: item.description ? "6px" : 0,
      }}>
        {item.title}
      </h3>

      {item.description && (
        <p style={{
          fontSize: "13px",
          color: "var(--text-muted)",
          lineHeight: 1.55,
          display: "-webkit-box",
          WebkitLineClamp: 2,
          WebkitBoxOrient: "vertical",
          overflow: "hidden",
        }}>
          {item.description}
        </p>
      )}
    </a>
  );
}

function NewsCardList({
  news,
  loading,
  error,
  emptyMessage,
}: {
  news: NewsItem[];
  loading: boolean;
  error: string | null;
  emptyMessage: string;
}) {
  if (loading) {
    return <div style={{ color: "var(--text-muted)", padding: "24px 0" }}>Loading news…</div>;
  }
  if (error) {
    return (
      <div style={{
        background: "var(--bearish-bg)",
        border: "1px solid var(--bearish)",
        borderRadius: "var(--radius)",
        padding: "12px 16px",
        color: "var(--bearish)",
        fontSize: "13px",
      }}>
        {error}
      </div>
    );
  }
  if (news.length === 0) {
    return <div style={{ color: "var(--text-muted)", padding: "24px 0" }}>{emptyMessage}</div>;
  }
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "10px" }}>
      {news.map((item, i) => (
        <NewsCard key={`${item.link}-${i}`} item={item} />
      ))}
    </div>
  );
}

function StockNewsPanel() {
  const [symbol, setSymbol] = useState("");
  const [submittedSymbol, setSubmittedSymbol] = useState("");
  const [stockNews, setStockNews] = useState<NewsItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStockNews = useCallback(async (sym: string) => {
    const clean = sym.trim().toUpperCase();
    if (!clean) return;
    setLoading(true);
    setError(null);
    setSubmittedSymbol(clean);
    try {
      const res = await fetch(`/api/news/stock/${encodeURIComponent(clean)}`);
      if (!res.ok) throw new Error(`Server error: ${res.status}`);
      const data: NewsItem[] = await res.json();
      setStockNews(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch stock news");
    } finally {
      setLoading(false);
    }
  }, []);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    fetchStockNews(symbol);
  };

  return (
    <div>
      <form onSubmit={handleSubmit} style={{ display: "flex", gap: "10px", marginBottom: "20px" }}>
        <input
          type="text"
          value={symbol}
          onChange={(e) => setSymbol(e.target.value.toUpperCase())}
          placeholder="Stock symbol, e.g. HDFCBANK"
          style={{
            flex: 1,
            background: "var(--surface)",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            padding: "10px 14px",
            fontSize: "14px",
            color: "var(--text)",
            outline: "none",
          }}
        />
        <button
          type="submit"
          disabled={loading || !symbol.trim()}
          style={{
            background: "rgba(99,102,241,0.15)",
            border: "1px solid rgba(99,102,241,0.3)",
            color: "#818cf8",
            borderRadius: "var(--radius)",
            padding: "10px 20px",
            fontSize: "13px",
            fontWeight: 500,
            cursor: loading || !symbol.trim() ? "not-allowed" : "pointer",
            opacity: loading || !symbol.trim() ? 0.6 : 1,
            whiteSpace: "nowrap",
          }}
        >
          {loading ? "Loading…" : "Fetch News"}
        </button>
      </form>

      {!submittedSymbol && !loading && (
        <div style={{ color: "var(--text-muted)", padding: "24px 0", fontSize: "14px" }}>
          Enter a stock symbol above to see its news.
        </div>
      )}

      {submittedSymbol && (
        <NewsCardList
          news={stockNews}
          loading={loading}
          error={error}
          emptyMessage={`No news stored for ${submittedSymbol} yet. Ingest news via the pipeline to populate this view.`}
        />
      )}
    </div>
  );
}

type Tab = "market" | "stock";

export function NewsSection({
  news,
  loading,
  error,
}: {
  news: NewsItem[];
  loading: boolean;
  error: string | null;
}) {
  const [activeTab, setActiveTab] = useState<Tab>("market");

  const tabStyle = (tab: Tab) => ({
    padding: "8px 18px",
    fontSize: "14px",
    fontWeight: 600,
    borderRadius: "var(--radius)",
    cursor: "pointer",
    border: activeTab === tab ? "1px solid rgba(99,102,241,0.4)" : "1px solid transparent",
    background: activeTab === tab ? "rgba(99,102,241,0.15)" : "transparent",
    color: activeTab === tab ? "#818cf8" : "var(--text-muted)",
    transition: "all 0.15s",
  });

  return (
    <section>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "20px", flexWrap: "wrap", gap: "12px" }}>
        <h2 style={{ fontSize: "18px", fontWeight: 700, color: "var(--text)" }}>
          News
        </h2>
        <div style={{ display: "flex", gap: "6px" }}>
          <button style={tabStyle("market")} onClick={() => setActiveTab("market")}>
            Market News
          </button>
          <button style={tabStyle("stock")} onClick={() => setActiveTab("stock")}>
            Stock News
          </button>
        </div>
      </div>

      {activeTab === "market" && (
        <NewsCardList
          news={news}
          loading={loading}
          error={error}
          emptyMessage="No news available."
        />
      )}

      {activeTab === "stock" && <StockNewsPanel />}
    </section>
  );
}
