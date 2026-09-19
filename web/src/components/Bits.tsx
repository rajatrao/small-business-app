import { useState } from "react";
import { useMutation } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { CHAT, VOTE } from "../lib/ops";
import { loginPath } from "../lib/apollo";

export function Upvote({
  count,
  on,
  storeId,
  productId,
  authed,
}: {
  count: number;
  on: boolean;
  storeId?: string;
  productId?: string;
  authed: boolean;
}) {
  const nav = useNavigate();
  const [vote, { loading }] = useMutation(VOTE, { refetchQueries: "active" });
  return (
    <button
      className={`vote ${on ? "on" : ""}`}
      disabled={loading}
      aria-label="upvote"
      onClick={() => {
        if (!authed) {
          nav(loginPath(window.location.pathname));
          return;
        }
        void vote({ variables: { storeId, productId } });
      }}
    >
      <svg viewBox="0 0 20 20" fill="currentColor" aria-hidden>
        <path d="M10 3.2 16.5 12H3.5L10 3.2Z" />
      </svg>
      <b>{count}</b>
    </button>
  );
}

export function PhotoStrip({ urls }: { urls: string[] }) {
  if (!urls.length) return null;
  return (
    <div className="strip">
      {urls.map((u) => (
        <img key={u} src={u} alt="" />
      ))}
    </div>
  );
}

export function Stars({ n }: { n: number }) {
  return <span className="stars">{"★".repeat(n) + "☆".repeat(Math.max(0, 5 - n))}</span>;
}

export function ReviewForm({
  onSubmit,
}: {
  storeId?: string;
  productId?: string;
  onSubmit: (rating: number, body: string) => Promise<void>;
}) {
  return (
    <form
      className="form"
      onSubmit={(e) => {
        e.preventDefault();
        const fd = new FormData(e.currentTarget);
        void onSubmit(Number(fd.get("rating")), String(fd.get("body")));
        e.currentTarget.reset();
      }}
    >
      <label>
        Stars
        <select name="rating" defaultValue="5">
          {[5, 4, 3, 2, 1].map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </select>
      </label>
      <label>
        What should neighbors know?
        <textarea name="body" required rows={3} />
      </label>
      <button className="btn primary" type="submit">
        Post review
      </button>
    </form>
  );
}

export function ChatDrawer({
  storeId,
  productId,
  phone,
  title,
}: {
  storeId?: string;
  productId?: string;
  phone?: string | null;
  title: string;
}) {
  const [open, setOpen] = useState(false);
  const [threadId, setThreadId] = useState<string | undefined>();
  const [msgs, setMsgs] = useState<{ me: boolean; text: string }[]>([]);
  const [chat] = useMutation(CHAT);
  return (
    <>
      <button className="btn primary" onClick={() => setOpen(true)}>
        Ask this listing
      </button>
      {open && (
        <>
          <div className="drawer-backdrop" onClick={() => setOpen(false)} />
          <aside className="drawer">
            <header>
              <strong>Catalog chat · {title}</strong>
              <button className="btn ghost" onClick={() => setOpen(false)}>
                Close
              </button>
            </header>
            <div className="msgs">
              {msgs.length === 0 && (
                <div className="bubble">
                  Ask about listed prices, address, or what’s on the catalog. I won’t invent hours or deals.
                </div>
              )}
              {msgs.map((m, i) => (
                <div key={i} className={`bubble ${m.me ? "me" : ""}`}>
                  {m.text}
                </div>
              ))}
            </div>
            {phone && (
              <a className="btn primary" style={{ margin: "0 12px" }} href={`tel:${phone}`}>
                Call {phone}
              </a>
            )}
            <form
              className="composer"
              onSubmit={async (e) => {
                e.preventDefault();
                const fd = new FormData(e.currentTarget);
                const message = String(fd.get("message") || "").trim();
                if (!message) return;
                e.currentTarget.reset();
                setMsgs((m) => [...m, { me: true, text: message }]);
                const res = await chat({ variables: { storeId, productId, threadId, message } });
                const payload = res.data?.chat;
                if (payload) {
                  setThreadId(payload.threadId);
                  setMsgs((m) => [...m, { me: false, text: payload.message }]);
                }
              }}
            >
              <input name="message" placeholder="How much is the brisket plate?" />
              <button className="btn" type="submit">
                Send
              </button>
            </form>
          </aside>
        </>
      )}
    </>
  );
}
