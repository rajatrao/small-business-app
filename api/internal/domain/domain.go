package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrConflict           = errors.New("conflict")
	ErrInvalid            = errors.New("invalid input")
	ErrOIDCNotConfigured  = errors.New("oidc is not configured")
	ErrDuplicateEmail     = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type Role string

const (
	RoleCustomer Role = "CUSTOMER"
	RoleOwner    Role = "OWNER"
	RoleAdmin    Role = "ADMIN"
)

func ParseRole(s string) Role {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case string(RoleOwner):
		return RoleOwner
	case string(RoleAdmin):
		return RoleAdmin
	default:
		return RoleCustomer
	}
}

var Categories = []string{
	"food", "coffee", "retail", "services", "health", "nightlife", "outdoor", "home",
}

const DefaultCity = "Austin"

const Gravity = 1.8

type TimeWindow string

const (
	WindowToday TimeWindow = "TODAY"
	WindowWeek  TimeWindow = "WEEK"
	WindowAll   TimeWindow = "ALL"
)

func ParseWindow(s string) TimeWindow {
	switch strings.ToUpper(s) {
	case "TODAY":
		return WindowToday
	case "WEEK":
		return WindowWeek
	default:
		return WindowAll
	}
}

func (w TimeWindow) Since(now time.Time) *time.Time {
	switch w {
	case WindowToday:
		t := now.Add(-24 * time.Hour)
		return &t
	case WindowWeek:
		t := now.Add(-7 * 24 * time.Hour)
		return &t
	default:
		return nil
	}
}

type AffinityKind string

const (
	AffinityViewCategory AffinityKind = "VIEW_CATEGORY"
	AffinityViewStore    AffinityKind = "VIEW_STORE"
	AffinityViewProduct  AffinityKind = "VIEW_PRODUCT"
	AffinitySearch       AffinityKind = "SEARCH"
	AffinityVote         AffinityKind = "VOTE"
	AffinityReview       AffinityKind = "REVIEW"
)

func (k AffinityKind) Weight() float64 {
	switch k {
	case AffinityViewCategory:
		return 0.5
	case AffinityViewStore, AffinityViewProduct:
		return 0.3
	case AffinitySearch:
		return 0.2
	case AffinityVote:
		return 1.0
	case AffinityReview:
		return 1.5
	default:
		return 0.1
	}
}

type User struct {
	ID           string
	Email        string
	PasswordHash *string
	Name         string
	Role         Role
	CreatedAt    time.Time
}

func (u User) FirstName() string {
	name := strings.TrimSpace(u.Name)
	if name == "" {
		return "Neighbor"
	}
	if i := strings.IndexFunc(name, unicode.IsSpace); i > 0 {
		return name[:i]
	}
	return name
}

type OIDCIdentity struct {
	ID        string
	UserID    string
	Issuer    string
	Subject   string
	CreatedAt time.Time
}

type Store struct {
	ID          string
	OwnerID     string
	Slug        string
	Name        string
	Description string
	Category    string
	City        string
	Phone       *string
	Address     *string
	CreatedAt   time.Time
}

type Product struct {
	ID          string
	StoreID     string
	Slug        string
	Name        string
	Description string
	PriceCents  int
	CreatedAt   time.Time
}

func DisplayPrice(cents int) string {
	return "$" + formatDollars(cents)
}

func formatDollars(cents int) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	d, c := cents/100, cents%100
	s := itoa(d) + "." + pad2(c)
	if neg {
		return "-" + s
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func pad2(n int) string {
	return string([]byte{'0' + byte(n/10), '0' + byte(n%10)})
}

type Photo struct {
	ID        string
	StoreID   *string
	ProductID *string
	URL       string
	SortOrder int
	CreatedAt time.Time
}

type Vote struct {
	ID        string
	UserID    string
	StoreID   *string
	ProductID *string
	CreatedAt time.Time
}

type Review struct {
	ID            string
	UserID        string
	StoreID       *string
	ProductID     *string
	Rating        int
	Body          string
	CreatedAt     time.Time
	AuthorName    string
	StoreName     string
	StoreSlug     string
	ProductName   string
	ProductSlug   string
	StoreCategory string
	StoreCity     string
}

type ReviewStats struct {
	Count     int
	RatingSum int
}

func (s ReviewStats) Weighted() float64 {
	if s.Count == 0 {
		return 0
	}
	return float64(s.RatingSum) / 5.0
}

func (s ReviewStats) Average() float64 {
	if s.Count == 0 {
		return 0
	}
	return float64(s.RatingSum) / float64(s.Count)
}

type CategoryAffinity struct {
	ID       string
	UserID   *string
	GuestID  *string
	Category string
	Score    float64
}

type ChatThread struct {
	ID        string
	StoreID   *string
	ProductID *string
	CreatedAt time.Time
}

type ChatMessage struct {
	ID        string
	ThreadID  string
	Role      string
	Body      string
	CreatedAt time.Time
}

type CatalogGrounding struct {
	StoreName        string
	StoreDescription string
	Category         string
	City             string
	Phone            *string
	Address          *string
	ProductName      string
	ProductDesc      string
	ProductPrice     string
	Products         []GroundedProduct
}

type GroundedProduct struct {
	Name  string
	Price string
	Desc  string
}

type Principal struct {
	UserID string
	Email  string
	Role   Role
}

var slugClean = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugClean.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "listing"
	}
	if len(s) > 48 {
		s = strings.Trim(s[:48], "-")
	}
	return s
}

func ValidCategory(c string) bool {
	c = strings.ToLower(c)
	for _, x := range Categories {
		if x == c {
			return true
		}
	}
	return false
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
