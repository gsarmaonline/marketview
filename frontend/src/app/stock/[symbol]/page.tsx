"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";

// ── Types ──────────────────────────────────────────────────────────────────

interface StockPrice {
  symbol: string;
  price: number;
  currency: string;
  short_name: string;
}

interface AnnualReport {
  seqNumber: number;
  issuer: string;
  year: string;
  subject: string;
  pdfLink: string;
}

interface SupplyChainEntity {
  name: string;
  relationship: string;
  amount?: string;
}

interface ProfitAndLoss {
  revenueFromOperations?: string;
  otherIncome?: string;
  totalIncome?: string;
  materialCost?: string;
  employeeBenefits?: string;
  financeCosts?: string;
  depreciation?: string;
  otherExpenses?: string;
  totalExpenses?: string;
  profitBeforeTax?: string;
  taxExpense?: string;
  profitAfterTax?: string;
}

interface BalanceSheet {
  totalAssets?: string;
  fixedAssets?: string;
  currentAssets?: string;
  cash?: string;
  inventory?: string;
  receivables?: string;
  totalEquity?: string;
  longTermDebt?: string;
  currentLiabilities?: string;
}

interface CashFlow {
  fromOperations?: string;
  fromInvesting?: string;
  fromFinancing?: string;
  netChange?: string;
}

interface FinancialHighlights {
  eps?: string;
  bookValuePerShare?: string;
  dividendPerShare?: string;
  roe?: string;
  roce?: string;
  debtToEquity?: string;
}

interface Financials {
  pnl: ProfitAndLoss;
  balanceSheet: BalanceSheet;
  cashFlow: CashFlow;
  highlights: FinancialHighlights;
}

interface DeepResearch {
  symbol: string;
  annualReports: AnnualReport[];
  annualReportsSource: string;
  supplyChain?: SupplyChainEntity[];
  financials?: Financials;
  parsedReportYear?: string;
}

interface NewsItem {
  title: string;
  description: string;
  link: string;
  publishedAt: string;
  source: string;
  symbol?: string;
}

// ── Helpers ─────────────────────────────────────────────────────────────────

function timeAgo(iso: string): string {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

const sourceColors: Record<string, string> = {
  "Economic Times": "#ff6b35",
  Moneycontrol: "#0077cc",
  "Business Standard": "#c0392b",
};

function hasData(obj: Record<string, string | undefined> | undefined): boolean {
  if (!obj) return false;
  return Object.values(obj).some((v) => v && v.trim() !== "");
}

function fmtNum(raw: string | undefined): string {
  if (!raw) return "—";
  const n = parseFloat(raw.replace(/,/g, ""));
  if (isNaN(n)) return raw;
  if (Math.abs(n) >= 1_00_00_000) return `₹${(n / 1_00_00_000).toFixed(2)} Cr`;
  if (Math.abs(n) >= 1_00_000) return `₹${(n / 1_00_000).toFixed(2)} L`;
  return `₹${n.toLocaleString("en-IN")}`;
}

const RELATIONSHIP_COLORS: Record<string, { bg: string; border: string; text: string }> = {
  subsidiary:              { bg: "rgba(99,102,241,0.12)",  border: "rgba(99,102,241,0.3)",  text: "#818cf8" },
  wholly_owned_subsidiary: { bg: "rgba(99,102,241,0.18)",  border: "rgba(99,102,241,0.4)",  text: "#a5b4fc" },
  holding:                 { bg: "rgba(245,158,11,0.12)",  border: "rgba(245,158,11,0.3)",  text: "#fbbf24" },
  associate:               { bg: "rgba(34,211,238,0.12)",  border: "rgba(34,211,238,0.3)",  text: "#67e8f9" },
  joint_venture:           { bg: "rgba(16,185,129,0.12)",  border: "rgba(16,185,129,0.3)",  text: "#6ee7b7" },
  key_management:          { bg: "rgba(239,68,68,0.10)",   border: "rgba(239,68,68,0.25)",  text: "#fca5a5" },
  promoter:                { bg: "rgba(236,72,153,0.10)",  border: "rgba(236,72,153,0.25)", text: "#f9a8d4" },
  supplier:                { bg: "rgba(251,146,60,0.12)",  border: "rgba(251,146,60,0.3)",  text: "#fdba74" },
  customer:                { bg: "rgba(52,211,153,0.12)",  border: "rgba(52,211,153,0.3)",  text: "#6ee7b7" },
  related_party:           { bg: "rgba(148,163,184,0.10)", border: "rgba(148,163,184,0.2)", text: "#94a3b8" },
};

function RelationshipBadge({ rel }: { rel: string }) {
  const c = RELATIONSHIP_COLORS[rel] ?? RELATIONSHIP_COLORS.related_party;
  const label = rel.replace(/_/g, " ");
  return (
    <span
      style={{
        fontSize: 11,
        fontWeight: 600,
        background: c.bg,
        border: `1px solid ${c.border}`,
        color: c.text,
        borderRadius: 20,
        padding: "2px 8px",
        whiteSpace: "nowrap",
        textTransform: "capitalize",
      }}
    >
      {label}
    </span>
  );
}

// ── Sub-components ─────────────────────────────────────────────────────────

function SearchBar({ currentSymbol }: { currentSymbol: string }) {
  const [query, setQuery] = useState(currentSymbol);
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const s = query.trim().toUpperCase();
    if (s && s !== currentSymbol) router.push(`/stock/${s}`);
  };

  return (
    <form onSubmit={handleSubmit} style={{ display: "flex", gap: 8 }}>
      <input
        ref={inputRef}
        value={query}
        onChange={(e) => setQuery(e.target.value.toUpperCase())}
        placeholder="Search symbol… (e.g. RELIANCE)"
        style={{
          background: "#1a1d27",
          border: "1px solid #2a2f45",
          borderRadius: 8,
          padding: "9px 14px",
          color: "#e2e8f0",
          fontSize: 13,
          outline: "none",
          width: 240,
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
          padding: "9px 18px",
          fontSize: 13,
          fontWeight: 500,
          cursor: query.trim() ? "pointer" : "not-allowed",
          opacity: query.trim() ? 1 : 0.5,
          whiteSpace: "nowrap",
        }}
      >
        Search
      </button>
    </form>
  );
}

function MetricCard({
  label,
  value,
  sub,
  highlight,
}: {
  label: string;
  value: string;
  sub?: string;
  highlight?: boolean;
}) {
  return (
    <div
      style={{
        background: highlight ? "rgba(99,102,241,0.08)" : "#1a1d27",
        border: highlight ? "1px solid rgba(99,102,241,0.3)" : "1px solid #2a2f45",
        borderRadius: 10,
        padding: "14px 18px",
      }}
    >
      <div
        style={{
          fontSize: 11,
          fontWeight: 600,
          color: "#64748b",
          textTransform: "uppercase",
          letterSpacing: "0.06em",
          marginBottom: 4,
        }}
      >
        {label}
      </div>
      <div
        style={{
          fontSize: 18,
          fontWeight: 700,
          color: highlight ? "#818cf8" : (value === "—" ? "#475569" : "#e2e8f0"),
        }}
      >
        {value}
      </div>
      {sub && (
        <div style={{ fontSize: 11, color: "#475569", marginTop: 2 }}>{sub}</div>
      )}
    </div>
  );
}

function RatioChip({ label, value }: { label: string; value: string | undefined }) {
  if (!value) return null;
  return (
    <div
      style={{
        background: "#1a1d27",
        border: "1px solid #2a2f45",
        borderRadius: 10,
        padding: "12px 16px",
        display: "flex",
        flexDirection: "column",
        gap: 4,
      }}
    >
      <div style={{ color: "#64748b", fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>
        {label}
      </div>
      <div style={{ color: "#e2e8f0", fontSize: 16, fontWeight: 700 }}>{value}</div>
    </div>
  );
}

function EmptyTable({ columns }: { columns: string[] }) {
  return (
    <div style={{ overflowX: "auto" }}>
      <table
        style={{
          width: "100%",
          borderCollapse: "collapse",
          fontSize: 13,
        }}
      >
        <thead>
          <tr>
            {columns.map((col) => (
              <th
                key={col}
                style={{
                  textAlign: "left",
                  padding: "10px 14px",
                  borderBottom: "1px solid #2a2f45",
                  fontSize: 11,
                  fontWeight: 600,
                  textTransform: "uppercase",
                  letterSpacing: "0.06em",
                  color: "#64748b",
                  background: "#1a1d27",
                  whiteSpace: "nowrap",
                }}
              >
                {col}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          <tr>
            <td
              colSpan={columns.length}
              style={{
                padding: "32px 14px",
                color: "#475569",
                fontSize: 13,
                textAlign: "center",
              }}
            >
              Data not yet available
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  );
}

function SectionCard({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div
      style={{
        background: "#1a1d27",
        border: "1px solid #2a2f45",
        borderRadius: 12,
        overflow: "hidden",
      }}
    >
      <div
        style={{
          padding: "16px 20px",
          borderBottom: "1px solid #2a2f45",
          fontSize: 14,
          fontWeight: 600,
          color: "#94a3b8",
          textTransform: "uppercase",
          letterSpacing: "0.05em",
        }}
      >
        {title}
      </div>
      <div style={{ padding: "0" }}>{children}</div>
    </div>
  );
}

function NewsCard({ item }: { item: NewsItem }) {
  const color = sourceColors[item.source] ?? "#3b82f6";
  return (
    <a
      href={item.link}
      target="_blank"
      rel="noopener noreferrer"
      style={{
        display: "block",
        padding: "14px 20px",
        borderBottom: "1px solid #2a2f45",
        textDecoration: "none",
        color: "inherit",
        transition: "background 0.15s",
      }}
      onMouseEnter={(e) =>
        ((e.currentTarget as HTMLElement).style.background = "#22263a")
      }
      onMouseLeave={(e) =>
        ((e.currentTarget as HTMLElement).style.background = "transparent")
      }
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          marginBottom: 6,
        }}
      >
        <span
          style={{
            background: `${color}22`,
            color,
            border: `1px solid ${color}44`,
            borderRadius: 4,
            padding: "1px 7px",
            fontSize: 10,
            fontWeight: 600,
            textTransform: "uppercase",
            letterSpacing: "0.04em",
          }}
        >
          {item.source}
        </span>
        {item.publishedAt && (
          <span style={{ color: "#475569", fontSize: 11 }}>
            {timeAgo(item.publishedAt)}
          </span>
        )}
      </div>
      <div
        style={{
          fontSize: 13,
          fontWeight: 600,
          color: "#e2e8f0",
          lineHeight: 1.5,
          marginBottom: item.description ? 4 : 0,
        }}
      >
        {item.title}
      </div>
      {item.description && (
        <p
          style={{
            fontSize: 12,
            color: "#64748b",
            lineHeight: 1.5,
            display: "-webkit-box",
            WebkitLineClamp: 2,
            WebkitBoxOrient: "vertical",
            overflow: "hidden",
          }}
        >
          {item.description}
        </p>
      )}
    </a>
  );
}

// ── Main page ──────────────────────────────────────────────────────────────

type Tab = "overview" | "quarterly" | "profit_loss" | "balance_sheet" | "documents" | "news";

const TABS: { id: Tab; label: string }[] = [
  { id: "overview", label: "Overview" },
  { id: "quarterly", label: "Quarterly Results" },
  { id: "profit_loss", label: "Profit & Loss" },
  { id: "balance_sheet", label: "Balance Sheet" },
  { id: "documents", label: "Documents" },
  { id: "news", label: "News" },
];

export default function StockPage() {
  const params = useParams();
  const symbol = (params.symbol as string).toUpperCase();

  const [priceData, setPriceData] = useState<StockPrice | null>(null);
  const [priceLoading, setPriceLoading] = useState(true);

  const [researchData, setResearchData] = useState<DeepResearch | null>(null);
  const [researchLoading, setResearchLoading] = useState(true);
  const [researchError, setResearchError] = useState<string | null>(null);

  const [news, setNews] = useState<NewsItem[]>([]);
  const [newsLoading, setNewsLoading] = useState(false);

  const [activeTab, setActiveTab] = useState<Tab>("overview");

  // Price
  const fetchPrice = useCallback(async () => {
    try {
      const res = await fetch(`/api/stock/${symbol}/price`);
      if (res.ok) {
        const data: StockPrice = await res.json();
        setPriceData(data);
      }
    } catch {
      // price is optional; don't block the page
    } finally {
      setPriceLoading(false);
    }
  }, [symbol]);

  // Deep research (annual reports + financials + supply chain)
  const fetchResearch = useCallback(async () => {
    try {
      const res = await fetch(`/api/stock/${symbol}/deep-research`);
      if (!res.ok) throw new Error(`Server error: ${res.status}`);
      const data: DeepResearch = await res.json();
      setResearchData(data);
      setResearchError(null);
    } catch (err) {
      setResearchError(err instanceof Error ? err.message : "Failed to fetch");
    } finally {
      setResearchLoading(false);
    }
  }, [symbol]);

  // News (only when tab is active)
  const fetchNews = useCallback(async () => {
    setNewsLoading(true);
    try {
      const res = await fetch(`/api/news/stock/${encodeURIComponent(symbol)}`);
      if (res.ok) {
        const data: NewsItem[] = await res.json();
        setNews(data);
      }
    } catch {
      // non-fatal
    } finally {
      setNewsLoading(false);
    }
  }, [symbol]);

  useEffect(() => {
    fetchPrice();
    fetchResearch();
  }, [fetchPrice, fetchResearch]);

  // Lazy-load news when tab is first opened
  const newsFetched = useRef(false);
  useEffect(() => {
    if (activeTab === "news" && !newsFetched.current) {
      newsFetched.current = true;
      fetchNews();
    }
  }, [activeTab, fetchNews]);

  const tabStyle = (tab: Tab) => ({
    padding: "8px 16px",
    fontSize: 13,
    fontWeight: 600,
    borderRadius: 8,
    cursor: "pointer" as const,
    border:
      activeTab === tab
        ? "1px solid rgba(99,102,241,0.4)"
        : "1px solid transparent",
    background:
      activeTab === tab ? "rgba(99,102,241,0.15)" : "transparent",
    color: activeTab === tab ? "#818cf8" : "#64748b",
    transition: "all 0.15s",
    whiteSpace: "nowrap" as const,
  });

  const pnl = researchData?.financials?.pnl;
  const bs = researchData?.financials?.balanceSheet;
  const cf = researchData?.financials?.cashFlow;
  const hl = researchData?.financials?.highlights;

  return (
    <div style={{ maxWidth: 1100, margin: "0 auto", padding: "32px 24px" }}>
      {/* ── Top bar ── */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          flexWrap: "wrap",
          gap: 16,
          marginBottom: 32,
        }}
      >
        <Link
          href="/"
          style={{
            color: "#64748b",
            fontSize: 13,
            textDecoration: "none",
            display: "inline-flex",
            alignItems: "center",
            gap: 4,
          }}
        >
          ← Dashboard
        </Link>
        <SearchBar currentSymbol={symbol} />
      </div>

      {/* ── Stock header ── */}
      <div style={{ marginBottom: 28 }}>
        <div
          style={{
            display: "flex",
            alignItems: "flex-start",
            gap: 24,
            flexWrap: "wrap",
          }}
        >
          <div style={{ flex: 1 }}>
            <h1
              style={{
                fontSize: 28,
                fontWeight: 700,
                color: "#e2e8f0",
                letterSpacing: "-0.5px",
                lineHeight: 1.2,
              }}
            >
              {priceData?.short_name || symbol}
            </h1>
            <div
              style={{
                fontSize: 13,
                color: "#64748b",
                marginTop: 4,
              }}
            >
              NSE: {symbol}
            </div>
          </div>

          {/* Price block */}
          <div style={{ textAlign: "right" }}>
            {priceLoading ? (
              <div
                style={{
                  width: 120,
                  height: 40,
                  background: "#1a1d27",
                  borderRadius: 8,
                  animation: "pulse 1.5s ease-in-out infinite",
                }}
              />
            ) : priceData ? (
              <>
                <div
                  style={{
                    fontSize: 32,
                    fontWeight: 700,
                    color: "#e2e8f0",
                    fontVariantNumeric: "tabular-nums",
                    lineHeight: 1,
                  }}
                >
                  {priceData.currency === "INR" ? "₹" : priceData.currency}{" "}
                  {priceData.price.toLocaleString("en-IN", {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 2,
                  })}
                </div>
                <div
                  style={{ fontSize: 12, color: "#475569", marginTop: 4 }}
                >
                  Current Market Price
                </div>
              </>
            ) : (
              <div style={{ color: "#475569", fontSize: 13 }}>
                Price unavailable
              </div>
            )}
          </div>
        </div>
      </div>

      {/* ── Key metrics ── */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fill, minmax(150px, 1fr))",
          gap: 12,
          marginBottom: 32,
        }}
      >
        <MetricCard label="EPS" value={hl?.eps ?? "—"} />
        <MetricCard label="Book Value" value={hl?.bookValuePerShare ?? "—"} />
        <MetricCard label="Dividend / Share" value={hl?.dividendPerShare ?? "—"} />
        <MetricCard label="ROE" value={hl?.roe ? `${hl.roe}%` : "—"} />
        <MetricCard label="ROCE" value={hl?.roce ? `${hl.roce}%` : "—"} />
        <MetricCard label="Debt / Equity" value={hl?.debtToEquity ?? "—"} />
        <MetricCard label="Revenue" value={fmtNum(pnl?.revenueFromOperations)} highlight />
        <MetricCard label="Net Profit" value={fmtNum(pnl?.profitAfterTax)} highlight />
        <MetricCard label="Total Assets" value={fmtNum(bs?.totalAssets)} />
      </div>

      {/* ── Tabs ── */}
      <div
        style={{
          display: "flex",
          gap: 4,
          borderBottom: "1px solid #2a2f45",
          marginBottom: 24,
          overflowX: "auto",
          paddingBottom: 1,
        }}
      >
        {TABS.map((t) => (
          <button key={t.id} onClick={() => setActiveTab(t.id)} style={tabStyle(t.id)}>
            {t.label}
          </button>
        ))}
      </div>

      {/* ── Tab content ── */}

      {activeTab === "overview" && (
        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          {/* Supply Chain / Related Party Transactions */}
          {researchData?.supplyChain && researchData.supplyChain.length > 0 && (
            <SectionCard
              title={`Related Party Transactions${researchData.parsedReportYear ? ` · ${researchData.parsedReportYear}` : ""}`}
            >
              <table style={{ width: "100%", borderCollapse: "collapse" }}>
                <thead>
                  <tr style={{ borderBottom: "1px solid #2a2f45" }}>
                    <th style={{ textAlign: "left", padding: "12px 16px", color: "#64748b", fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>
                      Company
                    </th>
                    <th style={{ textAlign: "left", padding: "12px 16px", color: "#64748b", fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>
                      Relationship
                    </th>
                    <th style={{ textAlign: "right", padding: "12px 16px", color: "#64748b", fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>
                      Amount
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {researchData.supplyChain.map((entity, idx) => (
                    <tr
                      key={idx}
                      style={{
                        borderBottom: idx < researchData.supplyChain!.length - 1 ? "1px solid #1e2235" : "none",
                      }}
                    >
                      <td style={{ padding: "12px 16px", color: "#e2e8f0", fontSize: 13 }}>
                        {entity.name}
                      </td>
                      <td style={{ padding: "12px 16px" }}>
                        <RelationshipBadge rel={entity.relationship} />
                      </td>
                      <td style={{ padding: "12px 16px", color: "#94a3b8", fontSize: 13, textAlign: "right" }}>
                        {entity.amount ?? "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </SectionCard>
          )}

          <SectionCard title="Peer Comparison">
            <EmptyTable
              columns={["Company", "CMP", "P/E", "Market Cap", "Div. Yield", "NP Qtr", "Qtr Profit Var", "Sales Qtr", "Qtr Sales Var", "ROCE"]}
            />
          </SectionCard>
        </div>
      )}

      {activeTab === "quarterly" && (
        <SectionCard title="Quarterly Results">
          <EmptyTable
            columns={["", "Sep 2024", "Dec 2024", "Mar 2025", "Jun 2025", "Sep 2025"]}
          />
        </SectionCard>
      )}

      {activeTab === "profit_loss" && (
        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          {pnl && hasData(pnl as Record<string, string | undefined>) ? (
            <>
              <SectionCard title="Profit & Loss">
                <div style={{ padding: 20, display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))", gap: 12 }}>
                  <MetricCard label="Revenue from Ops" value={fmtNum(pnl.revenueFromOperations)} highlight />
                  <MetricCard label="Other Income" value={fmtNum(pnl.otherIncome)} />
                  <MetricCard label="Total Income" value={fmtNum(pnl.totalIncome)} />
                  <MetricCard label="Material Cost" value={fmtNum(pnl.materialCost)} />
                  <MetricCard label="Employee Benefits" value={fmtNum(pnl.employeeBenefits)} />
                  <MetricCard label="Finance Costs" value={fmtNum(pnl.financeCosts)} />
                  <MetricCard label="Depreciation" value={fmtNum(pnl.depreciation)} />
                  <MetricCard label="Other Expenses" value={fmtNum(pnl.otherExpenses)} />
                  <MetricCard label="Total Expenses" value={fmtNum(pnl.totalExpenses)} />
                  <MetricCard label="Profit Before Tax" value={fmtNum(pnl.profitBeforeTax)} />
                  <MetricCard label="Tax Expense" value={fmtNum(pnl.taxExpense)} />
                  <MetricCard label="Profit After Tax" value={fmtNum(pnl.profitAfterTax)} highlight />
                </div>
              </SectionCard>
              {cf && hasData(cf as Record<string, string | undefined>) && (
                <SectionCard title="Cash Flows">
                  <div style={{ padding: 20, display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))", gap: 12 }}>
                    <MetricCard label="From Operations" value={fmtNum(cf.fromOperations)} highlight />
                    <MetricCard label="From Investing" value={fmtNum(cf.fromInvesting)} />
                    <MetricCard label="From Financing" value={fmtNum(cf.fromFinancing)} />
                    <MetricCard label="Net Change" value={fmtNum(cf.netChange)} />
                  </div>
                </SectionCard>
              )}
              {hl && hasData(hl as Record<string, string | undefined>) && (
                <SectionCard title="Key Ratios">
                  <div style={{ padding: 20, display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(160px, 1fr))", gap: 12 }}>
                    <RatioChip label="EPS" value={hl.eps} />
                    <RatioChip label="Book Value / Share" value={hl.bookValuePerShare} />
                    <RatioChip label="Dividend / Share" value={hl.dividendPerShare} />
                    <RatioChip label="ROE" value={hl.roe ? `${hl.roe}%` : undefined} />
                    <RatioChip label="ROCE" value={hl.roce ? `${hl.roce}%` : undefined} />
                    <RatioChip label="Debt / Equity" value={hl.debtToEquity} />
                  </div>
                </SectionCard>
              )}
            </>
          ) : (
            <>
              <SectionCard title="Profit & Loss">
                {researchLoading ? (
                  <div style={{ padding: 20, color: "#64748b", fontSize: 13 }}>Loading…</div>
                ) : (
                  <EmptyTable columns={["", "Mar 2021", "Mar 2022", "Mar 2023", "Mar 2024", "Mar 2025"]} />
                )}
              </SectionCard>
              <SectionCard title="Cash Flows">
                <EmptyTable columns={["", "Mar 2021", "Mar 2022", "Mar 2023", "Mar 2024", "Mar 2025"]} />
              </SectionCard>
            </>
          )}
        </div>
      )}

      {activeTab === "balance_sheet" && (
        bs && hasData(bs as Record<string, string | undefined>) ? (
          <SectionCard title="Balance Sheet">
            <div style={{ padding: 20, display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))", gap: 12 }}>
              <MetricCard label="Total Assets" value={fmtNum(bs.totalAssets)} highlight />
              <MetricCard label="Fixed Assets" value={fmtNum(bs.fixedAssets)} />
              <MetricCard label="Current Assets" value={fmtNum(bs.currentAssets)} />
              <MetricCard label="Cash" value={fmtNum(bs.cash)} />
              <MetricCard label="Inventory" value={fmtNum(bs.inventory)} />
              <MetricCard label="Receivables" value={fmtNum(bs.receivables)} />
              <MetricCard label="Total Equity" value={fmtNum(bs.totalEquity)} highlight />
              <MetricCard label="Long-Term Debt" value={fmtNum(bs.longTermDebt)} />
              <MetricCard label="Current Liabilities" value={fmtNum(bs.currentLiabilities)} />
            </div>
          </SectionCard>
        ) : (
          <SectionCard title="Balance Sheet">
            {researchLoading ? (
              <div style={{ padding: 20, color: "#64748b", fontSize: 13 }}>Loading…</div>
            ) : (
              <EmptyTable columns={["", "Mar 2021", "Mar 2022", "Mar 2023", "Mar 2024", "Mar 2025"]} />
            )}
          </SectionCard>
        )
      )}

      {activeTab === "documents" && (
        <SectionCard
          title={
            researchData
              ? `Annual Reports · via ${researchData.annualReportsSource}`
              : "Annual Reports"
          }
        >
          {researchLoading ? (
            <div style={{ padding: 20, color: "#64748b", fontSize: 13 }}>
              Loading documents…
            </div>
          ) : researchError ? (
            <div
              style={{
                margin: 16,
                background: "rgba(239,68,68,0.1)",
                border: "1px solid rgba(239,68,68,0.3)",
                borderRadius: 8,
                padding: "12px 16px",
                color: "#fca5a5",
                fontSize: 13,
              }}
            >
              ⚠ {researchError}
            </div>
          ) : !researchData || researchData.annualReports.length === 0 ? (
            <div style={{ padding: 20, color: "#475569", fontSize: 13 }}>
              No annual reports found.
            </div>
          ) : (
            researchData.annualReports.map((report) => (
              <div
                key={report.seqNumber}
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
                  gap: 16,
                  padding: "14px 20px",
                  borderBottom: "1px solid #2a2f45",
                }}
              >
                <div>
                  <div style={{ fontSize: 13, fontWeight: 500, color: "#e2e8f0" }}>
                    {report.subject || report.year}
                  </div>
                  {report.issuer && (
                    <div style={{ fontSize: 12, color: "#64748b", marginTop: 2 }}>
                      {report.issuer}
                    </div>
                  )}
                  <div style={{ fontSize: 11, color: "#475569", marginTop: 2 }}>
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
                    padding: "6px 14px",
                    fontSize: 12,
                    fontWeight: 500,
                    textDecoration: "none",
                    whiteSpace: "nowrap",
                    flexShrink: 0,
                  }}
                >
                  View PDF
                </a>
              </div>
            ))
          )}
        </SectionCard>
      )}

      {activeTab === "news" && (
        <SectionCard title={`News · ${symbol}`}>
          {newsLoading ? (
            <div style={{ padding: 20, color: "#64748b", fontSize: 13 }}>
              Loading news…
            </div>
          ) : news.length === 0 ? (
            <div style={{ padding: 20, color: "#475569", fontSize: 13 }}>
              No news found for {symbol}. News is ingested from RSS feeds — check back later.
            </div>
          ) : (
            news.map((item, i) => (
              <NewsCard key={`${item.link}-${i}`} item={item} />
            ))
          )}
        </SectionCard>
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
