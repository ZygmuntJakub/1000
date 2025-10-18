package main

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/ZygmuntJakub/1000/internal/engine"
)

// makeDeck builds a deterministic 24-card deck matching engine tests order.
func makeDeck() []engine.Card {
	deck := make([]engine.Card, 0, 24)
	for _, s := range []engine.Suit{engine.Spades, engine.Hearts, engine.Diamonds, engine.Clubs} {
		for _, r := range []engine.Rank{engine.Nine, engine.Jack, engine.Queen, engine.King, engine.Ten, engine.Ace} {
			deck = append(deck, engine.Card{Suit: s, Rank: r})
		}
	}
	return deck
}

func shuffleDeck(deck []engine.Card) {
	for i := range deck {
		j := rand.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}
}

func main() {
	players := []engine.PlayerID{"P1", "P2"}
	params := engine.GameParams{
		Players:     players,
		MinBid:      100,
		MinRaise:    10,
		HandCards:   10,
		MusiksCount: 2,
		MusikSize:   2,
	}
	g := engine.NewGame(params, players[0], players, nil)
	fmt.Printf("Tysiak CLI demo. Phase=%v, Dealer=%s, Players=%v\n", g.Phase, g.Dealer, players)

	// Deterministic deal (same as tests basic split):
	deck := makeDeck()
	shuffleDeck(deck)
	h1 := append([]engine.Card{}, deck[:10]...)
	h2 := append([]engine.Card{}, deck[10:20]...)
	m1 := append([]engine.Card{}, deck[20:22]...)
	m2 := append([]engine.Card{}, deck[22:24]...)
	if err := g.SetDealtCards(map[engine.PlayerID][]engine.Card{"P1": h1, "P2": h2}, [][]engine.Card{m1, m2}); err != nil {
		fmt.Printf("SetDealtCards error: %v\n", err)
		return
	}
	fmt.Printf("Deal set. Phase=%v, Auction leader=%s\n", g.Phase, g.Auction.CurrentLeader)

	// Auction: P2 bids 100, P1 passes
	if err := g.PlaceBid("P2", 100); err != nil {
		fmt.Printf("bid error: %v\n", err)
		return
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		fmt.Printf("pass error: %v\n", err)
		return
	}
	fmt.Printf("Auction done. Declarer=%s, Phase=%v\n", *g.Declarer, g.Phase)

	// Declarer chooses first musik, discard two smallest by rank (stable)
	if err := g.ChooseMusik("P2", 0); err != nil {
		fmt.Printf("choose musik: %v\n", err)
		return
	}
	// Choose two discard cards deterministically (lowest two ranks, by suit then rank)
	p2 := append([]engine.Card{}, g.Deal.Hands["P2"]...)
	sort.Slice(p2, func(i, j int) bool {
		if p2[i].Suit != p2[j].Suit {
			return p2[i].Suit < p2[j].Suit
		}
		return p2[i].Rank < p2[j].Rank
	})
	disc := []engine.Card{p2[0], p2[1]}
	if err := g.Discard("P2", disc); err != nil {
		fmt.Printf("discard: %v\n", err)
		return
	}
	fmt.Printf("Declarer took musik[0] and discarded %v, %v. Phase=%v\n", disc[0], disc[1], g.Phase)

	// Auto-play the rest of the hand:
	trick := 1
	var lastTrump *engine.Suit
	for g.Phase == engine.PhasePlay {
		leader := g.Play.CurrentTrick.Leader
		fmt.Printf("Trick %d: leader=%s\n", trick, leader)
		// Leader plays: announce marriage if possible (play a Queen of a suit where leader has both K and Q)
		leadCard, announce := pickLeadWithPossibleMarriage(g, leader)
		if err := g.PlayCard(leader, leadCard, announce); err != nil {
			fmt.Printf("lead error: %v\n", err)
			return
		}
		fmt.Printf("  %s plays %v%s\n", leader, leadCard, ternary(announce, " (announce)", ""))
		// Follower plays first legal
		follower := other(players, leader)
		legal := g.LegalPlays(follower)
		if len(legal) == 0 {
			fmt.Printf("no legal plays for %s\n", follower)
			return
		}
		if err := g.PlayCard(follower, legal[0], false); err != nil {
			fmt.Printf("follow error: %v\n", err)
			return
		}
		fmt.Printf("  %s plays %v\n", follower, legal[0])
		// Trick resolves automatically on second play
		if g.Play.LastTrickWinner != nil {
			fmt.Printf("  -> winner=%s\n", *g.Play.LastTrickWinner)
		}
		// Trump change?
		if g.Play.Trump != nil {
			if lastTrump == nil || *lastTrump != *g.Play.Trump {
				tmp := *g.Play.Trump
				fmt.Printf("  -> trump is now %v\n", tmp)
				lastTrump = &tmp
			}
		}
		trick++
		// When all cards are played, engine advances to scoring and finalizes automatically
	}

	// Print final results
	fmt.Printf("Hand finished. Phase=%v\n", g.Phase)
	fmt.Printf("Deal points: %v\n", g.Scores.DealPoints)
	fmt.Printf("Cumulative: %v\n", g.Scores.Cumulative)
}

func other(players []engine.PlayerID, p engine.PlayerID) engine.PlayerID {
	if players[0] == p {
		return players[1]
	}
	return players[0]
}

func ternary(b bool, x, y string) string {
	if b {
		return x
	}
	return y
}

// pickLeadWithPossibleMarriage selects a lead card, preferring a Queen of a suit
// where the leader also holds the King, to announce marriage.
func pickLeadWithPossibleMarriage(g *engine.GameState, leader engine.PlayerID) (engine.Card, bool) {
	hand := g.Deal.Hands[leader]
	hasK := map[engine.Suit]bool{}
	for _, c := range hand {
		if c.Rank == engine.King {
			hasK[c.Suit] = true
		}
	}
	for _, c := range hand {
		if c.Rank == engine.Queen && hasK[c.Suit] {
			return c, true
		}
	}
	// Fallback: first legal play
	legal := g.LegalPlays(leader)
	if len(legal) == 0 {
		return hand[0], false
	}
	return legal[0], false
}
