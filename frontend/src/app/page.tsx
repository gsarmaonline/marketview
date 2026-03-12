"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useRouter } from "next/navigation";
import { NewsSection, type NewsItem } from "@/components/NewsSection";

type Signal = "bullish" | "neutral" | "bearish";

interface Indicator {
  name: string;
  value: number;
  unit: string;
  signal: Signal;
  description: string;
}

const SIGNAL_CONFIG: Record<Signal, { label: string; color: string; bg: string; border: string; icon: string }> = {
  bullish: {
    label: "Bullish",
    color: "#22c55e",
    bg: "rgba(34, 197, 94, 0.12)",
    border: "rgba(34, 197, 94, 0.3)",
    icon: "▲",
  },
  neutral: {
    label: "Neutral",
    color: "#f59e0b",
    bg: "rgba(245, 158, 11, 0.12)",
    border: "rgba(245, 158, 11, 0.3)",
    icon: "◆",
  },
  bearish: {
    label: "Bearish",
    color: "#ef4444",
    bg: "rgba(239, 68, 68, 0.12)",
    border: "rgba(239, 68, 68, 0.3)",
    icon: "▼",
  },
};

const REFRESH_INTERVAL = 60_000; // 1 minute

function IndicatorCard({ indicator }: { indicator: Indicator }) {
  const sig = SIGNAL_CONFIG[indicator.signal] ?? SIGNAL_CONFIG.neutral;

  return (
    <div
      style={{
        background: sig.bg,
        border: `1px solid ${sig.border}`,
        borderRadius: 12,
        padding: "24px",
        display: "flex",
        flexDirection: "column",
        gap: 12,
        transition: "transform 0.15s, box-shadow 0.15s",
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.transform = "translateY(-2px)";
        (e.currentTarget as HTMLDivElement).style.boxShadow = `0 8px 24px ${sig.border}`;
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.transform = "translateY(0)";
        (e.currentTarget as HTMLDivElement).style.boxShadow = "none";
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: 8 }}>
        <span style={{ fontSize: 14, fontWeight: 500, color: "#94a3b8", lineHeight: 1.4 }}>
          {indicator.name}
        </span>
        <span
          style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 4,
            background: sig.bg,
            border: `1px solid ${sig.border}`,
            color: sig.color,
            borderRadius: 20,
            padding: "3px 10px",
            fontSize: 12,
            fontWeight: 600,
            whiteSpace: "nowrap",
            flexShrink: 0,
          }}
        >
          <span style={{ fontSize: 9 }}>{sig.icon}</span>
          {sig.label}
        </span>
      </div>

      <div style={{ display: "flex", alignItems: "baseline", gap: 4 }}>
        <span
          style={{
            fontSize: 42,
            fontWeight: 700,
            color: sig.color,
            lineHeight: 1,
            fontVariantNumeric: "tabular-nums",
          }}
        >
          {indicator.value.toFixed(1)}
        </span>
        <span style={{ fontSize: 18, color: "#64748b", fontWeight: 500 }}>{indicator.unit}</span>
      </div>

      <p style={{ fontSize: 13, color: "#94a3b8", lineHeight: 1.5, borderTop: `1px solid ${sig.border}`, paddingTop: 12 }}>
        {indicator.description}
      </p>
    </div>
  );
}

function SignalSummary({ indicators }: { indicators: Indicator[] }) {
  const counts = indicators.reduce<Record<Signal, number>>(
    (acc, ind) => {
      acc[ind.signal] = (acc[ind.signal] ?? 0) + 1;
      return acc;
    },
    { bullish: 0, neutral: 0, bearish: 0 }
  );

  return (
    <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
      {(["bullish", "neutral", "bearish"] as Signal[]).map((sig) => {
        const cfg = SIGNAL_CONFIG[sig];
        return (
          <div
            key={sig}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              background: cfg.bg,
              border: `1px solid ${cfg.border}`,
              borderRadius: 8,
              padding: "8px 14px",
            }}
          >
            <span style={{ color: cfg.color, fontSize: 11 }}>{cfg.icon}</span>
            <span style={{ color: cfg.color, fontWeight: 700, fontSize: 20 }}>{counts[sig]}</span>
            <span style={{ color: "#64748b", fontSize: 13 }}>{cfg.label}</span>
          </div>
        );
      })}
    </div>
  );
}

function StockSearch() {
  const [query, setQuery] = useState("");
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const symbol = query.trim().toUpperCase();
    if (symbol) router.push(`/stock/${symbol}`);
  };

  return (
    <form onSubmit={handleSubmit} style={{ display: "flex", gap: 8, alignItems: "center" }}>
      <input
        ref={inputRef}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Enter NSE symbol (e.g. RELIANCE)"
        style={{
          background: "#1a1d27",
          border: "1px solid #2a2f45",
          borderRadius: 8,
          padding: "8px 14px",
          color: "#e2e8f0",
          fontSize: 13,
          outline: "none",
          width: 260,
        }}
      />
      <button
        type="submit"
        disabled={!query.trim()}
        style={{
          background: "rgba(99,102,241,0.15)",
          border: "1px solid rgba(99,102,241,0.3)",
          color: "#818cf8",
          borderRadius: 8,
          padding: "8px 16px",
          fontSize: 13,
          fontWeight: 500,
          cursor: query.trim() ? "pointer" : "not-allowed",
          opacity: query.trim() ? 1 : 0.5,
          whiteSpace: "nowrap",
        }}
      >
        Deep Research
      </button>
    </form>
  );
}

export default function DashboardPage() {
  const [indicators, setIndicators] = useState<Indicator[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [countdown, setCountdown] = useState(REFRESH_INTERVAL / 1000);

  const [news, setNews] = useState<NewsItem[]>([]);
  const [newsLoading, setNewsLoading] = useState(true);
  const [newsError, setNewsError] = useState<string | null>(null);

  const fetchIndicators = useCallback(async () => {
    try {
      const res = await fetch("/api/indicators");
      if (!res.ok) throw new Error(`Server error: ${res.status}`);
      const data: Indicator[] = await res.json();
      setIndicators(data);
      setLastUpdated(new Date());
      setError(null);
      setCountdown(REFRESH_INTERVAL / 1000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch indicators");
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchNews = useCallback(async () => {
    try {
      const res = await fetch("/api/news");
      if (!res.ok) throw new Error(`Server error: ${res.status}`);
      const data: NewsItem[] = await res.json();
      setNews(data);
      setNewsError(null);
    } catch (err) {
      setNewsError(err instanceof Error ? err.message : "Failed to fetch news");
    } finally {
      setNewsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchIndicators();
    const interval = setInterval(fetchIndicators, REFRESH_INTERVAL);
    return () => clearInterval(interval);
  }, [fetchIndicators]);

  useEffect(() => {
    fetchNews();
    const interval = setInterval(fetchNews, REFRESH_INTERVAL);
    return () => clearInterval(interval);
  }, [fetchNews]);

  // Countdown timer
  useEffect(() => {
    const tick = setInterval(() => setCountdown((c) => (c > 0 ? c - 1 : REFRESH_INTERVAL / 1000)), 1000);
    return () => clearInterval(tick);
  }, [lastUpdated]);

  return (
    <div style={{ maxWidth: 1200, margin: "0 auto", padding: "40px 24px" }}>
      {/* Header */}
      <div style={{ marginBottom: 40 }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", flexWrap: "wrap", gap: 16 }}>
          <div>
            <h1
              style={{
                fontSize: 28,
                fontWeight: 700,
                color: "#e2e8f0",
                letterSpacing: "-0.5px",
              }}
            >
              Market Indicators
            </h1>
            <p style={{ color: "#64748b", fontSize: 14, marginTop: 4 }}>
              Indian equity market health dashboard
            </p>
          </div>

          <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
            <StockSearch />
            {lastUpdated && (
              <span style={{ fontSize: 12, color: "#475569" }}>
                Updated {lastUpdated.toLocaleTimeString()} · refreshing in {countdown}s
              </span>
            )}
            <button
              onClick={fetchIndicators}
              disabled={loading}
              style={{
                background: "rgba(99,102,241,0.15)",
                border: "1px solid rgba(99,102,241,0.3)",
                color: "#818cf8",
                borderRadius: 8,
                padding: "8px 16px",
                fontSize: 13,
                fontWeight: 500,
                cursor: loading ? "not-allowed" : "pointer",
                opacity: loading ? 0.6 : 1,
                transition: "opacity 0.15s",
              }}
            >
              {loading ? "Loading…" : "Refresh"}
            </button>
          </div>
        </div>

        {indicators.length > 0 && (
          <div style={{ marginTop: 24 }}>
            <SignalSummary indicators={indicators} />
          </div>
        )}
      </div>

      {/* Error state */}
      {error && (
        <div
          style={{
            background: "rgba(239,68,68,0.1)",
            border: "1px solid rgba(239,68,68,0.3)",
            borderRadius: 10,
            padding: "16px 20px",
            color: "#fca5a5",
            fontSize: 14,
            marginBottom: 24,
          }}
        >
          ⚠ {error}
        </div>
      )}

      {/* Loading skeleton */}
      {loading && indicators.length === 0 && (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(300px, 1fr))",
            gap: 20,
          }}
        >
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              style={{
                background: "#1a1d27",
                border: "1px solid #2a2f45",
                borderRadius: 12,
                padding: 24,
                height: 180,
                animation: "pulse 1.5s ease-in-out infinite",
              }}
            />
          ))}
        </div>
      )}

      {/* Indicator grid */}
      {indicators.length > 0 && (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(300px, 1fr))",
            gap: 20,
          }}
        >
          {indicators.map((ind) => (
            <IndicatorCard key={ind.name} indicator={ind} />
          ))}
        </div>
      )}

      {!loading && !error && indicators.length === 0 && (
        <div
          style={{
            textAlign: "center",
            padding: "80px 24px",
            color: "#475569",
            fontSize: 14,
          }}
        >
          No indicators available.
        </div>
      )}

      {/* News section */}
      <div style={{ marginTop: 56 }}>
        <NewsSection news={news} loading={newsLoading} error={newsError} />
      </div>

      <style>{`
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.4; }
        }
      `}</style>
    </div>
  );
}
