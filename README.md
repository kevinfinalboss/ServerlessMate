<p align="center">
  <img src="frontend/public/logo.png" alt="ServerlessMate" width="180" />
</p>

<h1 align="center">ServerlessMate</h1>

<p align="center">
  A real-time multiplayer chess platform, 100% serverless on AWS.
</p>

<p align="center">
  <a href="https://serverlessmate.kevindev.com.br">serverlessmate.kevindev.com.br</a>
</p>

---

## What it is

ServerlessMate is an online chess site where two players join a queue and get matched automatically by time control and rating band, or play against an AI with two difficulty levels. The entire game happens in real time over WebSocket: moves, chat, draw offers, resignation, and rematches, with a real chess clock ticking server-side.

There is no traditional server running at any point. Every action in the game (joining the queue, moving a piece, accepting a friend request) is an isolated Lambda function, triggered either by a WebSocket message or by an event from the database itself.

## Features

- Human vs. human matches with automatic matchmaking by time control and rating (Elo, K=32)
- Play against an AI with two levels (heuristic and minimax with alpha-beta pruning), with LLM-generated commentary (Amazon Bedrock) after each move
- Per-player chess clock, with loss on time
- Resignation, draw offer/acceptance, and one-click rematch
- Reconnection grace period: dropping the connection isn't an immediate loss
- Spectator mode for ongoing games
- Global, weekly, and monthly leaderboards
- Match history with PGN replay
- Friends system (request, accept, block) with a friends list
- Player profile with stats, configurable visibility (public or friends-only), and optional fields (country, birth date, GitHub, LinkedIn)
- Real authentication via Amazon Cognito (SRP, the password never travels over the wire) and a guest session with no signup required
- English and Portuguese UI

## How it's built

**Backend** — Go, compiled for `provided.al2023`/arm64. Every action in the system is its own Lambda (`cmd/lambda/*`), with no persistent server and no HTTP framework. All domain logic and AWS integration lives in `internal/` (`store`, `game`, `ai`, `auth`, `rating`, `ws`), always behind interfaces, so it can be unit tested without real infrastructure.

- API Gateway WebSocket as the entry point, one route per action
- DynamoDB as the only database, no database server at all
- Amazon Bedrock for the AI's move commentary
- Amazon Cognito for authentication
- CloudFront + S3 serving the static frontend

**Frontend** — React 19 with TypeScript and Vite, a single WebSocket connection shared across the whole session via Context. Board rendered with `react-chessboard`/`chess.js`, routing with `react-router-dom`, Tailwind CSS for styling.

**Infrastructure** — Terraform, around 30 provisioned resources: Lambdas, API Gateway WebSocket with a custom domain, DynamoDB tables, Cognito User Pool, S3 bucket + CloudFront, DNS via Cloudflare.

## Project structure

```
cmd/lambda/       one folder per Lambda function (handler + main)
internal/         domain logic and integrations, testable without AWS
frontend/         React + TypeScript SPA
terraform/        all infrastructure as code
Makefile          build, packaging, and deploy for Lambdas and frontend
```

## Running locally

Backend (build and tests):

```
go build ./...
go test ./... -race -cover
```

Frontend:

```
cd frontend
npm install
npm run dev
```

Copy `frontend/.env.example` to `frontend/.env` and fill it in with the WebSocket endpoint and Cognito credentials from your own infrastructure.

## Deploy

The `Makefile` builds the Lambda binaries, packages them as `.zip`, and uploads them to S3; Terraform points at that version and provisions/updates the resources.

```
make deploy-lambdas VERSION=v0.1.0
make deploy-frontend
```

Terraform is applied manually from `terraform/`.
