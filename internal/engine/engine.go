package engine

import (
	"errors"
	"fmt"
)

type PhaseError string

func (e PhaseError) Error() string { return string(e) }

func NewGame(params GameParams, dealer PlayerID, players []PlayerID, cumulative map[PlayerID]int) *GameState {
	if params.MinBid == 0 {
		params.MinBid = 100
	}
	if params.MinRaise == 0 {
		params.MinRaise = 10
	}
	if params.HandCards == 0 {
		params.HandCards = 10
	}
	if params.MusiksCount == 0 {
		params.MusiksCount = 2
	}
	if params.MusikSize == 0 {
		params.MusikSize = 2
	}
	if params.MaxGamePoints == 0 {
		params.MaxGamePoints = 1000
	}
	gs := &GameState{
		Phase:  PhaseInit,
		Params: params,
		Dealer: dealer,
		Scores: ScoreState{DealPoints: map[PlayerID]int{}, Cumulative: map[PlayerID]int{}},
	}
	gs.Params.Players = players
	for _, p := range players {
		if _, ok := gs.Scores.Cumulative[p]; !ok {
			if cumulative != nil {
				gs.Scores.Cumulative[p] = cumulative[p]
			} else {
				gs.Scores.Cumulative[p] = 0
			}
		}
	}
	gs.Phase = PhaseDeal
	gs.Auction = AuctionState{ActivePlayers: append([]PlayerID{}, players...), CurrentLeader: nextPlayer(players, dealer), MinRaise: gs.Params.MinRaise, Bids: []AuctionBid{{Player: dealer, Value: gs.Params.MinBid, Pass: false}}}
	return gs
}

func nextPlayer(players []PlayerID, current PlayerID) PlayerID {
	for i, p := range players {
		if p == current {
			return players[(i+1)%len(players)]
		}
	}
	return players[0]
}

func (g *GameState) SetDealtCards(hands map[PlayerID][]Card, musiks [][]Card) error {
	if g.Phase != PhaseDeal {
		return PhaseError("not in deal phase")
	}
	if len(g.Params.Players) != 2 {
		return fmt.Errorf("only 2 players supported in initial version")
	}
	for _, p := range g.Params.Players {
		if len(hands[p]) != g.Params.HandCards {
			return fmt.Errorf("player %s must have %d cards", p, g.Params.HandCards)
		}
	}
	if len(musiks) != g.Params.MusiksCount {
		return fmt.Errorf("expected %d musiks", g.Params.MusiksCount)
	}
	for i := range musiks {
		if len(musiks[i]) != g.Params.MusikSize {
			return fmt.Errorf("musik %d must have %d cards", i, g.Params.MusikSize)
		}
	}
	seen := map[string]bool{}
	add := func(c Card) error {
		key := fmt.Sprintf("%d-%d", c.Suit, c.Rank)
		if seen[key] {
			return fmt.Errorf("duplicate card detected: %v", c)
		}
		seen[key] = true
		return nil
	}
	for _, p := range g.Params.Players {
		for _, c := range hands[p] {
			if err := add(c); err != nil {
				return err
			}
		}
	}
	for _, m := range musiks {
		for _, c := range m {
			if err := add(c); err != nil {
				return err
			}
		}
	}
	if len(seen) != 24 {
		return fmt.Errorf("expected 24 unique cards, got %d", len(seen))
	}
	g.Deal.Hands = hands
	g.Deal.Musiks = musiks
	g.Deal.TableCards = nil
	g.Phase = PhaseAuction
	return nil
}

func (g *GameState) PlaceBid(player PlayerID, value int) error {
	if g.Phase != PhaseAuction {
		return PhaseError("not in auction phase")
	}
	turn := g.Auction.CurrentLeader
	if player != turn {
		return fmt.Errorf("not %s's turn", player)
	}

	// value == 0 is pass
	if value == 0 {
		g.Auction.Bids = append(g.Auction.Bids, AuctionBid{Player: player, Value: 0, Pass: true})
		g.Auction.ActivePlayers = removePlayer(g.Auction.ActivePlayers, player)
		if len(g.Auction.ActivePlayers) == 1 {
			g.Declarer = &g.Auction.ActivePlayers[0]
			g.Phase = PhaseTalonExchange
			return nil
		}
		g.Auction.CurrentLeader = nextPlayer(g.Params.Players, turn)
		return nil
	}
	high := 0
	for _, b := range g.Auction.Bids {
		if !b.Pass && b.Value > high {
			high = b.Value
		}
	}
	if value < g.Params.MinBid || value <= high || value-high < g.Params.MinRaise {
		return fmt.Errorf("illegal bid")
	}
	g.Auction.Bids = append(g.Auction.Bids, AuctionBid{Player: player, Value: value})
	g.Auction.CurrentLeader = nextPlayer(g.Params.Players, turn)
	return nil
}

func removePlayer(xs []PlayerID, x PlayerID) []PlayerID {
	out := make([]PlayerID, 0, len(xs))
	for _, v := range xs {
		if v != x {
			out = append(out, v)
		}
	}
	return out
}

func (g *GameState) ChooseMusik(player PlayerID, index int) error {
	if g.Phase != PhaseTalonExchange {
		return PhaseError("not in talon exchange phase")
	}
	if g.Declarer == nil || *g.Declarer != player {
		return errors.New("only declarer may choose musik")
	}
	if index < 0 || index >= len(g.Deal.Musiks) {
		return fmt.Errorf("invalid musik index")
	}
	chosen := g.Deal.Musiks[index]
	var unchosenCards []Card
	for i := range g.Deal.Musiks {
		if i != index {
			unchosenCards = append(unchosenCards, g.Deal.Musiks[i]...)
		}
	}
	g.Deal.TableCards = append([]Card{}, unchosenCards...)
	g.Deal.Hands[player] = append(g.Deal.Hands[player], chosen...)
	g.Deal.Musiks = nil
	return nil
}

func (g *GameState) Discard(player PlayerID, cards []Card) error {
	if g.Phase != PhaseTalonExchange {
		return PhaseError("not in talon exchange phase")
	}
	if g.Declarer == nil || *g.Declarer != player {
		return errors.New("only declarer may discard")
	}
	if len(cards) != 2 {
		return fmt.Errorf("must discard exactly 2 cards")
	}
	hand := g.Deal.Hands[player]
	for _, c := range cards {
		idx, ok := indexOfCard(hand, c)
		if !ok {
			return fmt.Errorf("card not in hand")
		}
		hand = append(hand[:idx], hand[idx+1:]...)
		g.Deal.TableCards = append(g.Deal.TableCards, c)
	}
	g.Deal.Hands[player] = hand
	g.Play.RemainingCards = g.Params.HandCards * len(g.Params.Players)
	g.Play.CurrentTrick = Trick{Leader: player}
	g.Phase = PhasePlay
	return nil
}

func indexOfCard(cards []Card, target Card) (int, bool) {
	for i, c := range cards {
		if c.Suit == target.Suit && c.Rank == target.Rank {
			return i, true
		}
	}
	return -1, false
}

func (g *GameState) PlayCard(player PlayerID, card Card, announceMarriage bool) error {
	if g.Phase != PhasePlay {
		return PhaseError("not in play phase")
	}
	if player != g.CurrentTurnPlayer() {
		return fmt.Errorf("not %s's turn", player)
	}
	hand := g.Deal.Hands[player]
	idx, ok := indexOfCard(hand, card)
	if !ok {
		return fmt.Errorf("card not in hand")
	}
	if len(g.Play.CurrentTrick.Plays) == 0 {
		g.Play.CurrentTrick.LedSuit = &card.Suit
		g.Play.CurrentTrick.Leader = player
		if announceMarriage {
			if (card.Rank == King || card.Rank == Queen) && holdsOtherKQ(hand, card) {
				s := card.Suit
				g.Play.Trump = &s
				g.Scores.DealPoints[player] += MarriageValue(s)
				g.Play.CurrentTrick.Plays = append(g.Play.CurrentTrick.Plays, Play{Player: player, Card: card, AnnouncedMarriage: &s})
			} else {
				return fmt.Errorf("invalid marriage announcement")
			}
		} else {
			g.Play.CurrentTrick.Plays = append(g.Play.CurrentTrick.Plays, Play{Player: player, Card: card})
		}
	} else {
		if !canFollow(g, player, card) {
			return fmt.Errorf("must follow suit or rules violated")
		}
		g.Play.CurrentTrick.Plays = append(g.Play.CurrentTrick.Plays, Play{Player: player, Card: card})
	}
	// remove card from hand
	g.Deal.Hands[player] = append(hand[:idx], hand[idx+1:]...)
	if len(g.Play.CurrentTrick.Plays) == len(g.Params.Players) {
		winner, trickPoints := g.resolveTrick()
		g.Scores.DealPoints[winner] += trickPoints
		g.Play.LastTrickWinner = &winner
		g.Play.CompletedTricks = append(g.Play.CompletedTricks, g.Play.CurrentTrick)
		g.Play.CurrentTrick = Trick{Leader: winner}
		g.Play.RemainingCards -= len(g.Params.Players)
		if g.Play.RemainingCards == 0 {
			for _, c := range g.Deal.TableCards {
				g.Scores.DealPoints[winner] += PointsFor(c.Rank)
			}
			g.Phase = PhaseScoring
			g.FinalizeScoring()
		}
	}
	return nil
}

// CurrentTurnPlayer returns the player whose turn it is within the current trick.
func (g *GameState) CurrentTurnPlayer() PlayerID {
	leader := g.Play.CurrentTrick.Leader
	// find leader index
	leaderIdx := 0
	for i, p := range g.Params.Players {
		if p == leader {
			leaderIdx = i
			break
		}
	}
	// next player = leader + number of plays % number of players
	idx := (leaderIdx + len(g.Play.CurrentTrick.Plays)) % len(g.Params.Players)
	return g.Params.Players[idx]
}

func holdsOtherKQ(hand []Card, played Card) bool {
	needRank := King
	if played.Rank == King {
		needRank = Queen
	}
	for _, c := range hand {
		if c.Suit == played.Suit && c.Rank == needRank {
			return true
		}
	}
	return false
}

func canFollow(g *GameState, player PlayerID, card Card) bool {
	led := g.Play.CurrentTrick.LedSuit
	// if no led suit, any card is allowed
	if led == nil {
		return true
	}
	hand := g.Deal.Hands[player]

	if card.Suit != *led && hasCardOfSuit(hand, *led) {
		// must follow suit if able to
		return false
	}

	return true
}

func hasCardOfSuit(hand []Card, s Suit) bool {
	for _, c := range hand {
		if c.Suit == s {
			return true
		}
	}
	return false
}

func (g *GameState) resolveTrick() (PlayerID, int) {
	led := *g.Play.CurrentTrick.LedSuit
	trump := g.Play.Trump
	bestIdx := 0
	for i := 1; i < len(g.Play.CurrentTrick.Plays); i++ {
		if firstCardBetter(g.Play.CurrentTrick.Plays[i].Card, g.Play.CurrentTrick.Plays[bestIdx].Card, led, trump) {
			bestIdx = i
		}
	}
	g.Play.CurrentTrick.WinningPlayIndex = bestIdx
	winner := g.Play.CurrentTrick.Plays[bestIdx].Player
	pts := 0
	for _, p := range g.Play.CurrentTrick.Plays {
		pts += PointsFor(p.Card.Rank)
	}
	return winner, pts
}

func firstCardBetter(a, b Card, led Suit, trump *Suit) bool {
	if trump != nil {
		if a.Suit == *trump && b.Suit != *trump {
			return true
		}
		if b.Suit == *trump && a.Suit != *trump {
			return false
		}
		if a.Suit == *trump && b.Suit == *trump {
			return firstCardRankGreater(a.Rank, b.Rank)
		}
	}
	if a.Suit == led && b.Suit != led {
		return true
	}
	if b.Suit == led && a.Suit != led {
		return false
	}
	if a.Suit == led && b.Suit == led {
		return firstCardRankGreater(a.Rank, b.Rank)
	}
	return false
}

func firstCardRankGreater(a, b Rank) bool {
	order := []Rank{Nine, Jack, Queen, King, Ten, Ace}
	idx := func(r Rank) int {
		for i, v := range order {
			if v == r {
				return i
			}
		}
		return -1
	}
	return idx(a) > idx(b)
}

func (g *GameState) ScoresView() ScoreState { return g.Scores }

// HighestBid returns the highest non-pass bid value from the auction.
func (g *GameState) HighestBid() int {
	high := 0
	for _, b := range g.Auction.Bids {
		if !b.Pass && b.Value > high {
			high = b.Value
		}
	}
	return high
}

// FinalizeScoring applies scoring settlement based on declarer success or failure and advances phase to HandEnd.
func (g *GameState) FinalizeScoring() error {
	if g.Phase != PhaseScoring {
		return PhaseError("not in scoring phase")
	}
	if g.Declarer == nil {
		return fmt.Errorf("no declarer set")
	}
	declarer := *g.Declarer
	bid := g.HighestBid()
	// ensure maps are initialized for all players
	for _, p := range g.Params.Players {
		if _, ok := g.Scores.DealPoints[p]; !ok {
			g.Scores.DealPoints[p] = 0
		}
		if _, ok := g.Scores.Cumulative[p]; !ok {
			g.Scores.Cumulative[p] = 0
		}
	}
	dp := g.Scores.DealPoints[declarer]
	if dp >= bid {
		// when declarer wins, add bid to cumulative
		g.Scores.Cumulative[declarer] += bid
	} else {
		g.Scores.Cumulative[declarer] -= bid
	}
	for _, p := range g.Params.Players {
		if p == declarer {
			continue
		}
		g.Scores.Cumulative[p] += g.Scores.DealPoints[p]
	}
	g.Phase = PhaseHandEnd
	return nil
}

// CurrentLeader returns the player who leads the current trick.
func (g *GameState) CurrentLeader() PlayerID { return g.Play.CurrentTrick.Leader }

// LegalBids returns basic legal bid options for the current auction turn.
// Always includes 0 (pass). Returns next minimum raise as a convenience.
func (g *GameState) LegalBids(player PlayerID) []int {
	if g.Phase != PhaseAuction || g.Auction.CurrentLeader != player {
		return nil
	}
	high := 0
	for _, b := range g.Auction.Bids {
		if !b.Pass && b.Value > high {
			high = b.Value
		}
	}
	next := g.Params.MinBid
	if high > 0 {
		next = high + g.Params.MinRaise
	}
	return []int{0, next}
}

// LegalPlays returns the set of cards a player may legally play now.
func (g *GameState) LegalPlays(player PlayerID) []Card {
	if g.Phase != PhasePlay {
		return nil
	}
	hand := g.Deal.Hands[player]
	// If first to play in trick, any card is legal
	if len(g.Play.CurrentTrick.Plays) == 0 {
		return append([]Card(nil), hand...)
	}
	led := g.Play.CurrentTrick.LedSuit
	haveLed := hasCardOfSuit(hand, *led)
	if haveLed {
		var out []Card
		for _, c := range hand {
			if c.Suit == *led {
				out = append(out, c)
			}
		}
		return out
	}
	// No led suit in hand: free to play any card (including trump); no overtrump requirement
	return append([]Card(nil), hand...)
}

func (g *GameState) IsWinningGame() (bool, PlayerID) {
	for playerId, points := range g.Scores.Cumulative {
		if points >= g.Params.MaxGamePoints {
			return true, playerId
		}
	}
	return false, ""
}
