import { FormEvent, useState } from "react";
import { Link, Navigate, useOutletContext } from "react-router-dom";
import { useMutation, useQuery } from "@apollo/client";
import { BECOME_OWNER, CREATE_PRODUCT, CREATE_STORE, DELETE_STORE, MY_STORES, UPDATE_STORE, UPLOAD_PHOTO } from "../lib/ops";
import { CATEGORIES, CITIES } from "../lib/apollo";
import type { Me } from "../components/Layout";

type Ctx = { me: Me | null; refetchMe: () => Promise<unknown> };

export function DashboardPage() {
  const { me, refetchMe } = useOutletContext<Ctx>();
  const { data, refetch, loading } = useQuery(MY_STORES, { skip: !me || (me.role !== "OWNER" && me.role !== "ADMIN") });
  const [become] = useMutation(BECOME_OWNER);
  const [createStore] = useMutation(CREATE_STORE);
  const [updateStore] = useMutation(UPDATE_STORE);
  const [deleteStore] = useMutation(DELETE_STORE);
  const [createProduct] = useMutation(CREATE_PRODUCT);
  const [uploadPhoto] = useMutation(UPLOAD_PHOTO);
  const [err, setErr] = useState("");
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);
  const stores = data?.myStores || [];

  if (!me) return <Navigate to="/login?next=/dashboard" replace />;

  if (me.role !== "OWNER" && me.role !== "ADMIN") {
    return (
      <div style={{ marginTop: 28 }}>
        <h1>Owner tools are behind the desk</h1>
        <p>You’re signed in as a neighbor. Promote this same account to list a shop — no second login.</p>
        <button
          className="btn primary"
          onClick={async () => {
            await become();
            await refetchMe();
          }}
        >
          Become a business owner
        </button>
      </div>
    );
  }

  async function onCreateStore(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErr("");
    const fd = new FormData(e.currentTarget);
    try {
      await createStore({
        variables: {
          input: {
            name: String(fd.get("name")),
            description: String(fd.get("description")),
            category: String(fd.get("category")),
            city: String(fd.get("city") || "Austin"),
            phone: String(fd.get("phone") || "") || null,
            address: String(fd.get("address") || "") || null,
          },
        },
      });
      e.currentTarget.reset();
      await refetch();
    } catch (ex: unknown) {
      setErr(ex instanceof Error ? ex.message : "Could not create store");
    }
  }

  async function fileToB64(file: File) {
    const buf = await file.arrayBuffer();
    const bytes = new Uint8Array(buf);
    let bin = "";
    for (const b of bytes) bin += String.fromCharCode(b);
    return btoa(bin);
  }

  return (
    <div style={{ marginTop: 20 }}>
      <p className="kicker">owner dashboard · display prices only, no checkout</p>
      <h1>Your storefronts</h1>
      {err && <div className="err">{err}</div>}
      {loading && <p className="empty">Loading listings…</p>}
      {stores.map((s: any) => (
        <section key={s.id} className="review" style={{ marginBottom: 16 }}>
          <h2>
            <Link to={`/s/${s.slug}`}>{s.name}</Link>
          </h2>
          <p className="meta">
            {s.category} · {s.city} · {s.phone}
          </p>
          <form
            className="form wide"
            onSubmit={async (e) => {
              e.preventDefault();
              const fd = new FormData(e.currentTarget);
              await updateStore({
                variables: {
                  id: s.id,
                  input: {
                    name: String(fd.get("name")),
                    description: String(fd.get("description")),
                    city: String(fd.get("city")),
                    phone: String(fd.get("phone")),
                    address: String(fd.get("address")),
                  },
                },
              });
              await refetch();
            }}
          >
            <label>
              Name
              <input name="name" defaultValue={s.name} />
            </label>
            <label>
              Description
              <textarea name="description" defaultValue={s.description} rows={3} />
            </label>
            <label>
              City
              <select name="city" defaultValue={s.city || "Austin"}>
                {CITIES.map((c) => (
                  <option key={c}>{c}</option>
                ))}
              </select>
            </label>
            <label>
              Phone
              <input name="phone" defaultValue={s.phone || ""} />
            </label>
            <label>
              Address
              <input name="address" defaultValue={s.address || ""} />
            </label>
            <div className="chips" style={{ marginTop: 4 }}>
              <button className="btn" type="submit">
                Save shop
              </button>
              {pendingDelete === s.id ? (
                <>
                  <button
                    className="btn danger"
                    type="button"
                    onClick={async () => {
                      setErr("");
                      try {
                        await deleteStore({ variables: { id: s.id } });
                        setPendingDelete(null);
                        await refetch();
                      } catch (ex: unknown) {
                        setErr(ex instanceof Error ? ex.message : "Could not delete store");
                      }
                    }}
                  >
                    Confirm delete
                  </button>
                  <button className="btn ghost" type="button" onClick={() => setPendingDelete(null)}>
                    Cancel
                  </button>
                </>
              ) : (
                <button className="btn danger" type="button" onClick={() => setPendingDelete(s.id)}>
                  Delete shop
                </button>
              )}
            </div>
          </form>
          <label>
            Add photo
            <input
              type="file"
              accept="image/*"
              onChange={async (e) => {
                const file = e.target.files?.[0];
                if (!file) return;
                await uploadPhoto({
                  variables: { storeId: s.id, filename: file.name, contentBase64: await fileToB64(file) },
                });
                await refetch();
              }}
            />
          </label>
          <h2>Products</h2>
          <ul>
            {s.products.map((p: any) => (
              <li key={p.id}>
                <Link to={`/p/${p.slug}`}>{p.name}</Link> · {p.displayPrice}
              </li>
            ))}
          </ul>
          <form
            className="form"
            onSubmit={async (e) => {
              e.preventDefault();
              const fd = new FormData(e.currentTarget);
              await createProduct({
                variables: {
                  storeId: s.id,
                  input: {
                    name: String(fd.get("name")),
                    description: String(fd.get("description")),
                    priceCents: Math.round(Number(fd.get("price")) * 100),
                  },
                },
              });
              e.currentTarget.reset();
              await refetch();
            }}
          >
            <label>
              Product name
              <input name="name" required />
            </label>
            <label>
              Description
              <input name="description" />
            </label>
            <label>
              Display price (USD)
              <input name="price" type="number" step="0.01" min="0" required />
            </label>
            <button className="btn primary" type="submit">
              Add product
            </button>
          </form>
        </section>
      ))}

      <h2>{stores.length ? "Open another shop" : "Create your first shop"}</h2>
      <form className="form wide" onSubmit={onCreateStore}>
        <label>
          Shop name
          <input name="name" required />
        </label>
        <label>
          Category
          <select name="category" defaultValue="food">
            {CATEGORIES.map((c) => (
              <option key={c}>{c}</option>
            ))}
          </select>
        </label>
        <label>
          City
          <select name="city" defaultValue="Austin">
            {CITIES.map((c) => (
              <option key={c}>{c}</option>
            ))}
          </select>
        </label>
        <label>
          Description
          <textarea name="description" required rows={3} />
        </label>
        <label>
          Phone
          <input name="phone" placeholder="512-555-0100" />
        </label>
        <label>
          Address
          <input name="address" />
        </label>
        <button className="btn primary" type="submit">
          Publish shop
        </button>
      </form>
    </div>
  );
}

export function NewStorePage() {
  return <DashboardPage />;
}
