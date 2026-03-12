"use client";

export interface IndicatorResult {
  name: string;
  value: number;
  unit: string;
  signal: "bullish" | "neutral" | "bearish";
  description: string;
}

const signalStyles: Record<
  IndicatorResult["signal"],
  { label: string; color: string; bg: string; icon: string }
> = {
  bullish: { label: "Bullish", color: "var(--bullish)", bg: "var(--bullish-bg)", icon: "↑" },
  bearish: { label: "Bearish", color: "var(--bearish)", bg: "var(--bearish-bg)", icon: "↓" },
  neutral: { label: "Neutral", color: "var(--neutral)", bg: "var(--neutral-bg)", icon: "→" },
};

export function IndicatorCard({ indicator }: { indicator: IndicatorResult }) {
  const s = signalStyles[indicator.signal];

  return (
    <div style={{
      background: "var(--surface)",
      border: "1px solid var(--border)",
      borderRadius: "var(--radius)",
      padding: "20px 24px",
      display: "flex",
      flexDirection: "column",
      gap: "10px",
    }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
        <span style={{ color: "var(--text-muted)", fontSize: "13px", fontWeight: 500 }}>
          {indicator.name}
        </span>
        <span style={{
          color: s.color,
          background: s.bg,
          border: `1px solid ${s.color}33`,
          borderRadius: "6px",
          padding: "2px 10px",
          fontSize: "12px",
          fontWeight: 600,
          letterSpacing: "0.03em",
          display: "flex",
          alignItems: "center",
          gap: "4px",
        }}>
          <span>{s.icon}</span>
          {s.label}
        </span>
      </div>

      <div style={{ display: "flex", alignItems: "baseline", gap: "4px" }}>
        <span style={{ fontSize: "32px", fontWeight: 700, letterSpacing: "-0.02em" }}>
          {indicator.value.toFixed(2)}
        </span>
        <span style={{ color: "var(--text-muted)", fontSize: "16px" }}>
          {indicator.unit}
        </span>
      </div>

      <p style={{ color: "var(--text-muted)", fontSize: "13px", lineHeight: 1.5 }}>
        {indicator.description}
      </p>
    </div>
  );
}
