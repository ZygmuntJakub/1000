# Tysiąc Engine State-Machine Specification (Go)

## Overview
Deterministic, pure engine managing rules, validation, and scoring for 2–4 players. CLI is a thin shell. No randomness beyond deal input.

## High-Level Flow
States are explicit; transitions are driven by validated actions.

1. Init
2. Deal
3. Auction
4. TalonExchange
5. Play
6. Scoring
7. HandEnd

## Core Types (Go-oriented)
- `Suit`: enum { Spades, Clubs, Diamonds, Hearts }
- `Rank`: enum { Nine, Jack, Queen, King, Ten, Ace }
- `Card`: `{ Suit, Rank }`
- `PlayerID`: string or small int
- `Trick`: `{ Leader PlayerID, Plays []Play, LedSuit *Suit, WinningPlayIndex int }`
- `Play`: `{ Player PlayerID, Card Card, AnnouncedMarriage *Suit }`
- `MarriageValue(s Suit) int`: ♠40, ♣60, ♦80, ♥100
- `PointsFor(rank Rank) int`: A11, 1010, K4, Q3, J2, 90
- `AuctionBid`: `{ Player PlayerID, Value int, Pass bool }`
- `AuctionState`: `{ Bids []AuctionBid, ActivePlayers []PlayerID, CurrentLeader PlayerID, MinRaise int (>=10) }`
- `DealState`: `{ Hands map[PlayerID][]Card, Musiks [][]Card, TableCards []Card }`
- `PlayState`: `{ CurrentTrick Trick, CompletedTricks []Trick, Trump *Suit, RemainingCards int, LastTrickWinner *PlayerID }`
- `ScoreState`: `{ DealPoints map[PlayerID]int, Cumulative map[PlayerID]int }`
- `GameParams`: `{ Players []PlayerID, MinBid int(=100), MinRaise int(=10), HandCards int(=10), MusiksCount int(=2), MusikSize int(=2) }`
- `GameState`:
  ```
  type GameState struct {
    Phase        Phase
    Params       GameParams
    Dealer       PlayerID
    Declarer     *PlayerID
    Deal         DealState
    Auction      AuctionState
    Play         PlayState
    Scores       ScoreState
  }
  ```
- `Phase`: enum { Init, Deal, Auction, TalonExchange, Play, Scoring, HandEnd }

## State Transitions and Actions

### Init -> Deal
- Input: `StartGame(Players, Dealer, CumulativeScores)`
- Output: `Phase=Deal` with empty hands awaiting `SetDealtCards` or `DealRandom` (CLI may supply RNG).

### Deal
- Action A: `SetDealtCards(hands map[PlayerID][]Card, musiks [][]Card)`
  - Validates (2 players default): |hands[p]| == 10 for each player; len(musiks) == 2; each musik has len == 2; total 24 unique cards across both hands and musiks; players == 2.
  - Next: `Phase=Auction` with `Auction.ActivePlayers = all` and `CurrentLeader = player after Dealer`.
- Action B (optional): `DealRandom(seed)` for local use; not used by deterministic engine core unless provided.

### Auction
- Action: `PlaceBid(player, value)`
  - Preconditions: player in `ActivePlayers` and it is their turn.
  - Validate:
    - If `value==0`: treated as Pass.
    - Else `value >= MinBid` and `value > highestBid`, and `(value - highestBid) >= MinRaise`.
  - Effect: append bid; if Pass, remove from active.
  - End condition: only one active remains -> `Declarer=that player`, `Phase=TalonExchange`.

### TalonExchange (Musik Exchange for 2P)
- Actions by Declarer only:
  - `ChooseMusik(index int)` -> privately take musiks[index] (2 cards) into hand (hand becomes 12 cards). The unchosen musik remains face-down.
  - `Discard(cards []Card)` -> must discard exactly 2 cards, all owned. Discarded cards are added face-down to `Deal.TableCards` along with the unchosen musik (total 4 face-down table cards).
  - After discard: `Phase=Play` with `Play.CurrentTrick.Leader = Declarer` and `Play.RemainingCards = HandCards * len(Players)`.

### Play (2 players, 10 tricks)
- Turn order: clockwise from `CurrentTrick.Leader`.
- Action: `PlayCard(player, card, announceMarriage bool)`
  - Validate ownership: card in player hand.
  - Enforce follow-suit: if led suit set, player must play that suit when possible.
  - If void in led suit: player may play any card. If trump led and has trump, must play trump.
  - No must-beat rule; no overtrump requirement.
  - Marriage announcement:
    - Allowed only by the trick leader.
    - `announceMarriage==true` only if played card is K or Q of suit S, and player also holds the other of K/Q of S.
    - Effect when valid: `Play.Trump = &S` and add `MarriageValue(S)` to leader’s deal points immediately.
  - Trick formation:
    - If first play of trick: set `LedSuit = card.Suit` (trump is irrelevant to led suit).
    - Append play; when trick has N plays (N = number of players): resolve winner.
  - Resolve trick winner:
    - Determine highest trump if any trump cards present; else highest of `LedSuit`.
    - Rank order: A > 10 > K > Q > J > 9.
    - Winner gets sum of card points of the trick added to deal points.
    - Next leader = trick winner; record `Play.LastTrickWinner` when it changes; start next trick.
  - End of Play: after `HandCards` completed tricks (10 in 2P default) -> move 4 face-down `Deal.TableCards` to `LastTrickWinner`'s captured pile and points, then `Phase=Scoring`.

### Scoring
- Declarer success check:
  - `DeclarerDealPoints = trickPoints(declarer) + marriagePoints(declarer)`
  - Success if `DeclarerDealPoints >= highestBid`.
- Settlement:
  - If success: `Cumulative[Declarer] += DeclarerDealPoints`.
  - If failure: `Cumulative[Declarer] -= highestBid`.
  - For each defender D: `Cumulative[D] += DealPoints[D]`.
- Next: `Phase=HandEnd`.

### HandEnd
- Check game end:
  - If any `Cumulative[p] >= 1000`: game ends. If multiple cross 1000 simultaneously, tie stands.
  - Else rotate Dealer clockwise and transition to `Deal` for next hand.

## Validation Rules Summary
- Card uniqueness across hands + talon.
- Turn order enforced by phase and trick leader.
- Follow-suit enforced; if trump is led and player has trump, must play trump.
- No forced overtrump; no forced beat.
- Marriage only by leader, only when holding both K and Q of suit announced, and only on the trick they lead with K or Q of that suit.
- Multiple marriages allowed across deal; each announcement instantly changes trump to that suit.
- No kontra/re; no blind bids.
- 2P defaults: two musiks (2 cards each); declarer privately chooses one, discards 2; 4 table cards awarded to last trick winner at end of play.

## Determinism and Side Effects
- All mutations occur via validated actions returning a new or mutated `GameState` object.
- Engine is deterministic given `SetDealtCards` or `DealRandom(seed)`.

## Error Handling
- Actions return typed errors for:
  - Wrong phase/turn
  - Illegal bid
  - Illegal card (ownership or follow-suit/trump-led violation)
  - Invalid marriage announcement
  - Invalid discard size/content

## API Sketch (Go)
```go
// Construction
func NewGame(params GameParams, dealer PlayerID, players []PlayerID, cumulative map[PlayerID]int) *GameState

// Dealing
func (g *GameState) SetDealtCards(hands map[PlayerID][]Card, talon []Card) error

// Auction
func (g *GameState) PlaceBid(player PlayerID, value int) error

// Talon
func (g *GameState) TakeTalon(player PlayerID) error
func (g *GameState) Discard(player PlayerID, cards []Card) error

// Play
func (g *GameState) PlayCard(player PlayerID, card Card, announceMarriage bool) error

// Introspection
func (g *GameState) LegalBids(player PlayerID) []int
func (g *GameState) LegalPlays(player PlayerID) []Card
func (g *GameState) CurrentLeader() PlayerID
func (g *GameState) Scores() ScoreState
```

## Test Matrix (essentials)
- **[auction]** minimal raise, pass-out, large bids, invalid raises
- **[talon/musik]**: choose exactly one musik; discard exactly 2; table cards awarded to last-trick winner; invalid discards/choices
- **[play]**: follow-suit enforcement; trump-led enforcement; no-overtrump constraint; illegal marriage attempts; multiple marriages toggling trump; no-trump entire deal
- **[trick win]**: with and without trump; rank order correctness
- **[scoring]**: declarer success vs fail; defenders’ additions; cumulative targets; 1000+ end
- **[players]**: 2, 3, 4 players hand length and trick count validation (2P: 10 tricks by default)
```
