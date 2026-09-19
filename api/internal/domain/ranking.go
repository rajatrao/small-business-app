package domain

import "math"

// CommunityScore is HN-style gravity: (upvotes + weighted reviews) / (hours + 2)^gravity.
func CommunityScore(upvotes int, weightedReview float64, hoursSincePosted float64, gravity float64) float64 {
	if gravity <= 0 {
		gravity = Gravity
	}
	if hoursSincePosted < 0 {
		hoursSincePosted = 0
	}
	return (float64(upvotes) + weightedReview) / math.Pow(hoursSincePosted+2, gravity)
}

// ProductHeat ranks trending products: (recentVotes + 2*recentReviews) / (hoursSinceLastActivity + 2)^gravity.
func ProductHeat(recentVotes, recentReviews int, hoursSinceActivity, gravity float64) float64 {
	if gravity <= 0 {
		gravity = Gravity
	}
	if hoursSinceActivity < 0 {
		hoursSinceActivity = 0
	}
	return (float64(recentVotes) + 2*float64(recentReviews)) / math.Pow(hoursSinceActivity+2, gravity)
}

// ReviewHeat ranks trending reviews.
func ReviewHeat(hoursSince, rating, parentCommunity float64) float64 {
	if hoursSince < 0 {
		hoursSince = 0
	}
	recencyDecay := 1 / math.Pow(hoursSince+2, 0.8)
	ratingWeight := rating / 5.0
	if ratingWeight < 0 {
		ratingWeight = 0
	}
	if parentCommunity < 0 {
		parentCommunity = 0
	}
	return recencyDecay * ratingWeight * (1 + math.Log(1+parentCommunity))
}

// FeedScore applies category affinity on top of community rank.
func FeedScore(communityScore, affinityBoost float64) float64 {
	if affinityBoost < 0 {
		affinityBoost = 0
	}
	return communityScore * (1 + affinityBoost)
}

// AffinityBoost maps a stored affinity score onto a 0..0.6 multiplier bump.
func AffinityBoost(score float64) float64 {
	if score <= 0 {
		return 0
	}
	b := score / 8.0
	if b > 0.6 {
		return 0.6
	}
	return b
}

const DiversityFloor = 0.30

// ApplyDiversityFloor keeps ~30% of page-one slots from outside the viewer's
// favored categories so the feed does not collapse into a single vertical.
func ApplyDiversityFloor(items []RankedStore, favored map[string]bool, pageSize int) []RankedStore {
	if pageSize <= 0 {
		pageSize = 12
	}
	if len(items) <= 1 || len(favored) == 0 {
		if len(items) > pageSize {
			return items[:pageSize]
		}
		return items
	}
	needOther := int(math.Ceil(float64(pageSize) * DiversityFloor))
	if needOther < 1 {
		needOther = 1
	}

	favoredList := make([]RankedStore, 0, len(items))
	otherList := make([]RankedStore, 0, len(items))
	for _, it := range items {
		if favored[it.Store.Category] {
			favoredList = append(favoredList, it)
		} else {
			otherList = append(otherList, it)
		}
	}
	if len(otherList) == 0 {
		if len(items) > pageSize {
			return items[:pageSize]
		}
		return items
	}
	if len(otherList) < needOther {
		needOther = len(otherList)
	}

	out := make([]RankedStore, 0, pageSize)
	fi, oi, othersUsed := 0, 0, 0
	for len(out) < pageSize && (fi < len(favoredList) || oi < len(otherList)) {
		slotsLeft := pageSize - len(out)
		othersStillNeeded := needOther - othersUsed
		takeOther := oi < len(otherList) && (othersStillNeeded >= slotsLeft || (len(out) > 0 && othersUsed < needOther && len(out)%3 == 2))
		if takeOther {
			out = append(out, otherList[oi])
			oi++
			othersUsed++
			continue
		}
		if fi < len(favoredList) {
			out = append(out, favoredList[fi])
			fi++
			continue
		}
		if oi < len(otherList) {
			out = append(out, otherList[oi])
			oi++
			if favored[otherList[oi-1].Store.Category] {
				// not favored
			} else {
				othersUsed++
			}
			continue
		}
		break
	}
	return out
}

type RankedStore struct {
	Store           Store
	Rank            int
	CommunityScore  float64
	FeedScore       float64
	Upvotes         int
	WeightedReviews float64
}

type TrendingProduct struct {
	Product Product
	Store   Store
	Heat    float64
}

type TrendingReview struct {
	Review Review
	Heat   float64
}

type HomePage struct {
	City             string
	RankedStores     []RankedStore
	TrendingProducts []TrendingProduct
	TrendingReviews  []TrendingReview
}
