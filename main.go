package main

import (
	"flag"
	"fmt"
	rand "github.com/rwtodd/Go.Rand/xoroshiro"
	gorand "math/rand"
	"time"
)

// a custom reverse insertion sort... because the built-in one is sloooow.
func mySort(data []uint8) {
	b := len(data)
	for i := 1; i < b; i++ {
		for j := i; j > 0 && data[j] > data[j-1]; j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}
}

// a function to shuffle a []uint8. This is another case where
// Go falls down for not having genrics.
func shuffle(vals []uint8, rnd *rand.Rand) {
	for n := len(vals); n > 0; n-- {
		randIndex := int(rnd.Int32n(int32(n)))
		vals[n-1], vals[randIndex] = vals[randIndex], vals[n-1]
	}
}

const (
	qlen = 64 // the length of the queue the players use
)

// A player is represented by a circular queue of `qlen` cards.
// I picked 64 to keep the memory aligned.
type player struct {
	cards [qlen]uint8 // the cards held
	rIdx  int         // the read index into cards
	wIdx  int         // the write index into cards
}

// Set up the player for then next game. Give them their
// cards and prepare to start drawing them.
func (rg *player) Reset(cards []uint8) {
	rg.wIdx = copy(rg.cards[:], cards)
	rg.rIdx = 0
}

// Alive tells if the player has any more cards in hand.
func (rg *player) Alive() bool {
	return (rg.rIdx != rg.wIdx)
}

// DrawCard pulls the next card from the player's hand, or
// 0 if there are no more cards.  Zero is chosen because
// it will always lose.
func (rg *player) DrawCard() uint8 {
	if rg.rIdx == rg.wIdx {
		return 0
	}
	answer := rg.cards[rg.rIdx]
	rg.rIdx++
	if rg.rIdx == qlen {
		rg.rIdx = 0
	}
	return answer
}

// Accept adds cards to the hand.
func (rg *player) Accept(winnings []uint8) {
	n := copy(rg.cards[rg.wIdx:], winnings)
	if n < len(winnings) {
		copy(rg.cards[:], winnings[n:])
	}
	rg.wIdx += len(winnings)
	if rg.wIdx > qlen {
		rg.wIdx -= qlen
	}
}

// makeDeck creates a 52-card deck, with values from
// 2 to 14, where 11=J, 12=Q, 13=K, 14=Ace since Ace
// is high in this game.
func makeDeck() []uint8 {
	deck := make([]uint8, 52)
	idx := 0
	for j := 0; j < 4; j++ {
		for i := 2; i <= 14; i++ {
			deck[idx] = uint8(i)
			idx++
		}
	}
	return deck
}

// playGame plays a single game of "War" high-card,
// returning 1 if player 2 won, and 0 otherwise.
// To avoid threading issues, it has to take the prng
// as an argument.
func playGame(p1 *player, p2 *player, rnd *rand.Rand) int {
	var wins [qlen]uint8
	for p1.Alive() && p2.Alive() {
		c1, c2 := p1.DrawCard(), p2.DrawCard()
		wins[0], wins[1] = c1, c2
		winIdx := 2
		for c1 == c2 && p1.Alive() {
			wins[winIdx], wins[winIdx+1] = p1.DrawCard(), p2.DrawCard()
			c1, c2 = p1.DrawCard(), p2.DrawCard()
			wins[winIdx+2], wins[winIdx+3] = c1, c2
			winIdx = winIdx + 4
		}
		winnings := wins[:winIdx]
		if c1 > c2 {
			shuffle(winnings, rnd)
			p1.Accept(winnings)
		} else {
			mySort(winnings)
			p2.Accept(winnings)
		}
	}

	if p2.Alive() {
		return 1
	}
	return 0
}

// playN plays `n` games of "War" high-card. It
// writes the number of times player 2 won to the
// given `answer` channel.
func playN(n int, answer chan int) {
	rnd := rand.New(gorand.Uint64(), gorand.Uint64())
	deck := makeDeck()
	p1, p2 := &player{}, &player{}
	wins := 0
	for i := 0; i < n; i++ {
		shuffle(deck, rnd)
		p1.Reset(deck[:26])
		p2.Reset(deck[26:])
		wins += playGame(p1, p2, rnd)
	}
	answer <- wins
}

var (
   nGames = flag.Int("games", 10000, "number of games to play")
   nProcs = flag.Int("procs", 4, "number of concurrent games to play")
)
func main() {
	flag.Parse()

	gamesPerCore := *nGames / *nProcs
	totalGames := gamesPerCore *  *nProcs

	fmt.Printf("Playing %d games each on %d cores.\n", gamesPerCore, *nProcs)

	outputCh := make(chan int, *nProcs)
	gorand.Seed(time.Now().Unix())

	for i := 0; i < *nProcs; i++ {
		go playN(gamesPerCore, outputCh)
	}

	total := 0
	for i := 0; i < *nProcs; i++ {
		total += <-outputCh
	}
	close(outputCh)

	fmt.Printf("Smart player wins: %d games out of %d (%0.2f%%)\n",
		total,
		totalGames,
		(float64(total*100) / float64(totalGames)))
}
