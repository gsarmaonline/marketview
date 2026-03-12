"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";

interface AnnualReport {
  seqNumber: number;
  issuer: string;
  year: string;
  subject: string;
  pdfLink: string;
}

interface DeepResearch {
  symbol: string;
  annualReports: AnnualReport[];
  annualReportsSource: string;
}

export default function StockDeepResearchPage() {
  const params = useParams();
  const symbol = (params.symbol as string).toUpperCase();

  const [data, setData] = useState<DeepResearch | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchDeepResearch() {
      try {
        const res = await fetch(`/api/stock/${symbol}/deep-research`);
        if (!res.ok) throw new Error(`Server error: ${res.status}`);
        const json: DeepResearch = await res.json();
        setData(json);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to fetch data");
      } finally {
        setLoading(false);
      }
    }
    fetchDeepResearch();
  }, [symbol]);

  return (
    <div style={{ maxWidth: 900, margin: "0 auto", padding: "40px 24px" }}>
      {/* Back link */}
      <Link
        href="/"
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: 6,
          color: "#64748b",
          fontSize: 13,
          textDecoration: "none",
          marginBottom: 32,
        }}
      >
        ← Back to dashboard
      </Link>

      {/* Header */}
      <div style={{ marginBottom: 36 }}>
        <h1 style={{ fontSize: 28, fontWeight: 700, color: "#e2e8f0", letterSpacing: "-0.5px" }}>
          {symbol}
        </h1>
        <p style={{ color: "#64748b", fontSize: 14, marginTop: 4 }}>Deep Research</p>
      </div>

      {/* Error */}
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

      {/* Loading */}
      {loading && (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              style={{
                background: "#1a1d27",
                border: "1px solid #2a2f45",
                borderRadius: 10,
                height: 72,
                animation: "pulse 1.5s ease-in-out infinite",
              }}
            />
          ))}
        </div>
      )}

      {/* Annual Reports */}
      {data && (
        <section>
          <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 16 }}>
            <h2
              style={{
                fontSize: 16,
                fontWeight: 600,
                color: "#94a3b8",
                textTransform: "uppercase",
                letterSpacing: "0.05em",
              }}
            >
              Annual Reports
            </h2>
            <span
              style={{
                fontSize: 11,
                fontWeight: 600,
                color: data.annualReportsSource === "NSE" ? "#22c55e" : "#f59e0b",
                background: data.annualReportsSource === "NSE"
                  ? "rgba(34,197,94,0.12)"
                  : "rgba(245,158,11,0.12)",
                border: `1px solid ${data.annualReportsSource === "NSE"
                  ? "rgba(34,197,94,0.3)"
                  : "rgba(245,158,11,0.3)"}`,
                borderRadius: 20,
                padding: "2px 8px",
              }}
            >
              via {data.annualReportsSource}
            </span>
          </div>

          {data.annualReports.length === 0 ? (
            <p style={{ color: "#475569", fontSize: 14 }}>No annual reports found.</p>
          ) : (
            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              {data.annualReports.map((report) => (
                <div
                  key={report.seqNumber}
                  style={{
                    background: "#1a1d27",
                    border: "1px solid #2a2f45",
                    borderRadius: 10,
                    padding: "16px 20px",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    gap: 16,
                  }}
                >
                  <div>
                    <div style={{ color: "#e2e8f0", fontSize: 14, fontWeight: 500 }}>
                      {report.subject || report.year}
                    </div>
                    {report.issuer && (
                      <div style={{ color: "#64748b", fontSize: 12, marginTop: 2 }}>
                        {report.issuer}
                      </div>
                    )}
                    <div style={{ color: "#475569", fontSize: 12, marginTop: 2 }}>
                      {report.year}
                    </div>
                  </div>
                  <a
                    href={report.pdfLink}
                    target="_blank"
                    rel="noopener noreferrer"
                    style={{
                      display: "inline-flex",
                      alignItems: "center",
                      gap: 6,
                      background: "rgba(99,102,241,0.15)",
                      border: "1px solid rgba(99,102,241,0.3)",
                      color: "#818cf8",
                      borderRadius: 8,
                      padding: "8px 14px",
                      fontSize: 13,
                      fontWeight: 500,
                      textDecoration: "none",
                      whiteSpace: "nowrap",
                      flexShrink: 0,
                    }}
                  >
                    View PDF
                  </a>
                </div>
              ))}
            </div>
          )}
        </section>
      )}

      <style>{`
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.4; }
        }
      `}</style>
    </div>
  );
}
