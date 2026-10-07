import { useState, type FormEvent } from "react";
import { createShortlist, markUsed, previewUrl, PLATFORMS, type Platform, type Track } from "./api";

const EXAMPLE = "Calm acoustic background for a cooking tutorial, about two minutes. I talk over it.";

export default function App() {
  const [brief, setBrief] = useState("");
  const [tracks, setTracks] = useState<Track[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [previews, setPreviews] = useState<Record<string, string>>({});

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      setTracks(await createShortlist(brief));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function onUse(id: string, platform: Platform) {
    try {
      await markUsed(id, platform);
      setTracks((ts) => ts.map((t) => (t.id === id ? { ...t, used: true } : t)));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function onPreview(id: string) {
    try {
      const url = await previewUrl(id);
      setPreviews((p) => ({ ...p, [id]: url }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <main>
      <p className="eyebrow">Epidemic Sound</p>
      <h1>Soundtrack Agent</h1>
      <p className="muted">Describe your video and get a shortlist of tracks from the catalogue.</p>

      <form className="card" onSubmit={onSubmit}>
        <label className="eyebrow" htmlFor="brief">
          Your video
        </label>
        <textarea id="brief" value={brief} onChange={(e) => setBrief(e.target.value)} placeholder={EXAMPLE} rows={4} />
        <button className="btn btn--primary" type="submit" disabled={loading || !brief.trim()}>
          {loading ? "Finding tracks…" : "Find tracks"}
        </button>
      </form>

      {error && <p className="error">{error}</p>}

      {tracks.length > 0 && (
        <section>
          <h2>Shortlist</h2>
          <ol className="tracks">
            {tracks.map((t) => (
              <li key={t.id}>
                <div className="track">
                  <strong>{t.title}</strong> <span className="muted">by {t.artist}</span>
                  <p>{t.reason}</p>
                  {previews[t.id] ? (
                    <audio src={previews[t.id]} controls autoPlay />
                  ) : (
                    <button className="btn btn--ghost" onClick={() => onPreview(t.id)}>
                      ▶ Play preview
                    </button>
                  )}
                </div>
                {t.used ? (
                  <span className="used">Used ✓</span>
                ) : (
                  <select className="btn btn--secondary" value="" onChange={(e) => onUse(t.id, e.target.value as Platform)}>
                    <option value="" disabled>
                      Mark as used on…
                    </option>
                    {Object.entries(PLATFORMS).map(([value, label]) => (
                      <option key={value} value={value}>
                        {label}
                      </option>
                    ))}
                  </select>
                )}
              </li>
            ))}
          </ol>
        </section>
      )}
    </main>
  );
}
