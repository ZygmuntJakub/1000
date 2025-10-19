//go:generate stringer -type=Phase,Suit,Rank -linecomment

package engine

// Suit represents a card suit.
type Suit int

const (
	Spades   Suit = iota // ♠️
	Clubs                // ♣️
	Diamonds             // ♦️
	Hearts               // ♥️
)

// Rank represents a card rank.
type Rank int

const (
	Nine  Rank = iota // 9
	Jack              // J
	Queen             // Q
	King              // K
	Ten               // 10
	Ace               // A
)

// Card represents a playing card.
type Card struct {
	Suit Suit
	Rank Rank
}

// PlayerID identifies a player.
type PlayerID string

// Phase represents the game phase.
type Phase int

const (
	PhaseInit          Phase = iota // init
	PhaseDeal                       // deal
	PhaseAuction                    // auction
	PhaseTalonExchange              // talon exchange
	PhasePlay                       // play
	PhaseScoring                    // scoring
	PhaseHandEnd                    // hand end
)

// MarriageValue returns the marriage points for a suit.
func MarriageValue(s Suit) int {
	switch s {
	case Spades:
		return 40
	case Clubs:
		return 60
	case Diamonds:
		return 80
	case Hearts:
		return 100
	default:
		return 0
	}
}

// PointsFor returns points of a rank for trick scoring.
func PointsFor(r Rank) int {
	switch r {
	case Ace:
		return 11
	case Ten:
		return 10
	case King:
		return 4
	case Queen:
		return 3
	case Jack:
		return 2
	default:
		return 0
	}
}

// Play represents a single play in a trick.
type Play struct {
	Player            PlayerID
	Card              Card
	AnnouncedMarriage *Suit // set when leader announces marriage of this suit
}

// Trick holds the state of a trick.
type Trick struct {
	Leader           PlayerID
	Plays            []Play
	LedSuit          *Suit
	WinningPlayIndex int
}

// AuctionBid represents a single auction action.
type AuctionBid struct {
	Player PlayerID
	Value  int // 0 means pass
	Pass   bool
}

// AuctionState holds current auction info.
type AuctionState struct {
	Bids          []AuctionBid
	ActivePlayers []PlayerID
	CurrentLeader PlayerID
	MinRaise      int
}

// DealState holds dealt hands and musiks.
type DealState struct {
	Hands      map[PlayerID][]Card
	Musiks     [][]Card // two musiks (2 cards each) in 2P mode
	TableCards []Card   // unchosen musik + declarer discards
}

// PlayState tracks play progress.
type PlayState struct {
	CurrentTrick    Trick
	CompletedTricks []Trick
	Trump           *Suit
	RemainingCards  int
	LastTrickWinner *PlayerID
}

// ScoreState holds per-deal and cumulative scores.
type ScoreState struct {
	DealPoints map[PlayerID]int
	Cumulative map[PlayerID]int
}

// GameParams parameterizes game rules.
type GameParams struct {
	Players       []PlayerID
	MinBid        int
	MinRaise      int
	HandCards     int
	MusiksCount   int
	MusikSize     int
	MaxGamePoints int
}

// GameState is the root state container.
type GameState struct {
	Phase    Phase
	Params   GameParams
	Dealer   PlayerID
	Declarer *PlayerID

	Deal    DealState
	Auction AuctionState
	Play    PlayState
	Scores  ScoreState
}
