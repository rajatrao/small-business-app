package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajat/localdiscovery/internal/adapter/postgres"
	"github.com/rajat/localdiscovery/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations and exit")
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal("connect: %v", err)
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool); err != nil {
		fatal("migrate: %v", err)
	}
	if *migrateOnly {
		fmt.Println("migrations applied")
		return
	}
	if err := seed(ctx, pool); err != nil {
		fatal("seed: %v", err)
	}
	fmt.Println("seeded Austin catalog + demo users (customer@demo.local / owner@demo.local, password demo1234)")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM stores`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		fmt.Println("stores already present; skip seed")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hs := string(hash)
	now := time.Now().UTC()

	type user struct {
		id, email, name, role string
	}
	users := []user{
		{uuid.NewString(), "customer@demo.local", "Maya Navarro", "CUSTOMER"},
		{uuid.NewString(), "owner@demo.local", "Owen Hart", "OWNER"},
	}
	for i := 1; i <= 40; i++ {
		first := []string{"Priya", "Luis", "Asha", "Ken", "Nina", "Omar", "Jules", "Rae", "Sam", "Ivy"}[(i-1)%10]
		users = append(users, user{
			uuid.NewString(),
			fmt.Sprintf("voter%d@demo.local", i),
			fmt.Sprintf("%s %c.", first, 'A'+(i%26)),
			"CUSTOMER",
		})
	}
	for _, u := range users {
		if _, err := pool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, name, role, created_at)
			VALUES ($1,$2,$3,$4,$5,$6)`, u.id, u.email, hs, u.name, u.role, now.Add(-20*24*time.Hour)); err != nil {
			return err
		}
	}
	ownerID := users[1].id
	customerID := users[0].id

	type prod struct {
		name, desc string
		cents      int
	}
	type store struct {
		name, slug, desc, cat, city, phone, addr string
		hoursAgo                                 int
		photos                             []string
		products                           []prod
		reviews                            []struct {
			who, body string
			stars     int
			hoursAgo  int
		}
	}

	catalog := []store{
		{
			name: "Franklin Smokehouse", slug: "franklin-smokehouse", cat: "food",
			desc: "Oak-smoked brisket that still sells out. Line forms early; pecan pie is the closer.",
			phone: "512-555-0140", addr: "900 E 11th St, Austin, TX", hoursAgo: 36,
			photos: []string{"franklin-1", "franklin-2", "franklin-3"},
			products: []prod{{"Brisket plate", "Half pound with pickles and onions", 1899}, {"Pecan pie slice", "Toasted pecan, still warm", 650}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: customerID, body: "Worth the wait. Bark on the brisket is textbook.", stars: 5, hoursAgo: 5}},
		},
		{
			name: "Houndstooth Coffee", slug: "houndstooth-coffee", cat: "coffee",
			desc: "South Congress espresso bar. Single-origin pours and a patio that catches late light.",
			phone: "512-555-0188", addr: "401 Congress Ave, Austin, TX", hoursAgo: 40,
			photos: []string{"hound-1", "hound-2"},
			products: []prod{{"Cortado", "Equal parts espresso and steamed milk", 450}, {"Cardamom bun", "Morning pastry, limited", 425}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[2].id, body: "Cortado is the move. Patio is the office.", stars: 5, hoursAgo: 8}},
		},
		{
			name: "BookPeople", slug: "bookpeople", cat: "retail",
			desc: "Independent bookstore stacked to the rafters. Staff picks you actually want to read.",
			phone: "512-555-0112", addr: "603 N Lamar Blvd, Austin, TX", hoursAgo: 72,
			photos: []string{"books-1", "books-2", "books-3"},
			products: []prod{{"Staff pick hardcover", "This week's fiction rec", 2800}, {"Austin tote", "Canvas, local print", 1800}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[3].id, body: "Best browsing in town. Kids' room is a gift.", stars: 5, hoursAgo: 12}},
		},
		{
			name: "Barton Springs Bikes", slug: "barton-springs-bikes", cat: "outdoor",
			desc: "Tune-ups, gravel builds, and rentals a short roll from the greenbelt.",
			phone: "512-555-0166", addr: "1800 Barton Springs Rd, Austin, TX", hoursAgo: 28,
			photos: []string{"bikes-1", "bikes-2"},
			products: []prod{{"Tune-up", "Gears, brakes, wipe down", 8500}, {"Day rental hybrid", "Helmet included", 4500}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[4].id, body: "Honest shop. Didn't upsell me a new cassette.", stars: 4, hoursAgo: 20}},
		},
		{
			name: "South Congress Vintage", slug: "south-congress-vintage", cat: "retail",
			desc: "Western shirts, worn denim, and a boot wall that photographs itself.",
			phone: "512-555-0191", addr: "1512 S Congress Ave, Austin, TX", hoursAgo: 18,
			photos: []string{"vintage-1", "vintage-2", "vintage-3"},
			products: []prod{{"Pearl-snap shirt", "1970s, medium", 6200}, {"Broken-in denim", " Levi's 501, 32x32", 7800}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[5].id, body: "Left with a snap shirt I will never take off.", stars: 5, hoursAgo: 3}},
		},
		{
			name: "Easy Tiger Bake Shop", slug: "easy-tiger", cat: "food",
			desc: "Pretzels, country loaf, and beer in the garden. East Austin carb temple.",
			phone: "512-555-0133", addr: "709 E 6th St, Austin, TX", hoursAgo: 22,
			photos: []string{"tiger-1", "tiger-2"},
			products: []prod{{"Bavarian pretzel", "Mustard on the side", 450}, {"Country loaf", "Naturally leavened", 900}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[6].id, body: "Pretzel plus a pilsner in the garden. That's the whole plan.", stars: 5, hoursAgo: 6}},
		},
		{
			name: "Waterloo Records", slug: "waterloo-records", cat: "retail",
			desc: "Vinyl, tickets, and the smell of cardboard sleeves. Austin's record store of record.",
			phone: "512-555-0177", addr: "600 N Lamar Blvd, Austin, TX", hoursAgo: 90,
			photos: []string{"vinyl-1", "vinyl-2"},
			products: []prod{{"New arrival LP", "This week's staff spin", 2499}, {"Local 7-inch", "Austin bands, rotating", 1200}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[7].id, body: "Found a copy I'd hunted for two years. They knew the bin.", stars: 5, hoursAgo: 30}},
		},
		{
			name: "Bouldin Creek Cafe", slug: "bouldin-creek-cafe", cat: "coffee",
			desc: "Vegetarian cafe with migas that convert skeptics and a patio full of dogs.",
			phone: "512-555-0120", addr: "1900 S 1st St, Austin, TX", hoursAgo: 14,
			photos: []string{"bouldin-1", "bouldin-2"},
			products: []prod{{"Migra tacos", "Tofu scramble, still the move", 1299}, {"Cold brew", "House concentrate", 450}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[8].id, body: "Brunch without the meat hangover. Patio rules.", stars: 4, hoursAgo: 9}},
		},
		{
			name: "Barton Hills Yoga", slug: "barton-hills-yoga", cat: "health",
			desc: "Sweaty vinyasa, slow restorative evenings, sliding-scale community class on Sundays.",
			phone: "512-555-0155", addr: "2414 S Lamar Blvd, Austin, TX", hoursAgo: 50,
			photos: []string{"yoga-1"},
			products: []prod{{"Drop-in class", "75 minutes", 2200}, {"10-class card", "Use within 90 days", 18000}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: customerID, body: "Sunday community class is the reset I needed.", stars: 5, hoursAgo: 16}},
		},
		{
			name: "Rainey Social Club", slug: "rainey-social-club", cat: "nightlife",
			desc: "Highballs, a tiny stage, and a backyard that still feels like a house party.",
			phone: "512-555-0108", addr: "79 Rainey St, Austin, TX", hoursAgo: 10,
			photos: []string{"rainey-1", "rainey-2"},
			products: []prod{{"House highball", "Rye, citrus, soda", 1100}, {"Backyard nachos", "Enough to share, maybe", 1400}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[2].id, body: "Live set in the backyard, no cover. That's Rainey.", stars: 4, hoursAgo: 4}},
		},
		{
			name: "East Side Hardware", slug: "east-side-hardware", cat: "home",
			desc: "Bins of screws, borrowed knowledge, and a key-cutting desk that never queues long.",
			phone: "512-555-0144", addr: "1100 E 12th St, Austin, TX", hoursAgo: 60,
			photos: []string{"hw-1", "hw-2"},
			products: []prod{{"Spare key", "Cut while you wait", 350}, {"Shop broom", "The good corn one", 1800}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[3].id, body: "They found the exact hinge I didn't know the name of.", stars: 5, hoursAgo: 22}},
		},
		{
			name: "Lady Bird Kayaks", slug: "lady-bird-kayaks", cat: "outdoor",
			desc: "Sunset paddles under the bats. Board shorts optional, dry bag not.",
			phone: "512-555-0199", addr: "2100 S Lakeshore Blvd, Austin, TX", hoursAgo: 8,
			photos: []string{"kayak-1", "kayak-2", "kayak-3"},
			products: []prod{{"Sunset double kayak", "2 hours, life vest included", 6500}, {"SUP hourly", "Calm morning water", 2800}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[4].id, body: "Bats overhead on the way in. Do this once.", stars: 5, hoursAgo: 2}},
		},
		{
			name: "The Breakfast Klub", slug: "the-breakfast-klub", cat: "food", city: "Houston",
			desc: "Wings and waffles, katfish and grits. Midtown Houston breakfast that still draws a line.",
			phone: "713-555-0101", addr: "3711 Travis St, Houston, TX", hoursAgo: 14,
			photos: []string{"klub-1"},
			products: []prod{{"Wings and waffles", "A Houston classic", 1699}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[5].id, body: "Worth the wait every time I'm in town.", stars: 5, hoursAgo: 7}},
		},
		{
			name: "Ritual Coffee Roasters", slug: "ritual-coffee", cat: "coffee", city: "San Francisco",
			desc: "Mission espresso and a patio that still feels like a neighborhood hang.",
			phone: "415-555-0144", addr: "1026 Valencia St, San Francisco, CA", hoursAgo: 30,
			photos: []string{"ritual-1"},
			products: []prod{{"Espresso", "Single origin, rotating", 400}},
			reviews: []struct {
				who, body string
				stars     int
				hoursAgo  int
			}{{who: users[6].id, body: "Valencia patio, strong shot. That's the move.", stars: 4, hoursAgo: 11}},
		},
	}

	voterIDs := make([]string, 0, len(users))
	for _, u := range users {
		voterIDs = append(voterIDs, u.id)
	}

	voteCounts := []int{42, 31, 28, 19, 37, 24, 22, 18, 11, 27, 9, 33, 21, 16}

	for i, st := range catalog {
		sid := uuid.NewString()
		created := now.Add(-time.Duration(st.hoursAgo) * time.Hour)
		city := st.city
		if city == "" {
			city = "Austin"
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO stores (id, owner_id, slug, name, description, category, city, phone, address, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			sid, ownerID, st.slug, st.name, st.desc, st.cat, city, st.phone, st.addr, created); err != nil {
			return err
		}
		for pi, ph := range st.photos {
			url := fmt.Sprintf("https://picsum.photos/seed/%s/960/640", ph)
			if _, err := pool.Exec(ctx, `
				INSERT INTO photos (id, store_id, product_id, url, sort_order, created_at)
				VALUES ($1,$2,NULL,$3,$4,$5)`, uuid.NewString(), sid, url, pi, created); err != nil {
				return err
			}
		}
		var firstProduct string
		for pi, p := range st.products {
			pid := uuid.NewString()
			pslug := slug(p.name) + fmt.Sprintf("-%d", i+1)
			if pi == 0 {
				firstProduct = pid
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO products (id, store_id, slug, name, description, price_cents, created_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7)`, pid, sid, pslug, p.name, p.desc, p.cents, created.Add(time.Duration(pi)*time.Hour)); err != nil {
				return err
			}
			purl := fmt.Sprintf("https://picsum.photos/seed/%s-p%d/800/600", st.slug, pi)
			if _, err := pool.Exec(ctx, `
				INSERT INTO photos (id, store_id, product_id, url, sort_order, created_at)
				VALUES ($1,$2,$3,$4,0,$5)`, uuid.NewString(), sid, pid, purl, created); err != nil {
				return err
			}
			if pi == 0 {
				// a few product votes
				for v := 0; v < 3+i%4; v++ {
					if _, err := pool.Exec(ctx, `
						INSERT INTO votes (id, user_id, store_id, product_id, created_at)
						VALUES ($1,$2,NULL,$3,$4)`, uuid.NewString(), voterIDs[v%len(voterIDs)], pid, now.Add(-time.Duration(v+1)*time.Hour)); err != nil {
						return err
					}
				}
			}
		}
		want := voteCounts[i]
		if want > len(voterIDs) {
			want = len(voterIDs)
		}
		for v := 0; v < want; v++ {
			uid := voterIDs[v%len(voterIDs)]
			if _, err := pool.Exec(ctx, `
				INSERT INTO votes (id, user_id, store_id, product_id, created_at)
				VALUES ($1,$2,$3,NULL,$4)`, uuid.NewString(), uid, sid, now.Add(-time.Duration(v)*time.Hour)); err != nil {
				return err
			}
		}
		for _, rv := range st.reviews {
			if _, err := pool.Exec(ctx, `
				INSERT INTO reviews (id, user_id, store_id, product_id, rating, body, created_at)
				VALUES ($1,$2,$3,NULL,$4,$5,$6)`,
				uuid.NewString(), rv.who, sid, rv.stars, rv.body, now.Add(-time.Duration(rv.hoursAgo)*time.Hour)); err != nil {
				return err
			}
		}
		if firstProduct != "" && i%3 == 0 {
			if _, err := pool.Exec(ctx, `
				INSERT INTO reviews (id, user_id, store_id, product_id, rating, body, created_at)
				VALUES ($1,$2,NULL,$3,$4,$5,$6)`,
				uuid.NewString(), customerID, firstProduct, 5, "Bought this. Listed price matched what they charged.", now.Add(-2*time.Hour)); err != nil {
				return err
			}
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO category_affinities (id, user_id, guest_id, category, score, updated_at)
		VALUES ($1,$2,NULL,'food',4.0, now()), ($3,$2,NULL,'coffee',2.5, now())`,
		uuid.NewString(), customerID, uuid.NewString()); err != nil {
		return err
	}
	return nil
}

func slug(name string) string {
	out := make([]rune, 0, len(name))
	dash := false
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			out = append(out, r-'A'+'a')
			dash = false
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			dash = false
		default:
			if !dash && len(out) > 0 {
				out = append(out, '-')
				dash = true
			}
		}
	}
	s := string(out)
	if len(s) > 0 && s[len(s)-1] == '-' {
		s = s[:len(s)-1]
	}
	if s == "" {
		return "item"
	}
	return s
}
