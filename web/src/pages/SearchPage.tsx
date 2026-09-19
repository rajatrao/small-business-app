import { Link, useSearchParams } from "react-router-dom";
import { useMutation, useQuery } from "@apollo/client";
import { RECORD_AFFINITY, SEARCH } from "../lib/ops";
import { PhotoStrip } from "../components/Bits";
import { useEffect } from "react";

export function SearchPage() {
  const [params] = useSearchParams();
  const q = params.get("q") || "";
  const { data, loading } = useQuery(SEARCH, { variables: { q, city: "Austin" }, skip: !q });
  const [record] = useMutation(RECORD_AFFINITY);

  useEffect(() => {
    const lower = q.toLowerCase();
    const cats = ["food", "coffee", "retail", "services", "health", "nightlife", "outdoor", "home"];
    const hit = cats.find((c) => lower.includes(c));
    if (hit) void record({ variables: { category: hit, kind: "SEARCH" } });
  }, [q, record]);

  const stores = data?.search?.stores || [];
  const products = data?.search?.products || [];

  return (
    <div style={{ marginTop: 20 }}>
      <p className="kicker">organic search · no ads in this build</p>
      <h1>Results for “{q}”</h1>
      {loading && <p className="empty">Looking around Austin…</p>}
      <h2>Shops</h2>
      {stores.map((s: any) => (
        <article className="row" key={s.id}>
          <div />
          <div />
          <div>
            <Link className="title" to={`/s/${s.slug}`}>
              {s.name}
            </Link>
            <p className="blurb">{s.description}</p>
            <div className="meta">
              {s.category} · {s.city}
            </div>
            <PhotoStrip urls={(s.photos || []).map((p: { url: string }) => p.url)} />
          </div>
        </article>
      ))}
      {!loading && stores.length === 0 && <p className="empty">No shops matched.</p>}
      <h2>Products</h2>
      <div className="cards">
        {products.map((p: any) => (
          <Link className="card" key={p.id} to={`/p/${p.slug}`}>
            {p.photos?.[0] && <img src={p.photos[0].url} alt="" />}
            <div className="pad">
              <div className="title">{p.name}</div>
              <div className="meta">{p.store?.name}</div>
              <div className="price">{p.displayPrice}</div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
