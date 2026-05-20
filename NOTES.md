```
   __     __               __
  / /__  / /____  ________/ /   ___ ____ _
 / / _ \/ __/ _ \/ __/ __/ _ \_/ _ `/ _ `/
/_/_//_/\__/\___/_/  \__/_//_(_)_, /\_, /
                              /___//___/
```

**features**

+ waiting rooms
  * "join waiting room" as CTA
  * no need to create game, invite players etc.
  * notification when enough players joined
  * new waiting list when new game starts

+ no login required
  * minimize friction to get into game
  * auth: magic passphrase or name?
  * proper auth can be configured later (when more important)
    - nostr
    - lnurl
    - email

+ hand-crafted pixel art animations

**flow**

1. user visits site
2. user buys ticket to join waiting room
3.1 if waiting room reached time/player threshold: start new game
3.2 else: notify user that we're waiting for other players now (refund after timeout)
