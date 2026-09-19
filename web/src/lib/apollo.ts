import { ApolloClient, InMemoryCache, createHttpLink } from "@apollo/client";

const uri = `${import.meta.env.VITE_API_URL || "http://localhost:8080"}/graphql`;

export const apollo = new ApolloClient({
  link: createHttpLink({ uri, credentials: "include" }),
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: { fetchPolicy: "cache-and-network" },
  },
});

export const CATEGORIES = [
  "food",
  "coffee",
  "retail",
  "services",
  "health",
  "nightlife",
  "outdoor",
  "home",
] as const;

export type Category = (typeof CATEGORIES)[number];

export const CITIES = [
  "Austin",
  "San Francisco",
  "New York",
  "Los Angeles",
  "Chicago",
  "Houston",
  "Miami",
  "Seattle",
  "San Diego",
  "San Jose",
] as const;

export function apiURL(path: string) {
  const base = import.meta.env.VITE_API_URL || "http://localhost:8080";
  return `${base}${path}`;
}

export function loginPath(next = "/") {
  return `/login?next=${encodeURIComponent(next)}`;
}

export function favCats(): string[] {
  const m = document.cookie.match(/(?:^|; )fav_cats=([^;]*)/);
  if (!m) return [];
  return decodeURIComponent(m[1])
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

export function pushFavCat(cat: string) {
  const next = [cat, ...favCats().filter((c) => c !== cat)].slice(0, 8);
  document.cookie = `fav_cats=${encodeURIComponent(next.join(","))};path=/;max-age=${60 * 60 * 24 * 365}`;
}

export function relTime(iso: string) {
  const then = new Date(iso).getTime();
  const h = Math.max(0, (Date.now() - then) / 36e5);
  if (h < 1) return `${Math.max(1, Math.round(h * 60))}m ago`;
  if (h < 24) return `${Math.round(h)}h ago`;
  const d = Math.round(h / 24);
  return d === 1 ? "1 day ago" : `${d} days ago`;
}
