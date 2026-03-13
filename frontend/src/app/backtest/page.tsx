"use client";

import { useState, FormEvent } from "react";

interface StrategyConfig {
  name: string;
}

interface Trade {
  entry_date: string;
  entry_price: number;
  exit_date: string;
  exit_price: number;
  shares: number;
  pnl: number;
  return_pct: number;
}

interface EquityPoint {
  date: string;
  value: number;
}

interface Metrics {
  total_return_pct: number;
  cagr_pct: number;
  max_drawdown_pct: number;
  sharpe_ratio: number;
  win_rate_pct: number;
  total_trades: number;
}

interface BacktestResult {
  strategy: string;
  symbol: string;
  from: string;
  to: string;
  capital: number;
  final_value: number;
  trades: Trade[];
  equity_curve: EquityPoint[];
  metrics: Metrics;
}

const STRATEGIES = [
  { value: "buy_and_hold", label: "Buy & Hold" },
];

function fmt(n: number, prefix = "₹") {
  return prefix + n.toLocaleString("en-IN", { maximumFractionDigits: 2 });
}

function fmtPct(n: number) {
  const sign = n >= 0 ? "+" : "";
  return `${sign}${n.toFixed(2)}%`;
}

function MetricCard({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="summary-card">
      <div className="label">{label}</div>
      <div className="value" style={{ fontSize: "1.3rem" }}>{value}</div>
      {sub && <div style={{ fontSize: "0.75rem", color: "var(--text-muted)", marginTop: 2 }}>{sub}</div>}
    </div>
  );
}

function EquityChart({ curve, capital }: { curve: EquityPoint[]; capital: number }) {
  if (curve.length < 2) return null;

  const W = 800;
  const H = 200;
  const PAD = { top: 10, right: 10, bottom: 24, left: 10 };

  const values = curve.map((p) => p.value);
  const minVal = Math.min(...values);
  const maxVal = Math.max(...values);
  const range = maxVal - minVal || 1;

  const xScale = (i: number) =>
    PAD.left + (i / (curve.length - 1)) * (W - PAD.left - PAD.right);
  const yScale = (v: number) =>
    PAD.top + (1 - (v - minVal) / range) * (H - PAD.top - PAD.bottom);

  const points = curve.map((p, i) => `${xScale(i)},${yScale(p.value)}`).join(" ");

  // Filled area path
  const baselineY = yScale(Math.max(minVal, capital));
  const linePts = curve.map((p, i) => `${xScale(i)},${yScale(p.value)}`).join(" L");
  const areaPath = `M${xScale(0)},${baselineY} L${linePts} L${xScale(curve.length - 1)},${baselineY} Z`;

  const finalAboveCapital = curve[curve.length - 1].value >= capital;
  const lineColor = finalAboveCapital ? "var(--bullish)" : "var(--bearish)";

  const firstDate = curve[0].date;
  const lastDate = curve[curve.length - 1].date;
  const midDate = curve[Math.floor(curve.length / 2)].date;

  return (
    <div className="card" style={{ padding: "1rem 1.5rem" }}>
      <div style={{ fontSize: "0.75rem", color: "var(--text-muted)", textTransform: "uppercase", fontWeight: 600, marginBottom: "0.75rem" }}>
        Equity Curve
      </div>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        style={{ width: "100%", height: 200, display: "block" }}
        preserveAspectRatio="none"
      >
        <defs>
          <linearGradient id="equityGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={lineColor} stopOpacity="0.3" />
            <stop offset="100%" stopColor={lineColor} stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={areaPath} fill="url(#equityGradient)" />
        <polyline
          points={points}
          fill="none"
          stroke={lineColor}
          strokeWidth="2"
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      <div style={{ display: "flex", justifyContent: "space-between", fontSize: "0.7rem", color: "var(--text-muted)", marginTop: 4 }}>
        <span>{firstDate}</span>
        <span>{midDate}</span>
        <span>{lastDate}</span>
      </div>
    </div>
  );
}

export default function BacktestPage() {
  const today = new Date().toISOString().split("T")[0];
  const fiveYearsAgo = new Date(Date.now() - 5 * 365.25 * 24 * 3600 * 1000).toISOString().split("T")[0];

  const [symbol, setSymbol] = useState("RELIANCE");
  const [from, setFrom] = useState(fiveYearsAgo);
  const [to, setTo] = useState(today);
  const [capital, setCapital] = useState("100000");
  const [strategy, setStrategy] = useState<StrategyConfig>({ name: "buy_and_hold" });

  const [result, setResult] = useState<BacktestResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    setResult(null);

    try {
      const resp = await fetch("/api/backtest", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          symbol: symbol.trim().toUpperCase(),
          from,
          to,
          capital: parseFloat(capital),
          strategy,
        }),
      });
      const data = await resp.json();
      if (!resp.ok) {
        setError(data.error || "Backtest failed");
        return;
      }
      setResult(data);
    } catch {
      setError("Network error");
    } finally {
      setLoading(false);
    }
  }

  const m = result?.metrics;

  return (
    <div>
      <h1>Backtesting</h1>

      <div className="card">
        <form onSubmit={handleSubmit}>
          <div className="form-row" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))", gap: "1rem", marginBottom: "1rem" }}>
            <div className="form-group" style={{ margin: 0 }}>
              <label>Symbol</label>
              <input
                value={symbol}
                onChange={(e) => setSymbol(e.target.value)}
                placeholder="e.g. RELIANCE"
                required
              />
            </div>
            <div className="form-group" style={{ margin: 0 }}>
              <label>From</label>
              <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} required />
            </div>
            <div className="form-group" style={{ margin: 0 }}>
              <label>To</label>
              <input type="date" value={to} onChange={(e) => setTo(e.target.value)} required />
            </div>
            <div className="form-group" style={{ margin: 0 }}>
              <label>Capital (₹)</label>
              <input
                type="number"
                value={capital}
                onChange={(e) => setCapital(e.target.value)}
                min="1"
                required
              />
            </div>
          </div>

          <div style={{ display: "flex", flexWrap: "wrap", gap: "1rem", alignItems: "flex-end" }}>
            <div className="form-group" style={{ margin: 0, minWidth: 160 }}>
              <label>Strategy</label>
              <select
                value={strategy.name}
                onChange={(e) => setStrategy({ name: e.target.value })}
              >
                {STRATEGIES.map((s) => (
                  <option key={s.value} value={s.value}>{s.label}</option>
                ))}
              </select>
            </div>
            <div>
              <button className="btn btn-primary" type="submit" disabled={loading}>
                {loading ? "Running…" : "Run Backtest"}
              </button>
            </div>
          </div>
        </form>
      </div>

      {error && (
        <div className="card" style={{ borderColor: "var(--bearish-border)", color: "var(--bearish)" }}>
          {error}
        </div>
      )}

      {result && (
        <>
          <div className="summary-grid" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(130px, 1fr))" }}>
            <MetricCard
              label="Final Value"
              value={fmt(result.final_value)}
              sub={fmtPct(m!.total_return_pct)}
            />
            <MetricCard label="Total Return" value={fmtPct(m!.total_return_pct)} />
            <MetricCard label="CAGR" value={fmtPct(m!.cagr_pct)} />
            <MetricCard label="Max Drawdown" value={`-${m!.max_drawdown_pct.toFixed(2)}%`} />
            <MetricCard label="Sharpe Ratio" value={m!.sharpe_ratio.toFixed(2)} />
            <MetricCard
              label="Win Rate"
              value={`${m!.win_rate_pct.toFixed(0)}%`}
              sub={`${m!.total_trades} trade${m!.total_trades !== 1 ? "s" : ""}`}
            />
          </div>

          <EquityChart curve={result.equity_curve} capital={result.capital} />

          {result.trades.length > 0 && (
            <div className="card" style={{ padding: 0, overflow: "hidden" }}>
              <div style={{ padding: "1rem 1.5rem", borderBottom: "1px solid var(--border)", fontSize: "0.75rem", textTransform: "uppercase", fontWeight: 600, color: "var(--text-muted)" }}>
                Trades
              </div>
              <table>
                <thead>
                  <tr>
                    <th>Entry Date</th>
                    <th>Entry Price</th>
                    <th>Exit Date</th>
                    <th>Exit Price</th>
                    <th>Shares</th>
                    <th>P&amp;L</th>
                    <th>Return</th>
                  </tr>
                </thead>
                <tbody>
                  {result.trades.map((t, i) => (
                    <tr key={i}>
                      <td>{t.entry_date}</td>
                      <td>{fmt(t.entry_price)}</td>
                      <td>{t.exit_date}</td>
                      <td>{fmt(t.exit_price)}</td>
                      <td>{t.shares.toFixed(0)}</td>
                      <td style={{ color: t.pnl >= 0 ? "var(--bullish)" : "var(--bearish)" }}>
                        {fmt(t.pnl)}
                      </td>
                      <td style={{ color: t.return_pct >= 0 ? "var(--bullish)" : "var(--bearish)" }}>
                        {fmtPct(t.return_pct)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
