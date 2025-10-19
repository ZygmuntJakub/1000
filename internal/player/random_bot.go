package player

import (
	"math/rand"
	"strconv"

	"github.com/ZygmuntJakub/1000/internal/engine"
)

type RandomBot struct {
	BotName string
}

func (b *RandomBot) Name() string {
	if b.BotName == "" {
		b.BotName = "RandomBot_" + strconv.Itoa(rand.Intn(100))
	}
	return b.BotName
}

func (b *RandomBot) MakeBidDecision(cards *[]engine.Card, legalBids []int) (int, error) {
	return legalBids[rand.Intn(len(legalBids))], nil
}

func (b *RandomBot) ChooseMusik(numberOfMusiks int) (int, error) {
	return rand.Intn(numberOfMusiks), nil
}

func (b *RandomBot) ChooseDiscardCards(cards *[]engine.Card, numberOfCards int) (*[]engine.Card, error) {
	out := make([]engine.Card, numberOfCards)
	for i := range numberOfCards {
		out[i] = (*cards)[i]
	}
	return &out, nil
}

func (b *RandomBot) PlayCard(cards *[]engine.Card, playState *engine.PlayState) (*engine.Card, bool, error) {
	return &(*cards)[rand.Intn(len(*cards))], false, nil
}

func NewRandomBot() Player {
	return &RandomBot{}
}
