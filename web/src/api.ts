import { getUserId } from "./user";

export type Track = {
  id: string;
  title: string;
  artist: string;
  reason: string;
  used: boolean;
};

export const PLATFORMS = {
  YOUTUBE: "YouTube",
  TIKTOK: "TikTok",
  INSTAGRAM: "Instagram",
  OTHER: "Somewhere else",
};
export type Platform = keyof typeof PLATFORMS;

async function request<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: body === undefined ? "GET" : "POST",
    headers: {
      "Content-Type": "application/json",
      "X-User-ID": getUserId(),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error ?? `Request failed (${res.status})`);
  }
  return data as T;
}

export async function createShortlist(brief: string): Promise<Track[]> {
  const data = await request<{ tracks: Track[] }>("/api/shortlist", { brief });
  return data.tracks;
}

export async function markUsed(trackId: string, platform: Platform): Promise<void> {
  await request("/api/usage", { trackId, platform });
}

export async function previewUrl(trackId: string): Promise<string> {
  const data = await request<{ url: string }>(`/api/tracks/${encodeURIComponent(trackId)}/preview`);
  return data.url;
}
