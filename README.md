```
   __     __               __
  / /__  / /____  ________/ /   ___ ____ _
 / / _ \/ __/ _ \/ __/ __/ _ \_/ _ `/ _ `/
/_/_//_/\__/\___/_/  \__/_//_(_)_, /\_, /
                              /___//___/
```

Pay to keep the torch lit. Last player not consumed by the dark wins the pot.

## ✨ features

**💫 fun game**

* only one player has to pay `:sats_per_round:` sats per round ("keep the
  torch lit")
* players have `:game_interval:` time to pay and stay in the game
* if payment is due and timer runs out, player is "consumed by the dark"
* at the end of each round, the torch is passed to the next player
* last player wins everything

_yes, the game being fun is a feature._

**💫 waiting rooms**

* there's always a waiting room you can join
* a waiting room turns into a game every `:game_interval:` if there are at
  least two players
* a waiting room turns immediately into a game when there are `:max_players:`
  players
* (players can vote to start a game early)
* (players can talk shit in the waiting room)
* (anyone can see who's in the waiting room -> username?)

**💫 lightning payments**

* to join a waiting room, a ticket is required
* a ticket costs `:sats_per_round:` sats
* lightning-native refunds if there aren't enough players to start the game
* players need to pay to stay in the game
* payment method: bolt11
* (`payment_metadata` to link invoice and session)

**💫 no login required**

* to minimize friction, no login is required to join a game
* if there's no session cookie yet, the servers sets a cookie with a token
  from a CSPRNG
* we remind user to configure auth when they can lose money ("start winning")
* lightning address, bolt11 as withdrawal methods
* username: ask for lightning address, else generate random one?

**💫 push notifications**

* notify when other player joins waiting room
* notify when game starts
* notify when turn changes

## ⚙️ tech stack

Minimal. For now: net/http, html/template, sqlite3.

github.com/ekzyis/lnpilot for lightning stuff.

TBD: htmx, lnd/cln/phoenixd/cashu

## 🛢️ database

SQLite. Schema is created on startup.

TODO: migrations

**games**

| column     | type      | notes                          |
|------------|-----------|--------------------------------|
| id         | INTEGER   | PRIMARY KEY                    |
| status     | TEXT      | NOT NULL, default `'waiting'`  |
| created_at | TIMESTAMP | default `CURRENT_TIMESTAMP`    |

**players**

| column     | type      | notes                          |
|------------|-----------|--------------------------------|
| id         | INTEGER   | PRIMARY KEY                    |
| session    | TEXT      | UNIQUE NOT NULL (cookie token) |
| created_at | TIMESTAMP | default `CURRENT_TIMESTAMP`    |

**games_players**

| column     | type      | notes                          |
|------------|-----------|--------------------------------|
| game_id    | INTEGER   | REFERENCES games(id)           |
| player_id  | INTEGER   | REFERENCES players(id)         |
| created_at | TIMESTAMP | default `CURRENT_TIMESTAMP`    |

PRIMARY KEY `(game_id, player_id)`.

## 🔗 constants

These are game constants that are still to be determined (TBD). They are
referenced in other parts of this document like this: `:name:`. For now, I have
picked these values:

```
sats_per_round = 100
max_players = 5
game_interval = every hour
```

## 💥 deployment

The server is live at https://lntorch.gg. It is deployed on every push, see
GitHub workflow.

## 🎨 design / philosophy 🏛️

Simple, animated pixel art, maybe anachronistic, also silly, fun, terrible,
while still incredibly complex, sophisticated, ambiguous. The game does not take
itself very seriously. The credits will be my name scrolling down forever. If it
looks so bad it could make someone laugh, put it in the game. If it looks so
good it could make someone cry, put it in the game. If the game is between
cringe and "Why did this make me laugh?", it's great.

Smooth page transitions. There aren't many pages.

This game LOVES you, and shows it to you through its obsession over details it
thinks most wouldn't notice -- but YOU would.

**example ideas with the "right amount of dumb"**

* random events: "the custodian gambled with your money and doubled it!" \
  (and I actually gambled with it)

**Color theme**

TBD

**Animations**

TBD, but ideas:

* dungeon with torch in background of main menu
* lightning strike / strikes torch when paying invoice
* "consumed by the dark" when losing a game, horror vibes
