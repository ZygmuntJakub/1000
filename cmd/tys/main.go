package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ZygmuntJakub/1000/internal/engine"
	"github.com/ZygmuntJakub/1000/internal/player"
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
	bot1 := player.NewRandomBot()
	bot2 := player.NewRandomBot()
	players := []engine.PlayerID{engine.PlayerID(bot1.Name()), engine.PlayerID(bot2.Name())}
	playerToBot := map[engine.PlayerID]player.Player{
		players[0]: bot1,
		players[1]: bot2,
	}
	params := engine.GameParams{
		Players:     players,
		MinBid:      100,
		MinRaise:    10,
		HandCards:   10,
		MusiksCount: 2,
		MusikSize:   2,
	}
	var lastPlayPoints map[engine.PlayerID]int = nil
	numberOfPlays := 0
	for {
		g := engine.NewGame(params, players[0], players, lastPlayPoints)
		fmt.Printf("Phase=%v, Dealer=%s, Players=%v\n", g.Phase, g.Dealer, players)

		// Deterministic deal (same as tests basic split):
		deck := makeDeck()
		shuffleDeck(deck)
		h1 := append([]engine.Card{}, deck[:10]...)
		h2 := append([]engine.Card{}, deck[10:20]...)
		m1 := append([]engine.Card{}, deck[20:22]...)
		m2 := append([]engine.Card{}, deck[22:24]...)
		if err := g.SetDealtCards(map[engine.PlayerID][]engine.Card{players[0]: h1, players[1]: h2}, [][]engine.Card{m1, m2}); err != nil {
			fmt.Printf("SetDealtCards error: %v\n", err)
			return
		}
		fmt.Printf("Deal set. Phase=%v, Auction leader=%s\n", g.Phase, g.Auction.CurrentLeader)

		for g.Phase == engine.PhaseAuction {
			currentBot := playerToBot[g.Auction.CurrentLeader]
			hand := append([]engine.Card{}, g.Deal.Hands[g.Auction.CurrentLeader]...)
			bid, err := currentBot.MakeBidDecision(&hand, g.LegalBids(g.Auction.CurrentLeader))
			if err != nil {
				fmt.Printf("bid error: %v\n", err)
				return
			}
			if err := g.PlaceBid(g.Auction.CurrentLeader, bid); err != nil {
				fmt.Printf("bid error: %v\n", err)
				return
			}
		}

		fmt.Printf("Auction done. Declarer=%s, Value=%d Phase=%v\n", *g.Declarer, g.HighestBid(), g.Phase)

		currentBot := playerToBot[*g.Declarer]
		// Declarer chooses first musik, discard two smallest by rank (stable)
		musikIndex, err := currentBot.ChooseMusik(len(g.Deal.Musiks))
		if err != nil {
			fmt.Printf("choose musik: %v\n", err)
			return
		}
		if err := g.ChooseMusik(*g.Declarer, musikIndex); err != nil {
			fmt.Printf("choose musik: %v\n", err)
			return
		}
		// Choose two discard cards deterministically (lowest two ranks, by suit then rank)
		hand := append([]engine.Card{}, g.Deal.Hands[*g.Declarer]...)
		discards, err := currentBot.ChooseDiscardCards(&hand, 2)
		if err != nil {
			fmt.Printf("choose discard cards: %v\n", err)
			return
		}
		if err := g.Discard(*g.Declarer, *discards); err != nil {
			fmt.Printf("discard: %v\n", err)
			return
		}
		fmt.Printf("Declarer took musik[0] and discarded %v, %v. Phase=%v\n", (*discards)[0], (*discards)[1], g.Phase)

		// Auto-play the rest of the hand:
		for g.Phase == engine.PhasePlay {
			currentPlayer := g.CurrentTurnPlayer()
			currentBot := playerToBot[currentPlayer]
			legal := g.LegalPlays(currentPlayer)

			cards, marriage, err := currentBot.PlayCard(&legal, &g.Play)
			if err != nil {
				fmt.Printf("play card: %v\n", err)
				return
			}
			if err := g.PlayCard(currentPlayer, *cards, marriage); err != nil {
				fmt.Printf("play card: %v\n", err)
				return
			}
		}

		// Print final results
		fmt.Printf("Hand finished. Phase=%v\n", g.Phase)
		fmt.Printf("Deal points: %v\n", g.Scores.DealPoints)
		fmt.Printf("Cumulative: %v\n", g.Scores.Cumulative)
		if isWinning, player := g.IsWinningGame(); isWinning {
			fmt.Printf("Player %s wins!\n", player)
			fmt.Printf("Number of plays: %d\n", numberOfPlays)
			return
		}
		lastPlayPoints = g.Scores.Cumulative
		numberOfPlays++
		time.Sleep(1 * time.Second)
	}
}
