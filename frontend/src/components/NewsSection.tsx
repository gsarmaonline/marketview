"use client";

export interface NewsItem {
  title: string;
  description: string;
  link: string;
  publishedAt: string;
  source: string;
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

export function NewsSection({
  news,
  loading,
  error,
}: {
  news: NewsItem[];
  loading: boolean;
  error: string | null;
}) {
  return (
    <section>
      <h2 style={{
        fontSize: "18px",
        fontWeight: 700,
        marginBottom: "16px",
        color: "var(--text)",
      }}>
        Market News
      </h2>

      {loading && (
        <div style={{ color: "var(--text-muted)", padding: "24px 0" }}>
          Loading news...
        </div>
      )}

      {error && !loading && (
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
      )}

      {!loading && !error && news.length === 0 && (
        <div style={{ color: "var(--text-muted)", padding: "24px 0" }}>
          No news available.
        </div>
      )}

      <div style={{ display: "flex", flexDirection: "column", gap: "10px" }}>
        {news.map((item, i) => (
          <NewsCard key={`${item.link}-${i}`} item={item} />
        ))}
      </div>
    </section>
  );
}
