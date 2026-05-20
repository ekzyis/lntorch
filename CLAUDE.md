# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

lntorch.gg - Lightning Network elimination game.

## Game Rules

- Players join a waiting room by buying a ticket (Lightning)
- Game starts at 10 players, or 2+ players after a few hours
- Refund after 24h if game doesn't start (hold invoices)
- Each player has a torch (timer, e.g. 24h)
- Pay to "pass the torch" (reset your timer)
- Torch burns out = eliminated
- Last player standing wins the pot

## MVP

Game loop only, raw HTML + htmx, no design.
Cookie = player ID (set on join), no accounts.

## TODOs

- [x] Go server with landing page
  - "Join Waiting Room" button
  - Sets cookie (player ID) on click

## Stack

Go, net/http, SQLite, htmx, LND (gRPC)

## Later

- Push notifications with auto-pay (likely requires custodial balance)

## Design

Simple, (animated) pixel art, maybe anachronistic.

## Git Commits

Author: ekzyis <ramdip.singhgill@gmail.com>
Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
