import { FormEvent, useState } from "react";
import { Link, useNavigate, useOutletContext, useSearchParams } from "react-router-dom";
import { useMutation } from "@apollo/client";
import { LOGIN, ME, SIGNUP } from "../lib/ops";
import { apiURL } from "../lib/apollo";
import type { Me } from "../components/Layout";

type Ctx = { me: Me | null; oidcLabel?: string; refetchMe: () => Promise<unknown> };

export function LoginPage() {
  return <AuthForm mode="login" />;
}

export function SignupPage() {
  return <AuthForm mode="signup" />;
}

function AuthForm({ mode }: { mode: "login" | "signup" }) {
  const { oidcLabel, refetchMe } = useOutletContext<Ctx>();
  const [params] = useSearchParams();
  const nav = useNavigate();
  const next = params.get("next") || "/";
  const isOwner = params.get("role") === "owner";
  const [err, setErr] = useState("");
  const [login] = useMutation(LOGIN);
  const [signup] = useMutation(SIGNUP);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErr("");
    const fd = new FormData(e.currentTarget);
    const email = String(fd.get("email"));
    const password = String(fd.get("password"));
    const name = String(fd.get("name") || "");
    try {
      if (mode === "signup") {
        const res = await signup({
          variables: { email, password, name, role: isOwner ? "OWNER" : "CUSTOMER" },
          refetchQueries: [{ query: ME }],
        });
        const role = res.data?.signup?.role;
        await refetchMe();
        nav(role === "OWNER" ? "/dashboard" : next);
      } else {
        const res = await login({ variables: { email, password }, refetchQueries: [{ query: ME }] });
        await refetchMe();
        nav(res.data?.login?.role === "OWNER" && next === "/" ? "/dashboard" : next);
      }
    } catch (ex: unknown) {
      setErr(ex instanceof Error ? ex.message.replace(/^GraphQL error: /, "") : "Could not authenticate");
    }
  }

  const oidcHref = apiURL(`/auth/oidc/start?role=${isOwner ? "OWNER" : "CUSTOMER"}&next=${encodeURIComponent(next)}`);

  return (
    <div className="split" style={{ marginTop: 28 }}>
      <div>
        <p className="kicker">{isOwner ? "owner desk" : "neighbors"}</p>
        <h1>{mode === "signup" ? (isOwner ? "List your shop" : "Join the square") : isOwner ? "Owner sign in" : "Welcome back"}</h1>
        <p className="blurb">
          {isOwner
            ? "Owners create listings, photos, and display prices. No ads or checkout in this build — just discovery."
            : "Browse is public. Sign in only when you want to upvote or write a review."}
        </p>
        <form className="form" onSubmit={onSubmit}>
          {mode === "signup" && (
            <label>
              Name
              <input name="name" required placeholder={isOwner ? "Owen Hart" : "Maya Navarro"} />
            </label>
          )}
          <label>
            Email
            <input name="email" type="email" required placeholder={isOwner ? "owner@demo.local" : "customer@demo.local"} />
          </label>
          <label>
            Password
            <input name="password" type="password" required minLength={8} placeholder="demo1234" />
          </label>
          {err && <div className="err">{err}</div>}
          <button className="btn primary" type="submit">
            {mode === "signup" ? "Create account" : "Sign in"}
          </button>
        </form>
        <p style={{ marginTop: 16 }}>
          <a className="btn" href={oidcHref}>
            {oidcLabel || "Continue with OpenID"}
          </a>
        </p>
        <p className="meta">
          {mode === "login" ? (
            <>
              New here? <Link to={`/signup${isOwner ? "?role=owner" : ""}`}>Create an account</Link>
            </>
          ) : (
            <>
              Already on the square? <Link to={`/login${isOwner ? "?role=owner" : ""}`}>Log in</Link>
            </>
          )}
        </p>
        <p className="meta">Demo: customer@demo.local / owner@demo.local · password demo1234</p>
      </div>
      <aside className="review">
        {isOwner ? (
          <>
            <h2>For owners</h2>
            <p>Claim a storefront, hang photos, list products with display prices, and let the neighborhood find you.</p>
          </>
        ) : (
          <>
            <h2>For neighbors</h2>
            <p>Upvote shops that still feel like Austin. Reviews are attributed to first name only.</p>
          </>
        )}
      </aside>
    </div>
  );
}
