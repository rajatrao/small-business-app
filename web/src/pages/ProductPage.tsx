import { Link, useOutletContext, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@apollo/client";
import { CREATE_REVIEW, PRODUCT, RECORD_AFFINITY } from "../lib/ops";
import { loginPath, relTime } from "../lib/apollo";
import { ChatDrawer, ReviewForm, Stars, Upvote } from "../components/Bits";
import type { Me } from "../components/Layout";
import { useEffect } from "react";

type Ctx = { me: Me | null };

export function ProductPage() {
  const { slug } = useParams();
  const { me } = useOutletContext<Ctx>();
  const { data, loading } = useQuery(PRODUCT, { variables: { slug } });
  const [record] = useMutation(RECORD_AFFINITY);
  const [createReview] = useMutation(CREATE_REVIEW, { refetchQueries: ["Product"] });
  const p = data?.product;

  useEffect(() => {
    if (p?.store?.category) void record({ variables: { category: p.store.category, kind: "VIEW_PRODUCT" } });
  }, [p?.store?.category, record]);

  if (loading) return <p className="empty">Pulling the listing…</p>;
  if (!p) return <p className="empty">No product at this slug.</p>;

  return (
    <div style={{ marginTop: 20 }}>
      <p className="kicker">
        <Link to={`/s/${p.store.slug}`}>{p.store.name}</Link> · {p.store.category} · display price
      </p>
      <div className="split">
        <div>
          <h1>{p.name}</h1>
          <p className="price" style={{ fontSize: "1.2rem" }}>
            {p.displayPrice}
          </p>
          <p className="blurb">{p.description}</p>
          <div className="gallery">
            {(p.photos || []).map((ph: { url: string }) => (
              <img key={ph.url} src={ph.url} alt="" />
            ))}
          </div>
          <h2>Reviews</h2>
          <div className="reviews">
            {p.reviews.map((rv: any) => (
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
              productId={p.id}
              onSubmit={async (rating, body) => {
                await createReview({ variables: { productId: p.id, rating, body } });
              }}
            />
          ) : (
            <p className="notice">
              <Link to={loginPath(`/p/${p.slug}`)}>Log in</Link> to review this product.
            </p>
          )}
        </div>
        <aside>
          <Upvote count={p.upvoteCount} on={p.viewerHasVoted} productId={p.id} authed={!!me} />
          {p.store.phone && (
            <p>
              <a className="btn primary" href={`tel:${p.store.phone}`}>
                Call {p.store.name}
              </a>
            </p>
          )}
          <ChatDrawer storeId={p.store.id} productId={p.id} phone={p.store.phone} title={p.name} />
        </aside>
      </div>
    </div>
  );
}
