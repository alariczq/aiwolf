# aiwolf

AI agents play Werewolf (狼人杀) — powered by [Eino](https://github.com/cloudwego/eino) framework, Claude & Gemini. Watch LLMs deceive, deduce, and vote each other out.

## Features

- Each player is an independent AI agent with its own reasoning
- Supports Claude and Gemini models (mix and match)
- Full Werewolf rules: night kills, investigations, voting, sheriff election
- Web UI for watching games unfold
- AI-generated narration for immersive storytelling

## Roles

| Role | Team | Ability |
|------|------|---------|
| Werewolf | Wolf | Kill one player each night |
| Wolf King | Wolf | Self-explode and take one player down |
| Wolf Beauty | Wolf | Charm a player who dies if she dies |
| Seer | Village | Investigate one player's alignment each night |
| Witch | Village | One heal potion, one poison potion |
| Hunter | Village | Shoot one player upon death |
| Guard | Village | Protect one player from wolf kill each night |
| Knight | Village | Duel a player; if wolf, they die; if not, knight dies |
| Idiot | Village | Survives first vote but loses voting rights |
| Villager | Village | No special ability |

## Quick Start

```bash
# Clone
git clone https://github.com/alaric/aiwolf.git
cd aiwolf

# Configure API keys
cp .env.example .env
# Edit .env with your API keys

# Run CLI game
go run ./cmd/werewolf

# Run Web UI
go run ./cmd/web
```

## Configuration

Set at least one API key in `.env`:

```
CLAUDE_API_KEY=your-claude-api-key
GEMINI_API_KEY=your-gemini-api-key
```

## Requirements

- Go 1.21+
- Claude API key and/or Gemini API key

## License

MIT
