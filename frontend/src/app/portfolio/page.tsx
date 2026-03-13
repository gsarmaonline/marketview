"use client";

import { useEffect, useState, useRef, FormEvent } from "react";

// ── Analysis types ──────────────────────────────────────────────────────────

interface FundAllocation {
  fundName: string;
  percentage: number;
}

interface StockOverlap {
  stockName: string;
  symbol?: string;
  funds: FundAllocation[];
}

interface PortfolioFundAnalysis {
  name: string;
  schemeCode?: number;
  fundHouse?: string;
  category?: string;
  holdings: { name: string; symbol?: string; percentage: number }[];
}

interface PortfolioAnalysis {
  funds: PortfolioFundAnalysis[];
  overlaps: StockOverlap[];
  recommendations: string[];
}

type AssetType = "stock" | "fd" | "mutual_fund" | "gold" | "other";

interface Holding {
  id: number;
  asset_type: AssetType;
  name: string;
  quantity: number | null;
  buy_price: number | null;
  current_value: number | null;
  buy_date: string | null;
  notes: string;
  metadata: Record<string, unknown>;
}

const ASSET_LABELS: Record<AssetType, string> = {
  stock: "Stock",
  fd: "Fixed Deposit",
  mutual_fund: "Mutual Fund",
  gold: "Gold",
  other: "Other",
};

const emptyForm = (): Omit<Holding, "id" | "metadata"> & { metadata: string } => ({
  asset_type: "stock",
  name: "",
  quantity: null,
  buy_price: null,
  current_value: null,
  buy_date: "",
  notes: "",
  metadata: "{}",
});

function fmt(n: number | null, prefix = "₹") {
  if (n == null) return "—";
  return prefix + n.toLocaleString("en-IN", { maximumFractionDigits: 2 });
}

function totalInvested(holdings: Holding[]) {
  return holdings.reduce((sum, h) => {
    if (h.buy_price != null && h.quantity != null) return sum + h.buy_price * h.quantity;
    if (h.buy_price != null) return sum + h.buy_price;
    return sum;
  }, 0);
}

function totalCurrent(holdings: Holding[]) {
  return holdings.reduce((sum, h) => sum + (h.current_value ?? 0), 0);
}

export default function PortfolioPage() {
  const [holdings, setHoldings] = useState<Holding[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Holding | null>(null);
  const [form, setForm] = useState(emptyForm());
  const [saving, setSaving] = useState(false);
  const [priceFetching, setPriceFetching] = useState(false);
  const [priceError, setPriceError] = useState("");
  const [showAnalysis, setShowAnalysis] = useState(false);
  const [analysis, setAnalysis] = useState<PortfolioAnalysis | null>(null);
  const [analysing, setAnalysing] = useState(false);
  const [analysisError, setAnalysisError] = useState("");
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const load = () =>
    fetch("/api/portfolio/holdings")
      .then((r) => r.json())
      .then(setHoldings)
      .catch(() => setError("Failed to load holdings"))
      .finally(() => setLoading(false));

  useEffect(() => { load(); }, []);

  // Auto-fetch current price when stock symbol changes
  useEffect(() => {
    if (form.asset_type !== "stock" || !form.name.trim()) {
      setPriceError("");
      return;
    }

    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(async () => {
      setPriceFetching(true);
      setPriceError("");
      try {
        const res = await fetch(`/api/stock/${encodeURIComponent(form.name.trim())}/price`);
        if (!res.ok) throw new Error("Symbol not found");
        const data = await res.json();
        const price = data.price as number;
        const qty = form.quantity;
        setForm((prev) => ({
          ...prev,
          current_value: qty != null ? parseFloat((price * qty).toFixed(2)) : parseFloat(price.toFixed(2)),
        }));
      } catch {
        setPriceError("Could not fetch price — enter manually");
      } finally {
        setPriceFetching(false);
      }
    }, 600);

    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [form.name, form.asset_type]);

  // Recalculate current_value when quantity changes (for stocks with a fetched price)
  function handleQuantityChange(val: string) {
    const qty = val ? parseFloat(val) : null;
    setForm((prev) => {
      if (prev.asset_type === "stock" && prev.current_value != null && prev.quantity != null && prev.quantity !== 0) {
        const pricePerShare = prev.current_value / prev.quantity;
        return {
          ...prev,
          quantity: qty,
          current_value: qty != null ? parseFloat((pricePerShare * qty).toFixed(2)) : null,
        };
      }
      return { ...prev, quantity: qty };
    });
  }

  function openAdd() {
    setEditing(null);
    setForm(emptyForm());
    setPriceError("");
    setShowModal(true);
  }

  function openEdit(h: Holding) {
    setEditing(h);
    setForm({
      asset_type: h.asset_type,
      name: h.name,
      quantity: h.quantity,
      buy_price: h.buy_price,
      current_value: h.current_value,
      buy_date: h.buy_date ? h.buy_date.slice(0, 10) : "",
      notes: h.notes,
      metadata: JSON.stringify(h.metadata ?? {}, null, 2),
    });
    setPriceError("");
    setShowModal(true);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      let meta: Record<string, unknown> = {};
      try { meta = JSON.parse(form.metadata || "{}"); } catch { meta = {}; }

      const body = {
        asset_type: form.asset_type,
        name: form.name,
        quantity: form.quantity,
        buy_price: form.buy_price,
        current_value: form.current_value,
        buy_date: form.buy_date ? new Date(form.buy_date).toISOString() : null,
        notes: form.notes,
        metadata: meta,
      };

      const url = editing
        ? `/api/portfolio/holdings/${editing.id}`
        : "/api/portfolio/holdings";
      const method = editing ? "PUT" : "POST";

      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(await res.text());
      setShowModal(false);
      load();
    } catch (err) {
      alert("Error: " + (err as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: number) {
    if (!confirm("Delete this holding?")) return;
    await fetch(`/api/portfolio/holdings/${id}`, { method: "DELETE" });
    load();
  }

  const invested = totalInvested(holdings);
  const current = totalCurrent(holdings);
  const pnl = current - invested;

  // Group by asset type
  const groups = (Object.keys(ASSET_LABELS) as AssetType[]).map((type) => ({
    type,
    items: holdings.filter((h) => h.asset_type === type),
  })).filter((g) => g.items.length > 0);

  const isStock = form.asset_type === "stock";

  const hasMFs = holdings.some((h) => h.asset_type === "mutual_fund");

  async function handleAnalyse() {
    setAnalysing(true);
    setAnalysisError("");
    setAnalysis(null);
    setShowAnalysis(true);
    try {
      const res = await fetch("/api/portfolio/analyse");
      if (!res.ok) throw new Error(await res.text());
      const data: PortfolioAnalysis = await res.json();
      setAnalysis(data);
    } catch (err) {
      setAnalysisError("Failed to analyse portfolio: " + (err as Error).message);
    } finally {
      setAnalysing(false);
    }
  }

  return (
    <>
      <div className="portfolio-header">
        <h1>Portfolio</h1>
        <div style={{ display: "flex", gap: "0.75rem" }}>
          {hasMFs && (
            <button className="btn btn-secondary" onClick={handleAnalyse}>
              Analyse Portfolio
            </button>
          )}
          <button className="btn btn-primary" onClick={openAdd}>+ Add Holding</button>
        </div>
      </div>

      {/* Summary */}
      {holdings.length > 0 && (
        <div className="summary-grid">
          <div className="summary-card">
            <div className="label">Invested</div>
            <div className="value">{fmt(invested)}</div>
          </div>
          <div className="summary-card">
            <div className="label">Current Value</div>
            <div className="value">{current > 0 ? fmt(current) : "—"}</div>
          </div>
          {current > 0 && (
            <div className="summary-card">
              <div className="label">P&amp;L</div>
              <div className={`value ${pnl >= 0 ? "positive" : "negative"}`}>
                {pnl >= 0 ? "+" : ""}{fmt(pnl)}
              </div>
            </div>
          )}
          <div className="summary-card">
            <div className="label">Holdings</div>
            <div className="value">{holdings.length}</div>
          </div>
        </div>
      )}

      {loading && <p>Loading…</p>}
      {error && <p style={{ color: "red" }}>{error}</p>}

      {!loading && holdings.length === 0 && (
        <div className="card empty">
          <p>No holdings yet.</p>
          <p style={{ marginTop: "0.5rem" }}>
            <button className="btn btn-primary" onClick={openAdd}>Add your first holding</button>
          </p>
        </div>
      )}

      {/* Holdings by group */}
      {groups.map(({ type, items }) => (
        <div key={type} className="card" style={{ padding: 0, overflow: "hidden" }}>
          <div style={{ padding: "1rem 1.25rem", borderBottom: "1px solid #f0f0f0", display: "flex", alignItems: "center", gap: "0.75rem" }}>
            <span className={`badge badge-${type}`}>{ASSET_LABELS[type]}</span>
            <span style={{ color: "#888", fontSize: "0.85rem" }}>{items.length} holding{items.length !== 1 ? "s" : ""}</span>
          </div>
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Qty</th>
                <th>Buy Price</th>
                <th>Invested</th>
                <th>Current Value</th>
                <th>P&amp;L</th>
                <th>Buy Date</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {items.map((h) => {
                const inv = h.buy_price != null
                  ? (h.quantity != null ? h.buy_price * h.quantity : h.buy_price)
                  : null;
                const pl = inv != null && h.current_value != null ? h.current_value - inv : null;
                return (
                  <tr key={h.id}>
                    <td style={{ fontWeight: 500 }}>
                      {h.name}
                      {h.notes && <div style={{ fontSize: "0.8rem", color: "#888", marginTop: 2 }}>{h.notes}</div>}
                    </td>
                    <td>{h.quantity ?? "—"}</td>
                    <td>{fmt(h.buy_price)}</td>
                    <td>{fmt(inv)}</td>
                    <td>{fmt(h.current_value)}</td>
                    <td style={{ color: pl == null ? undefined : pl >= 0 ? "#15803d" : "#dc2626", fontWeight: 500 }}>
                      {pl != null ? (pl >= 0 ? "+" : "") + fmt(pl) : "—"}
                    </td>
                    <td style={{ color: "#888", fontSize: "0.85rem" }}>
                      {h.buy_date ? new Date(h.buy_date).toLocaleDateString("en-IN") : "—"}
                    </td>
                    <td>
                      <div style={{ display: "flex", gap: "0.4rem" }}>
                        <button className="btn btn-secondary btn-sm" onClick={() => openEdit(h)}>Edit</button>
                        <button className="btn btn-danger btn-sm" onClick={() => handleDelete(h.id)}>Delete</button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ))}

      {/* Analysis Modal */}
      {showAnalysis && (
        <div className="overlay" onClick={(e) => e.target === e.currentTarget && setShowAnalysis(false)}>
          <div className="modal" style={{ maxWidth: 820 }}>
            <div className="modal-header">
              <h2>Portfolio Analysis</h2>
              <button className="close-btn" onClick={() => setShowAnalysis(false)}>×</button>
            </div>

            {analysing && <p style={{ color: "var(--text-muted)" }}>Fetching fund holdings… this may take a moment.</p>}
            {analysisError && <p style={{ color: "var(--bearish)" }}>{analysisError}</p>}

            {analysis && (
              <>
                {/* Funds overview */}
                <section style={{ marginBottom: "1.5rem" }}>
                  <h3 style={{ fontSize: "0.85rem", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: "0.75rem", fontWeight: 600 }}>
                    Mutual Funds ({analysis.funds.length})
                  </h3>
                  {analysis.funds.length === 0
                    ? <p style={{ color: "var(--text-muted)" }}>No mutual funds found.</p>
                    : (
                      <table>
                        <thead>
                          <tr>
                            <th>Fund</th>
                            <th>Fund House</th>
                            <th>Category</th>
                            <th style={{ textAlign: "right" }}>Top Holdings</th>
                          </tr>
                        </thead>
                        <tbody>
                          {analysis.funds.map((f) => (
                            <tr key={f.name}>
                              <td style={{ fontWeight: 500 }}>{f.name}</td>
                              <td style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>{f.fundHouse || "—"}</td>
                              <td>
                                {f.category
                                  ? <span className="badge badge-mutual_fund">{f.category}</span>
                                  : <span style={{ color: "var(--text-muted)" }}>—</span>}
                              </td>
                              <td style={{ textAlign: "right", fontSize: "0.82rem", color: "var(--text-muted)" }}>
                                {f.holdings.length > 0
                                  ? f.holdings.slice(0, 3).map((h) => `${h.name} (${h.percentage.toFixed(1)}%)`).join(", ")
                                  : "No holdings data"}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )
                  }
                </section>

                {/* Overlap */}
                {analysis.overlaps.length > 0 && (
                  <section style={{ marginBottom: "1.5rem" }}>
                    <h3 style={{ fontSize: "0.85rem", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: "0.75rem", fontWeight: 600 }}>
                      Stock Overlap ({analysis.overlaps.length} shared stocks)
                    </h3>
                    <table>
                      <thead>
                        <tr>
                          <th>Stock</th>
                          {analysis.funds.map((f) => (
                            <th key={f.name} style={{ textAlign: "right" }}>{f.name.split(" ").slice(0, 2).join(" ")}</th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {analysis.overlaps.slice(0, 20).map((o) => {
                          const byFund = Object.fromEntries(o.funds.map((f) => [f.fundName, f.percentage]));
                          return (
                            <tr key={o.stockName}>
                              <td style={{ fontWeight: 500 }}>{o.stockName}</td>
                              {analysis.funds.map((f) => (
                                <td key={f.name} style={{ textAlign: "right", color: byFund[f.name] != null ? "var(--neutral)" : "var(--text-muted)" }}>
                                  {byFund[f.name] != null ? `${byFund[f.name].toFixed(1)}%` : "—"}
                                </td>
                              ))}
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                    {analysis.overlaps.length > 20 && (
                      <p style={{ color: "var(--text-muted)", fontSize: "0.82rem", marginTop: "0.5rem" }}>
                        … and {analysis.overlaps.length - 20} more shared stocks.
                      </p>
                    )}
                  </section>
                )}

                {analysis.overlaps.length === 0 && analysis.funds.some((f) => f.holdings.length > 0) && (
                  <section style={{ marginBottom: "1.5rem" }}>
                    <p style={{ color: "var(--bullish)" }}>No stock overlap found — funds are well-diversified from each other.</p>
                  </section>
                )}

                {/* Recommendations */}
                <section>
                  <h3 style={{ fontSize: "0.85rem", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: "0.75rem", fontWeight: 600 }}>
                    Recommendations
                  </h3>
                  <ul style={{ listStyle: "none", display: "flex", flexDirection: "column", gap: "0.6rem" }}>
                    {analysis.recommendations.map((r, i) => (
                      <li key={i} style={{
                        padding: "0.6rem 0.9rem",
                        borderRadius: 6,
                        background: "var(--surface-hover)",
                        border: "1px solid var(--border)",
                        fontSize: "0.9rem",
                        lineHeight: 1.5,
                      }}>
                        {r}
                      </li>
                    ))}
                  </ul>
                </section>
              </>
            )}
          </div>
        </div>
      )}

      {/* Holding Modal */}
      {showModal && (
        <div className="overlay" onClick={(e) => e.target === e.currentTarget && setShowModal(false)}>
          <div className="modal">
            <div className="modal-header">
              <h2>{editing ? "Edit Holding" : "Add Holding"}</h2>
              <button className="close-btn" onClick={() => setShowModal(false)}>×</button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="form-row">
                <div className="form-group">
                  <label>Asset Type *</label>
                  <select
                    value={form.asset_type}
                    onChange={(e) => setForm({ ...form, asset_type: e.target.value as AssetType, current_value: null })}
                    required
                  >
                    {(Object.keys(ASSET_LABELS) as AssetType[]).map((t) => (
                      <option key={t} value={t}>{ASSET_LABELS[t]}</option>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label>{isStock ? "Symbol" : form.asset_type === "fd" ? "Bank / Institution" : "Name"} *</label>
                  <input
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    placeholder={
                      isStock ? "e.g. RELIANCE" :
                      form.asset_type === "fd" ? "e.g. SBI" :
                      form.asset_type === "mutual_fund" ? "e.g. Mirae Asset Large Cap" :
                      form.asset_type === "gold" ? "e.g. Gold" : "Name"
                    }
                    required
                  />
                </div>
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>
                    {form.asset_type === "fd" ? "Principal Amount (₹)" :
                     form.asset_type === "mutual_fund" ? "NAV at Purchase (₹)" :
                     form.asset_type === "gold" ? "Buy Price / gram (₹)" :
                     "Buy Price (₹)"}
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={form.buy_price ?? ""}
                    onChange={(e) => setForm({ ...form, buy_price: e.target.value ? parseFloat(e.target.value) : null })}
                  />
                </div>
                <div className="form-group">
                  <label>
                    {form.asset_type === "fd" ? "Quantity (skip)" :
                     form.asset_type === "mutual_fund" ? "Units" :
                     form.asset_type === "gold" ? "Grams" :
                     "Quantity / Shares"}
                  </label>
                  <input
                    type="number"
                    step="0.0001"
                    min="0"
                    value={form.quantity ?? ""}
                    onChange={(e) => handleQuantityChange(e.target.value)}
                    disabled={form.asset_type === "fd"}
                  />
                </div>
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>
                    Current Value (₹)
                    {isStock && (
                      <span style={{ marginLeft: "0.4rem", fontSize: "0.78rem", color: priceFetching ? "#888" : priceError ? "#dc2626" : "#15803d" }}>
                        {priceFetching ? "fetching…" : priceError ? priceError : form.current_value != null ? "auto-fetched" : ""}
                      </span>
                    )}
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={form.current_value ?? ""}
                    onChange={(e) => setForm({ ...form, current_value: e.target.value ? parseFloat(e.target.value) : null })}
                    readOnly={isStock && !priceError && form.current_value != null}
                    style={isStock && !priceError && form.current_value != null ? { background: "#f6f9f6", color: "#555" } : undefined}
                  />
                </div>
                <div className="form-group">
                  <label>
                    {form.asset_type === "fd" ? "Start Date" : "Buy Date"}
                  </label>
                  <input
                    type="date"
                    value={form.buy_date ?? ""}
                    onChange={(e) => setForm({ ...form, buy_date: e.target.value })}
                  />
                </div>
              </div>

              <div className="form-group">
                <label>Notes</label>
                <textarea
                  value={form.notes}
                  onChange={(e) => setForm({ ...form, notes: e.target.value })}
                  placeholder={
                    form.asset_type === "fd" ? "e.g. 7.5% p.a., matures 2027-03-01" :
                    form.asset_type === "gold" ? "e.g. Sovereign Gold Bond series IV" : ""
                  }
                  rows={2}
                />
              </div>

              <div style={{ display: "flex", gap: "0.75rem", justifyContent: "flex-end", marginTop: "1.5rem" }}>
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary" disabled={saving || priceFetching}>
                  {saving ? "Saving…" : editing ? "Save Changes" : "Add Holding"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
