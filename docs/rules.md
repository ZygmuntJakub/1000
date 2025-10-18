# Tysiąc (Thousand) – Rules Specification (Go CLI version)

## Players
- **[players]**: 2–4 players
- Initial implementation targets 2-player rules; engine remains extensible to 3–4 players.

## Deck
- **[composition]**: 24 cards: ranks 9, J, Q, K, 10, A in each suit (♠, ♣, ♦, ♥)
- **[card point values]** (per trick scoring):
  - A = 11
  - 10 = 10
  - K = 4
  - Q = 3
  - J = 2
  - 9 = 0
- **[deal total]**: 120 points from tricks per deal

## Dealing and Talon (Pula/Musik) — 2 Players
- **[deal size]**: Each player receives 10 cards.
- **[musiks]**: Two separate musiks on the table, each with 2 face-down cards.
- **[who takes musik]**: After the auction, the declarer privately chooses one musik (2 cards) and adds it to hand (now 12 cards). The unchosen musik remains face-down and hidden.
- **[discard]**: Declarer discards exactly 2 cards face-down, returning to 10 cards.
- **[table cards resolution]**: After all tricks, the 4 face-down table cards (unchosen musik + declarer’s 2 discards) are added to the captured pile of the player who won the last trick and count toward that player’s trick points.
- **[tricks]**: With 10 cards per player, 10 tricks are played.

## Auction (Bidding)
- **[turn order]**: Clockwise
- **[opening]**: Minimum opening bid 100
- **[raises]**: Strictly increasing, typically in steps of 10 (engine validates step ≥ 10)
- **[passes]**: Once a player passes, they are out of the auction for the deal
- **[end condition]**: Auction ends when all but one have passed; last bidder is the declarer
- **[limits]**: No explicit maximum bid (no 120 cap)
- **[special calls]**: No kontra/re, no blind bids
- **[compulsory bid]**: None

## Marriages (Meldunki) and Trump
- **[marriage values]** (by suit):
  - ♠ spades = 40
  - ♣ clubs = 60
  - ♦ diamonds = 80
  - ♥ hearts = 100
- **[requirements]**: Must hold both K and Q of the same suit
- **[announcement timing]**: Only when leading a trick by playing either the K or Q of that suit; no pre-announcements
- **[multiple marriages]**: Allowed across the same deal; each can be announced on separate leads
- **[trump behavior]**: Announcing a marriage sets that suit as the trump suit
  - No trump is active at the start of play
  - Trump remains until the end of the deal or until another marriage is announced, which then changes trump to the new suit
  - If no marriages are announced during the deal, there is no trump for that deal

## Play Constraints (Trick-Taking)
- **[lead]**: Declarer leads the first trick
- **[must follow suit]**: Yes. If you have the led suit, you must play it
- **[void in suit]**: If void, you may play any card (including a trump if trump is active); not required to play trump
- **[trump led]**: If trump is led and you have trump, you must follow with trump
- **[overtrumping]**: Not required. You may undertrump if you choose to trump
- **[must beat]**: No requirement to beat a higher card; only follow suit if able
- **[trick winner]**:
  - If any trumps are present in the trick, highest trump wins
  - Else, highest card of the led suit wins
  - Rank order for winning comparison in suit: A > 10 > K > Q > J > 9

## Scoring Within a Deal
- **[trick points]**: Sum of card values from tricks won by a player
- **[marriage points]**: Added immediately upon valid announcement to the player’s deal score
- **[last trick bonus]**: None
- **[deal total]**: 120 trick points across all players; marriage points are additional on top and are not limited by 120
  - In 2-player mode, the 4 table cards are awarded to the last trick winner and their points are included in that player’s trick points.

## Contract Resolution and Settlement
- **[contract target]**: Declarer must achieve at least their bid using trick points + marriage points in the deal
- **[success]**: If declarer meets/exceeds bid, add the actual deal points they scored (trick + marriage) to their cumulative game score
- **[failure]**: If declarer fails, subtract the bid amount from their cumulative score; defenders add their own deal points to their cumulative scores
- **[rounding/caps]**: No rounding and no per-hand cap

## Penalties and Illegal Plays
- **[illegal marriage]**: Not permitted by engine; announcement validated by holding both K and Q
- **[reneging/illegal play]**: Engine enforces legal plays (follow suit, trump-only constraints)
- **[schneider/schwarz]**: Not used

## Game End
- **[goal]**: First player to reach or exceed 1000 cumulative points wins
- **[exact 1000]**: Not required; exceeding is allowed
- **[ties]**: A tie remains a tie

## Notes for 3- and 4-Player Variants
- Engine parameterizes player count (2–4). The current document specifies the 2-player baseline. For 3 players, the classic single 3-card talon applies; for 4 players, additional variants exist and will be documented later.

## Glossary
- **Declarer**: Auction winner
- **Musik/Talon (2P)**: Two 2-card face-down piles; declarer privately takes one pile, discards 2, and the 4 table cards go to the last trick winner
- **Marriage/Meldunek**: Holding K+Q of a suit; announcement on lead grants points and sets trump
