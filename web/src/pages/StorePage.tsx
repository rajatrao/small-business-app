import { Link, useNavigate, useOutletContext, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@apollo/client";
import { CREATE_REVIEW, RECORD_AFFINITY, STORE } from "../lib/ops";
import { loginPath, relTime } from "../lib/apollo";
import { ChatDrawer, ReviewForm, Stars, Upvote } from "../components/Bits";
import type { Me } from "../components/Layout";
import { useEffect } from "react";

type Ctx = { me: Me | null };

export function StorePage() {
  const { slug } = useParams();
  const { me } = useOutletContext<Ctx>();
  const nav = useNavigate();
  const { data, loading } = useQuery(STORE, { variables: { slug } });
  const [record] = useMutation(RECORD_AFFINITY);
  const [createReview] = useMutation(CREATE_REVIEW, { refetchQueries: ["Store"] });
  const s = data?.store;

  useEffect(() => {
    if (s?.category) void record({ variables: { category: s.category, kind: "VIEW_STORE" } });
  }, [s?.category, record]);

  if (loading) return <p className="empty">Opening the shop…</p>;
  if (!s) return <p className="empty">No shop at this slug.</p>;

  return (
    <div style={{ marginTop: 20 }}>
      <p className="kicker">
        <Link to="/">{s.city}</Link> · {s.category}
      </p>
      <div className="split">
        <div>
          <h1>{s.name}</h1>
          <p className="blurb">{s.description}</p>
          <div className="meta">
            {s.address} {s.phone && <> · <a href={`tel:${s.phone}`}>{s.phone}</a></>} · {relTime(s.createdAt)}
          </div>
          <div className="gallery">
            {(s.photos || []).map((p: { url: string; id: string }) => (
              <img key={p.id || p.url} src={p.url} alt="" />
            ))}
          </div>
          <h2>On the catalog</h2>
          <div className="cards">
            {s.products.map((p: any) => (
              <Link className="card" key={p.id} to={`/p/${p.slug}`}>
                {p.photos?.[0] && <img src={p.photos[0].url} alt="" />}
                <div className="pad">
                  <div className="title">{p.name}</div>
                  <div className="price">{p.displayPrice}</div>
                </div>
              </Link>
            ))}
          </div>
          <h2>Reviews</h2>
          <div className="reviews">
            {s.reviews.map((rv: any) => (
              <article className="review" key={rv.id}>
                <Stars n={rv.rating} />
                <p>{rv.body}</p>
                <div className="meta">
                  {rv.authorFirstName} · {relTime(rv.createdAt)}
                </div>
              </article>
            ))}
          </div>
          <h2>Write a review</h2>
          {me ? (
            <ReviewForm
              storeId={s.id}
              onSubmit={async (rating, body) => {
                await createReview({ variables: { storeId: s.id, rating, body } });
              }}
            />
          ) : (
            <p className="notice">
              Reviews are signed. <Link to={loginPath(`/s/${s.slug}`)}>Log in</Link> to write one.
            </p>
          )}
        </div>
        <aside>
          <Upvote count={s.upvoteCount} on={s.viewerHasVoted} storeId={s.id} authed={!!me} />
          <p className="meta">{s.upvoteCount} neighbors recommend this shop</p>
          {s.phone && (
            <p>
              <a className="btn primary" href={`tel:${s.phone}`}>
                Call {s.phone}
              </a>
            </p>
          )}
          <ChatDrawer storeId={s.id} phone={s.phone} title={s.name} />
          {!me && (
            <button className="btn ghost" onClick={() => nav(loginPath(`/s/${s.slug}`))}>
              Sign in to upvote
            </button>
          )}
        </aside>
      </div>
    </div>
  );
}
