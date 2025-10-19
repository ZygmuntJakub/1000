package player

import "github.com/ZygmuntJakub/1000/internal/engine"

type Player interface {
	Name() string
	MakeBidDecision(*[]engine.Card, []int) (int, error)
	ChooseMusik(int) (int, error)
	ChooseDiscardCards(*[]engine.Card, int) (*[]engine.Card, error)
	PlayCard(*[]engine.Card, *engine.PlayState) (*engine.Card, bool, error)
}

type PlayerFactory func() Player
