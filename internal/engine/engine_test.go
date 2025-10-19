package engine

import (
	"testing"
)

func makeDeck() []Card {
	deck := make([]Card, 0, 24)
	for s := Spades; s <= Hearts; s++ {
		for _, r := range []Rank{Nine, Jack, Queen, King, Ten, Ace} {
			deck = append(deck, Card{Suit: s, Rank: r})
		}
	}
	return deck
}

func TestEdgeCases_TableDriven(t *testing.T) {
	type tc struct {
		name string
		run  func(t *testing.T)
	}
	cases := []tc{
		{
			name: "invalid bid raise too small",
			run: func(t *testing.T) {
				players := []PlayerID{"P1", "P2"}
				g := NewGame(GameParams{Players: players}, players[0], players, nil)
				deck := makeDeck()
				_ = g.SetDealtCards(map[PlayerID][]Card{"P1": deck[:10], "P2": deck[10:20]}, [][]Card{deck[20:22], deck[22:24]})
				if err := g.PlaceBid("P2", 110); err != nil {
					t.Fatalf("seed bid: %v", err)
				}
				if err := g.PlaceBid("P1", 115); err == nil {
					t.Fatalf("expected invalid small raise")
				}
			},
		},
		{
			name: "out-of-turn bid",
			run: func(t *testing.T) {
				players := []PlayerID{"P1", "P2"}
				g := NewGame(GameParams{Players: players}, players[0], players, nil)
				deck := makeDeck()
				_ = g.SetDealtCards(map[PlayerID][]Card{"P1": deck[:10], "P2": deck[10:20]}, [][]Card{deck[20:22], deck[22:24]})
				// P2 starts; P1 tries out-of-turn
				if err := g.PlaceBid("P1", 100); err == nil {
					t.Fatalf("expected out-of-turn error")
				}
			},
		},
		{
			name: "invalid musik index",
			run: func(t *testing.T) {
				players := []PlayerID{"P1", "P2"}
				g := NewGame(GameParams{Players: players}, players[0], players, nil)
				deck := makeDeck()
				_ = g.SetDealtCards(map[PlayerID][]Card{"P1": deck[:10], "P2": deck[10:20]}, [][]Card{deck[20:22], deck[22:24]})
				_ = g.PlaceBid("P2", 100)
				_ = g.PlaceBid("P1", 0)
				if err := g.ChooseMusik("P2", 5); err == nil {
					t.Fatalf("expected invalid musik index error")
				}
			},
		},
		{
			name: "invalid discard size",
			run: func(t *testing.T) {
				players := []PlayerID{"P1", "P2"}
				g := NewGame(GameParams{Players: players}, players[0], players, nil)
				deck := makeDeck()
				_ = g.SetDealtCards(map[PlayerID][]Card{"P1": deck[:10], "P2": deck[10:20]}, [][]Card{deck[20:22], deck[22:24]})
				_ = g.PlaceBid("P2", 110)
				_ = g.PlaceBid("P1", 0)
				if err := g.ChooseMusik("P2", 0); err != nil {
					t.Fatalf("musik: %v", err)
				}
				// Discard 1 only
				if err := g.Discard("P2", []Card{g.Deal.Hands["P2"][0]}); err == nil {
					t.Fatalf("expected invalid discard size error")
				}
			},
		},
		{
			name: "duplicate deal cards",
			run: func(t *testing.T) {
				players := []PlayerID{"P1", "P2"}
				g := NewGame(GameParams{Players: players}, players[0], players, nil)
				// Intentionally duplicate a card in both hands
				c := Card{Spades, Nine}
				h1 := []Card{c, Card{Spades, Jack}, Card{Spades, Queen}, Card{Spades, King}, Card{Spades, Ten}, Card{Spades, Ace}, Card{Clubs, Nine}, Card{Clubs, Jack}, Card{Clubs, Queen}, Card{Clubs, King}}
				h2 := []Card{c, Card{Clubs, Ten}, Card{Clubs, Ace}, Card{Diamonds, Nine}, Card{Diamonds, Jack}, Card{Diamonds, Queen}, Card{Diamonds, King}, Card{Diamonds, Ten}, Card{Diamonds, Ace}, Card{Hearts, Nine}}
				musiks := [][]Card{{Card{Hearts, Jack}, Card{Hearts, Queen}}, {Card{Hearts, King}, Card{Hearts, Ten}}}
				if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, musiks); err == nil {
					t.Fatalf("expected duplicate card detection error")
				}
			},
		},
		{
			name: "must follow suit violation",
			run: func(t *testing.T) {
				players := []PlayerID{"P1", "P2"}
				g := NewGame(GameParams{Players: players}, players[0], players, nil)
				// Minimal deterministic hands
				h1 := []Card{Card{Spades, Ten}, Card{Clubs, Nine}, Card{Diamonds, Nine}, Card{Hearts, Nine}, Card{Clubs, Jack}, Card{Diamonds, Jack}, Card{Hearts, Jack}, Card{Clubs, Queen}, Card{Diamonds, Queen}, Card{Hearts, Queen}}
				h2 := []Card{Card{Spades, Nine}, Card{Clubs, Ten}, Card{Diamonds, Ten}, Card{Hearts, Ten}, Card{Clubs, Ace}, Card{Diamonds, Ace}, Card{Hearts, Ace}, Card{Clubs, King}, Card{Diamonds, King}, Card{Hearts, King}}
				musiks := [][]Card{{Card{Spades, Jack}, Card{Spades, Queen}}, {Card{Spades, King}, Card{Spades, Ace}}}
				if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, musiks); err != nil {
					t.Fatalf("deal: %v", err)
				}
				_ = g.PlaceBid("P2", 110)
				_ = g.PlaceBid("P1", 0)
				if err := g.ChooseMusik("P2", 0); err != nil {
					t.Fatalf("musik: %v", err)
				}
				// Discard two cards from the end to avoid removing Spades Nine
				p2 := g.Deal.Hands["P2"]
				if err := g.Discard("P2", []Card{p2[len(p2)-1], p2[len(p2)-2]}); err != nil {
					t.Fatalf("discard: %v", err)
				}
				// P2 leads Spades Nine; P1 has Spades Ten and must follow; attempt off-suit should error
				if err := g.PlayCard("P2", Card{Spades, Nine}, false); err != nil {
					t.Fatalf("lead: %v", err)
				}
				if err := g.PlayCard("P1", Card{Clubs, Nine}, false); err == nil {
					t.Fatalf("expected follow-suit violation")
				}
			},
		},
		{
			name: "trump-led must follow trump",
			run: func(t *testing.T) {
				players := []PlayerID{"P1", "P2"}
				g := NewGame(GameParams{Players: players}, players[0], players, nil)
				// Ensure marriage will set trump to clubs, then test trump-led must-follow
				h2 := []Card{{Clubs, Queen}, {Clubs, King}, {Spades, Nine}, {Spades, Jack}, {Diamonds, Nine}, {Diamonds, Jack}, {Hearts, Nine}, {Hearts, Jack}, {Hearts, Ace}, {Diamonds, Ace}}
				h1 := []Card{{Clubs, Ace}, {Clubs, Nine}, {Spades, Ten}, {Diamonds, Ten}, {Hearts, Ten}, {Spades, Queen}, {Diamonds, Queen}, {Hearts, Queen}, {Spades, King}, {Diamonds, King}}
				musiks := [][]Card{{Card{Clubs, Jack}, Card{Hearts, King}}, {Card{Clubs, Ten}, Card{Spades, Ace}}}
				if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, musiks); err != nil {
					t.Fatalf("deal: %v", err)
				}
				_ = g.PlaceBid("P2", 110)
				_ = g.PlaceBid("P1", 0)
				if err := g.ChooseMusik("P2", 0); err != nil {
					t.Fatalf("musik: %v", err)
				}
				// Discard last two to keep S9 and C9 intact
				p2h := g.Deal.Hands["P2"]
				if err := g.Discard("P2", []Card{p2h[len(p2h)-1], p2h[len(p2h)-2]}); err != nil {
					t.Fatalf("discard: %v", err)
				}
				if err := g.PlayCard("P2", Card{Clubs, Queen}, true); err != nil {
					t.Fatalf("announce: %v", err)
				}
				// P1 must follow clubs on the first trick
				if err := g.PlayCard("P1", Card{Clubs, Nine}, false); err != nil {
					t.Fatalf("follower must follow clubs: %v", err)
				}
				// Now lead trump clubs; P1 must follow clubs. Attempt non-club should fail.
				if err := g.PlayCard("P2", Card{Clubs, King}, false); err != nil {
					t.Fatalf("lead trump: %v", err)
				}
				if err := g.PlayCard("P1", Card{Spades, Queen}, false); err == nil {
					t.Fatalf("expected trump-led must-follow violation")
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, c.run)
	}
}

func TestScoringSettlementSuccess(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	// Set declarer and highest bid
	dec := PlayerID("P1")
	g.Declarer = &dec
	g.Auction.Bids = []AuctionBid{{Player: "P1", Value: 120, Pass: false}, {Player: "P2", Value: 0, Pass: true}}
	// Set deal points (including marriages hypothetically)
	g.Scores.DealPoints["P1"] = 130
	g.Scores.DealPoints["P2"] = 20
	g.Phase = PhaseScoring
	if err := g.FinalizeScoring(); err != nil {
		t.Fatalf("FinalizeScoring: %v", err)
	}
	if g.Phase != PhaseHandEnd {
		t.Fatalf("expected PhaseHandEnd, got %v", g.Phase)
	}
	if g.Scores.Cumulative["P1"] != 120 {
		t.Fatalf("declarer cumulative expected 120, got %d", g.Scores.Cumulative["P1"])
	}
	if g.Scores.Cumulative["P2"] != 20 {
		t.Fatalf("defender cumulative expected 20, got %d", g.Scores.Cumulative["P2"])
	}
}

func TestScoringSettlementFailure(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	// Set declarer and highest bid
	dec := PlayerID("P2")
	g.Declarer = &dec
	g.Auction.Bids = []AuctionBid{{Player: "P1", Value: 100, Pass: false}, {Player: "P2", Value: 120, Pass: false}}
	// Set deal points where declarer fails
	g.Scores.DealPoints["P2"] = 110
	g.Scores.DealPoints["P1"] = 10
	g.Phase = PhaseScoring
	if err := g.FinalizeScoring(); err != nil {
		t.Fatalf("FinalizeScoring: %v", err)
	}
	if g.Phase != PhaseHandEnd {
		t.Fatalf("expected PhaseHandEnd, got %v", g.Phase)
	}
	if g.Scores.Cumulative["P2"] != -120 {
		t.Fatalf("declarer cumulative expected -120, got %d", g.Scores.Cumulative["P2"])
	}
	if g.Scores.Cumulative["P1"] != 10 {
		t.Fatalf("defender cumulative expected 10, got %d", g.Scores.Cumulative["P1"])
	}
}

func TestTrumpLedMustFollowTrump(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	// Deterministic hands: P2 has clubs Q,K; P1 has clubs Ace and clubs Nine.
	h2 := []Card{deck[8], deck[9], deck[0], deck[1], deck[2], deck[3], deck[12], deck[13], deck[14], deck[15]}    // P2: Clubs Q,K + spades 0..3 + diamonds 12..15
	h1 := []Card{deck[11], deck[6], deck[4], deck[5], deck[16], deck[17], deck[18], deck[19], deck[20], deck[21]} // P1: Clubs A, Nine + spades 4,5 + diamonds 16,17 + hearts 18..21
	m1 := []Card{deck[7], deck[10]}                                                                               // clubs J, Ten
	m2 := []Card{deck[22], deck[23]}                                                                              // hearts 10,A
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if err := g.ChooseMusik("P2", 0); err != nil {
		t.Fatalf("musik: %v", err)
	}
	// Discard two non-clubs to keep Q,K
	disc := []Card{}
	for _, c := range g.Deal.Hands["P2"] {
		if c.Suit != Clubs {
			disc = append(disc, c)
			if len(disc) == 2 {
				break
			}
		}
	}
	if err := g.Discard("P2", disc); err != nil {
		t.Fatalf("discard: %v", err)
	}
	// Trick 1: P2 announces clubs by leading Q (trump set to clubs)
	if err := g.PlayCard("P2", Card{Clubs, Queen}, true); err != nil {
		t.Fatalf("announce: %v", err)
	}
	// P1 must follow the led suit (clubs); play Clubs Nine so P2 wins the trick
	if err := g.PlayCard("P1", Card{Clubs, Nine}, false); err != nil {
		t.Fatalf("follow clubs: %v", err)
	}
	// Trick 2: P2 leads trump (Clubs King); P1 must follow trump
	if err := g.PlayCard("P2", Card{Clubs, King}, false); err != nil {
		t.Fatalf("lead trump: %v", err)
	}
	// Try illegal non-club
	if err := g.PlayCard("P1", Card{Spades, Ace}, false); err == nil {
		t.Fatalf("expected error when not following trump suit")
	}
	// Legal follow
	if err := g.PlayCard("P1", Card{Clubs, Ace}, false); err != nil {
		t.Fatalf("follow trump: %v", err)
	}
}

func TestIllegalMarriageCases(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	// Ensure P2 does NOT have both K/Q of hearts; only Queen
	h2 := []Card{deck[22], deck[0], deck[1], deck[2], deck[3], deck[4], deck[5], deck[6], deck[7], deck[8]}
	h1 := []Card{deck[9], deck[10], deck[11], deck[12], deck[13], deck[14], deck[15], deck[16], deck[17], deck[18]}
	m1 := []Card{deck[19], deck[20]}
	m2 := []Card{deck[21], deck[23]}
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if err := g.ChooseMusik("P2", 0); err != nil {
		t.Fatalf("musik: %v", err)
	}
	// Discard from the end to avoid removing specific cards referenced later
	p2h := g.Deal.Hands["P2"]
	if err := g.Discard("P2", []Card{p2h[len(p2h)-1], p2h[len(p2h)-2]}); err != nil {
		t.Fatalf("discard: %v", err)
	}
	// Attempt to announce with Hearts Queen without holding King should fail
	if err := g.PlayCard("P2", Card{Hearts, Queen}, true); err == nil {
		t.Fatalf("expected invalid marriage announcement error")
	}
	// Also check non-leader cannot announce: let P2 lead a normal card, then P1 tries to announce improperly
	// Start a new trick: lead a spade without announce
	// If previous attempt failed, trick didn't start; lead properly now
	if err := g.PlayCard("P2", Card{Spades, Nine}, false); err != nil {
		t.Fatalf("lead: %v", err)
	}
	// P1 attempts to toggle announce though not leader; engine should ignore announce flag (no error, no trump/points)
	p1c := g.Deal.Hands["P1"][0]
	if err := g.PlayCard("P1", p1c, true); err != nil {
		t.Fatalf("unexpected error for follower announce flag: %v", err)
	}
	if g.Play.Trump != nil {
		t.Fatalf("trump should remain nil for follower announce attempt")
	}
}

func TestPlayNotInHandError(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	h1 := append([]Card{}, deck[:10]...)
	h2 := append([]Card{}, deck[10:20]...)
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{[]Card{deck[20], deck[21]}, []Card{deck[22], deck[23]}}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if err := g.ChooseMusik("P2", 0); err != nil {
		t.Fatalf("musik: %v", err)
	}
	if err := g.Discard("P2", []Card{g.Deal.Hands["P2"][0], g.Deal.Hands["P2"][1]}); err != nil {
		t.Fatalf("discard: %v", err)
	}
	// Try to play a card not in hand
	if err := g.PlayCard("P2", Card{Suit: Hearts, Rank: Ace}, false); err == nil {
		t.Fatalf("expected error playing card not in hand")
	}
}

func TestLegalPlaysWhenVoid(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	// Construct so P1 leads spade, P2 is void in spades
	h1 := []Card{deck[0], deck[1], deck[6], deck[12], deck[18], deck[7], deck[13], deck[19], deck[20], deck[21]}   // includes spades 0,1
	h2 := []Card{deck[8], deck[9], deck[10], deck[11], deck[14], deck[15], deck[16], deck[17], deck[22], deck[23]} // no spades
	m1 := []Card{deck[2], deck[3]}
	m2 := []Card{deck[4], deck[5]}
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	// Make P1 declarer so P1 leads first: P2 passes, P1 auto becomes declarer
	if err := g.PlaceBid("P2", 0); err != nil {
		t.Fatalf("p2 pass: %v", err)
	}
	if err := g.ChooseMusik("P1", 0); err != nil {
		t.Fatalf("p1 musik: %v", err)
	}
	if err := g.Discard("P1", []Card{g.Deal.Hands["P1"][0], g.Deal.Hands["P1"][1]}); err != nil {
		t.Fatalf("p1 discard: %v", err)
	}
	// P1 should lead a spade; find a spade in P1 hand
	// Find a spade in P1 hand
	var sp Card
	for _, c := range g.Deal.Hands["P1"] {
		if c.Suit == Spades {
			sp = c
			break
		}
	}
	if (sp == Card{}) {
		t.Fatalf("no spade in P1 hand to lead")
	}
	if err := g.PlayCard("P1", sp, false); err != nil {
		t.Fatalf("lead spade: %v", err)
	}
	lp := g.LegalPlays("P2")
	if len(lp) != len(g.Deal.Hands["P2"]) {
		t.Fatalf("when void in led suit, all cards should be legal to play")
	}
}

func TestFirstLeaderAutomaticallyBids100(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": deck[:10], "P2": deck[10:20]}, [][]Card{[]Card{deck[20], deck[21]}, []Card{deck[22], deck[23]}}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	if g.Auction.CurrentLeader != players[1] {
		t.Fatalf("expected first bid to be from %s", players[1])
	}
	if g.Dealer != players[0] {
		t.Fatalf("expected dealer to be %s", players[0])
	}
	if err := g.PlaceBid("P2", 0); err != nil {
		t.Fatalf("p2 pass: %v", err)
	}
	if g.Auction.Bids[0].Value != 100 {
		t.Fatalf("expected first bid to be 100")
	}
	if g.Auction.Bids[0].Player != players[0] {
		t.Fatalf("expected first bid to be from %s", players[0])
	}
}

func TestTrickResolutionNoTrump(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	// P2 will lead Diamonds Ten, P1 will play Diamonds Ace and win
	// Diamonds indices: 12..17 [Nine,Jack,Queen,King,Ten,Ace]
	h2 := []Card{deck[16], deck[0], deck[1], deck[2], deck[3], deck[4], deck[5], deck[6], deck[7], deck[8]}         // includes Diamonds Ten
	h1 := []Card{deck[17], deck[9], deck[10], deck[11], deck[13], deck[14], deck[15], deck[18], deck[19], deck[20]} // includes Diamonds Ace
	m1 := []Card{deck[21], deck[22]}
	m2 := []Card{deck[12], deck[23]}
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if err := g.ChooseMusik("P2", 0); err != nil {
		t.Fatalf("musik: %v", err)
	}
	// Discard two cards not diamonds to keep intended cards
	disc := []Card{}
	for _, c := range g.Deal.Hands["P2"] {
		if c.Suit != Diamonds {
			disc = append(disc, c)
		}
		if len(disc) == 2 {
			break
		}
	}
	if err := g.Discard("P2", disc); err != nil {
		t.Fatalf("discard: %v", err)
	}
	if err := g.PlayCard("P2", Card{Diamonds, Ten}, false); err != nil {
		t.Fatalf("lead: %v", err)
	}
	if err := g.PlayCard("P1", Card{Diamonds, Ace}, false); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if g.Play.LastTrickWinner == nil || *g.Play.LastTrickWinner != "P1" {
		t.Fatalf("expected P1 to win trick")
	}
	if g.Scores.DealPoints["P1"] <= 0 {
		t.Fatalf("expected trick points for P1")
	}
}

func TestTrickResolutionWithTrump(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	// P2: Clubs Q,K, Clubs Nine, Spades Nine; P1: no clubs, has Spades Ace and Hearts Nine.
	h2 := []Card{deck[8], deck[9], deck[6], deck[0], deck[12], deck[13], deck[14], deck[15], deck[16], deck[17]}
	h1 := []Card{deck[5], deck[4], deck[1], deck[2], deck[3], deck[18], deck[19], deck[20], deck[21], deck[22]}
	m1 := []Card{deck[7], deck[10]}
	m2 := []Card{deck[11], deck[23]}
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if err := g.ChooseMusik("P2", 0); err != nil {
		t.Fatalf("musik: %v", err)
	}
	// Discard two specific diamond cards to keep Spades Nine and Clubs Nine in hand
	if err := g.Discard("P2", []Card{{Suit: Diamonds, Rank: Nine}, {Suit: Diamonds, Rank: Jack}}); err != nil {
		t.Fatalf("discard: %v", err)
	}
	// Trick 1: P2 announces clubs by leading Q; P1 discards a spade
	if err := g.PlayCard("P2", Card{Clubs, Queen}, true); err != nil {
		t.Fatalf("announce: %v", err)
	}
	if err := g.PlayCard("P1", Card{Spades, Jack}, false); err != nil {
		t.Fatalf("follow any: %v", err)
	}
	// Trick 2: P2 leads Spades Nine; P1 wins with Spades Ace to gain lead
	if err := g.PlayCard("P2", Card{Spades, Nine}, false); err != nil {
		t.Fatalf("lead2: %v", err)
	}
	if err := g.PlayCard("P1", Card{Spades, Ace}, false); err != nil {
		t.Fatalf("follow2: %v", err)
	}
	// Trick 3: P1 leads Hearts Nine; P2 undertrumps with Clubs Nine and wins
	if err := g.PlayCard("P1", Card{Hearts, Nine}, false); err != nil {
		t.Fatalf("lead3: %v", err)
	}
	if err := g.PlayCard("P2", Card{Clubs, Nine}, false); err != nil {
		t.Fatalf("trump: %v", err)
	}
	if g.Play.LastTrickWinner == nil || *g.Play.LastTrickWinner != "P2" {
		t.Fatalf("expected P2 to win by trumping")
	}
}

func TestLegalHelpers(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	// Deterministic hands to guarantee shared suit (Spades)
	// P1: Spades {9,J,10}, Clubs {9, J}, Diamonds {9, J}, Hearts {9, J}
	// P2: Spades {Q,K,A}, Clubs {10, A}, Diamonds {10, A}, Hearts {10, A}
	h1 := []Card{{Spades, Nine}, {Spades, Jack}, {Spades, Ten}, {Clubs, Nine}, {Clubs, Jack}, {Diamonds, Nine}, {Diamonds, Jack}, {Hearts, Nine}, {Hearts, Jack}, {Clubs, Queen}}
	h2 := []Card{{Spades, Queen}, {Spades, King}, {Spades, Ace}, {Clubs, Ten}, {Clubs, Ace}, {Diamonds, Ten}, {Diamonds, Ace}, {Hearts, Ten}, {Hearts, Ace}, {Diamonds, Queen}}
	// Musiks arbitrary but consistent
	m1 := []Card{{Clubs, King}, {Hearts, Queen}}
	m2 := []Card{{Diamonds, King}, {Hearts, King}}
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	// Auction leader should be P2, legal bids are pass (0) and next (110)
	bids := g.LegalBids("P2")
	if len(bids) != 2 || bids[0] != 0 || bids[1] != 110 {
		t.Fatalf("unexpected LegalBids: %v", bids)
	}
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid: %v", err)
	}
	// Now P1 turn; next legal bid should be 120
	bids = g.LegalBids("P1")
	if len(bids) != 2 || bids[1] != 120 {
		t.Fatalf("expected next bid 120, got %v", bids)
	}

	// Start play to check LegalPlays
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if err := g.ChooseMusik("P2", 0); err != nil {
		t.Fatalf("musik: %v", err)
	}
	// Discard two cards that won't affect shared spade suit
	if err := g.Discard("P2", []Card{{Clubs, Ten}, {Diamonds, Ten}}); err != nil {
		t.Fatalf("discard: %v", err)
	}
	// On lead: all cards legal
	lp := g.LegalPlays("P2")
	if len(lp) != len(g.Deal.Hands["P2"]) {
		t.Fatalf("expected all cards legal on lead")
	}
	// Lead spade from P2, P1 must follow with spade
	if err := g.PlayCard("P2", Card{Spades, Queen}, false); err != nil {
		t.Fatalf("lead: %v", err)
	}
	lp2 := g.LegalPlays("P1")
	if len(lp2) == 0 {
		t.Fatalf("expected some legal plays for P1")
	}
	for _, c := range lp2 {
		if c.Suit != Spades {
			t.Fatalf("expected only spades in LegalPlays when must follow, got %v", c)
		}
	}
}

func TestAuctionFlow2P(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)

	// Deal 10/10 and two musiks of 2
	deck := makeDeck()
	h1 := append([]Card{}, deck[:10]...)
	h2 := append([]Card{}, deck[10:20]...)
	m1 := append([]Card{}, deck[20:22]...)
	m2 := append([]Card{}, deck[22:24]...)
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards error: %v", err)
	}

	// Dealer is P1, so CurrentLeader for auction is next player P2
	if g.Auction.CurrentLeader != "P2" {
		t.Fatalf("expected P2 to lead auction")
	}

	// P2 bids 110, P1 passes -> P2 becomes declarer and PhaseTalonExchange
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid error: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass error: %v", err)
	}
	if g.Declarer == nil || *g.Declarer != "P2" {
		t.Fatalf("expected P2 declarer")
	}
	if g.Phase != PhaseTalonExchange {
		t.Fatalf("expected PhaseTalonExchange, got %v", g.Phase)
	}
}

func TestMusikChooseDiscardAndStartPlay(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	h1 := append([]Card{}, deck[:10]...)
	h2 := append([]Card{}, deck[10:20]...)
	m1 := append([]Card{}, deck[20:22]...)
	m2 := append([]Card{}, deck[22:24]...)
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards error: %v", err)
	}
	// Make P2 declarer
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid error: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass error: %v", err)
	}
	// Choose musik index 0 and discard 2 from P2
	if err := g.ChooseMusik("P2", 0); err != nil {
		t.Fatalf("choose musik: %v", err)
	}
	if err := g.Discard("P2", []Card{g.Deal.Hands["P2"][0], g.Deal.Hands["P2"][1]}); err != nil {
		t.Fatalf("discard: %v", err)
	}
	if g.Phase != PhasePlay {
		t.Fatalf("expected PhasePlay, got %v", g.Phase)
	}
	if len(g.Deal.TableCards) != 4 {
		t.Fatalf("expected 4 table cards, got %d", len(g.Deal.TableCards))
	}
}

func TestFollowSuitEnforced(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	// Choose unique cards from deck:
	// P1 hand includes Spades Ten (idx 4)
	h1 := []Card{deck[4], deck[12], deck[13], deck[14], deck[15], deck[18], deck[19], deck[20], deck[21], deck[22]}
	// P2 hand includes Spades Nine (idx 0)
	h2 := []Card{deck[0], deck[1], deck[2], deck[3], deck[5], deck[6], deck[7], deck[8], deck[9], deck[10]}
	// Musiks are remaining two cards: deck[11] (Clubs Ace) and deck[23] (Hearts Ace)
	m1 := []Card{deck[11], deck[23]}
	m2 := []Card{deck[16], deck[17]}
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	// Make P2 declarer and start play
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if err := g.ChooseMusik("P2", 0); err != nil {
		t.Fatalf("musik: %v", err)
	}
	// Discard two cards that are not Spades Nine to keep test intention
	// Pick last two cards safely
	p2hand := g.Deal.Hands["P2"]
	if err := g.Discard("P2", []Card{p2hand[len(p2hand)-1], p2hand[len(p2hand)-2]}); err != nil {
		t.Fatalf("discard: %v", err)
	}
	// P2 leads Spades Nine
	if err := g.PlayCard("P2", Card{Spades, Nine}, false); err != nil {
		t.Fatalf("lead play: %v", err)
	}
	// P1 must follow spades; attempt illegal off-suit should fail
	if err := g.PlayCard("P1", h1[1], false); err == nil { // h1[1] is not spade
		t.Fatalf("expected error when not following suit")
	}
	// Legal follow with Spades Ten
	if err := g.PlayCard("P1", Card{Spades, Ten}, false); err != nil {
		t.Fatalf("follow spade: %v", err)
	}
}

func TestMarriageSetsTrumpAndScores(t *testing.T) {
	players := []PlayerID{"P1", "P2"}
	g := NewGame(GameParams{Players: players}, players[0], players, nil)
	deck := makeDeck()
	// Ensure P2 has Hearts K and Q in hand uniquely
	// Hearts indices are 18..23: [Nine,Jack,Queen,King,Ten,Ace]
	h2 := []Card{deck[20], deck[21], deck[22], deck[23], deck[0], deck[6], deck[7], deck[12], deck[13], deck[14]}
	// P1 takes other unique cards avoiding duplicates
	h1 := []Card{deck[1], deck[2], deck[3], deck[4], deck[5], deck[8], deck[9], deck[10], deck[11], deck[15]}
	// Musiks use remaining
	m1 := []Card{deck[16], deck[17]}
	m2 := []Card{deck[18], deck[19]}
	if err := g.SetDealtCards(map[PlayerID][]Card{"P1": h1, "P2": h2}, [][]Card{m1, m2}); err != nil {
		t.Fatalf("SetDealtCards: %v", err)
	}
	if err := g.PlaceBid("P2", 110); err != nil {
		t.Fatalf("bid: %v", err)
	}
	if err := g.PlaceBid("P1", 0); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if err := g.ChooseMusik("P2", 1); err != nil {
		t.Fatalf("musik: %v", err)
	}
	// Discard two cards that are not Hearts Q/K to keep them for announcement
	p2hand := g.Deal.Hands["P2"]
	// Find two non-heart cards to discard
	disc := []Card{}
	for _, c := range p2hand {
		if c.Suit != Hearts {
			disc = append(disc, c)
			if len(disc) == 2 {
				break
			}
		}
	}
	if len(disc) != 2 {
		t.Fatalf("could not find discard candidates")
	}
	if err := g.Discard("P2", disc); err != nil {
		t.Fatalf("discard: %v", err)
	}
	// P2 leads Hearts Queen and announces marriage -> trump becomes Hearts
	if err := g.PlayCard("P2", Card{Hearts, Queen}, true); err != nil {
		t.Fatalf("announce: %v", err)
	}
	if g.Play.Trump == nil || *g.Play.Trump != Hearts {
		t.Fatalf("expected hearts trump")
	}
	if g.Scores.DealPoints["P2"] != MarriageValue(Hearts) {
		t.Fatalf("expected marriage points added")
	}
}
