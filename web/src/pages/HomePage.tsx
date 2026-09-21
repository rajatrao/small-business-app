import { Link, useOutletContext, useSearchParams } from "react-router-dom";
import { useMutation, useQuery } from "@apollo/client";
import { HOME, RECORD_AFFINITY } from "../lib/ops";
import { CATEGORIES, CITIES, pushFavCat, relTime } from "../lib/apollo";
import { PhotoStrip, Stars, Upvote } from "../components/Bits";
import type { Me } from "../components/Layout";

type Ctx = { me: Me | null };

export function HomePage() {
  const { me } = useOutletContext<Ctx>();
  const [params, setParams] = useSearchParams();
  const city = params.get("city") || "Austin";
  const category = params.get("cat") || "";
  const window = (params.get("w") || "ALL").toUpperCase();
  const { data, loading } = useQuery(HOME, { variables: { city, category: category || null, window } });
  const [record] = useMutation(RECORD_AFFINITY);
  const home = data?.home;
  const rankedStores = (home?.rankedStores || []).filter(
    (row: { store?: { city?: string } }) => !city || row.store?.city?.toLowerCase() === city.toLowerCase(),
  );
  const trendingProducts = (home?.trendingProducts || []).filter(
    (row: { product?: { store?: { city?: string } } }) =>
      !city || row.product?.store?.city?.toLowerCase() === city.toLowerCase(),
  );
  const trendingReviews = (home?.trendingReviews || []).filter(
    (row: { review?: { store?: { city?: string } } }) =>
      !city || !row.review?.store?.city || row.review.store.city.toLowerCase() === city.toLowerCase(),
  );

  function set(k: string, v: string) {
    const next = new URLSearchParams(params);
    if (v) next.set(k, v);
    else next.delete(k);
    setParams(next);
  }

  return (
    <>
      <p className="kicker">the public square for neighborhood shops · browse free, write when you sign in</p>
      <h1>What’s buzzing in {city}</h1>
      <div className="toolbar">
        <div className="chips">
          <button className={`chip ${!category ? "on" : ""}`} onClick={() => set("cat", "")}>
            all
          </button>
          {CATEGORIES.map((c) => (
            <button
              key={c}
              className={`chip ${category === c ? "on" : ""}`}
              onClick={() => {
                set("cat", c);
                pushFavCat(c);
                void record({ variables: { category: c, kind: "VIEW_CATEGORY" } });
              }}
            >
              {c}
            </button>
          ))}
        </div>
        <div className="windows">
          {(["TODAY", "WEEK", "ALL"] as const).map((w) => (
            <button key={w} className={`win ${window === w ? "on" : ""}`} onClick={() => set("w", w)}>
              {w.toLowerCase()}
            </button>
          ))}
          <select value={city} onChange={(e) => set("city", e.target.value)} aria-label="City">
            {CITIES.map((c) => (
              <option key={c}>{c}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="home-layout">
        <section className="home-main">
          <h2>Ranked local businesses</h2>
          {loading && <p className="empty">Ranking the neighborhood…</p>}
          {!loading && rankedStores.length === 0 && (
            <p className="empty">No shops listed in {city} yet.</p>
          )}
          {rankedStores.map((row: any) => (
            <article className="row" key={row.store.id}>
              <div className="rank">{row.rank}.</div>
              <Upvote count={row.store.upvoteCount} on={row.store.viewerHasVoted} storeId={row.store.id} authed={!!me} />
              <div>
                <Link className="title" to={`/s/${row.store.slug}`}>
                  {row.store.name}
                </Link>
                <p className="blurb">{row.store.description}</p>
                <div className="meta">
                  <Link to={`/?cat=${row.store.category}&city=${encodeURIComponent(city)}`}>{row.store.category}</Link>
                  {" · "}
                  {relTime(row.store.createdAt)}
                  {" · "}
                  score {row.communityScore.toFixed(2)}
                </div>
                <PhotoStrip urls={(row.store.photos || []).map((p: { url: string }) => p.url)} />
              </div>
              <div className="city-col">{row.store.city}</div>
            </article>
          ))}
        </section>

        <aside className="home-rail">
          <h2>Trending products</h2>
          <div className="cards cards-rail">
            {trendingProducts.map((row: any) => (
              <Link className="card" key={row.product.id} to={`/p/${row.product.slug}`}>
                {row.product.photos?.[0] && <img src={row.product.photos[0].url} alt="" />}
                <div className="pad">
                  <div className="title">{row.product.name}</div>
                  <div className="meta">{row.product.store?.name}</div>
                  <div className="price">{row.product.displayPrice}</div>
                </div>
              </Link>
            ))}
          </div>
        </aside>

        <section className="home-reviews">
          <h2>Word on the street</h2>
          <div className="reviews">
            {trendingReviews.map((row: any) => (
              <article className="review" key={row.review.id}>
                <Stars n={row.review.rating} />
                <p>{row.review.body}</p>
                <div className="meta">
                  {row.review.authorFirstName}
                  {" · "}
                  {row.review.store ? <Link to={`/s/${row.review.store.slug}`}>{row.review.store.name}</Link> : null}
                  {row.review.product ? (
                    <>
                      {" · "}
                      <Link to={`/p/${row.review.product.slug}`}>{row.review.product.name}</Link>
                    </>
                  ) : null}
                  {" · "}
                  {relTime(row.review.createdAt)}
                </div>
              </article>
            ))}
          </div>
        </section>
      </div>
    </>
  );
}
