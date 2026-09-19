import { FormEvent } from "react";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useMutation, useQuery } from "@apollo/client";
import { LOGOUT, ME } from "../lib/ops";

export type Me = { id: string; email: string; name: string; role: "CUSTOMER" | "OWNER" | "ADMIN" };

export function Layout() {
  const nav = useNavigate();
  const loc = useLocation();
  const { data, refetch } = useQuery(ME);
  const [logout] = useMutation(LOGOUT);
  const me: Me | null = data?.me ?? null;
  const q = new URLSearchParams(loc.search).get("q") ?? "";

  async function onSearch(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const query = String(fd.get("q") || "").trim();
    nav(query ? `/search?q=${encodeURIComponent(query)}` : "/");
  }

  return (
    <div className="shell">
      <header className="topbar">
        <Link className="brand" to="/">
          local<span>discovery</span>
        </Link>
        <form className="search" onSubmit={onSearch}>
          <input name="q" defaultValue={q} placeholder="Search Austin shops and products" />
        </form>
        <nav className="nav">
          {me?.role === "OWNER" || me?.role === "ADMIN" ? (
            <Link to="/dashboard">Dashboard</Link>
          ) : me ? (
            <Link to="/signup?role=owner">List a shop</Link>
          ) : (
            <Link to="/signup?role=owner">For owners</Link>
          )}
          {me ? (
            <>
              <span className="who">
                {me.name.split(" ")[0]} · {me.role.toLowerCase()}
              </span>
              <button
                className="btn ghost"
                onClick={async () => {
                  await logout();
                  await refetch();
                  nav("/");
                }}
              >
                Log out
              </button>
            </>
          ) : (
            <Link to="/login">Log in</Link>
          )}
        </nav>
      </header>
      <Outlet context={{ me, oidcLabel: data?.oidcLabel as string | undefined, refetchMe: refetch }} />
    </div>
  );
}
