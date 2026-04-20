## Replace the corpus

The bot answers from every `.md` file under the `docs/` directory. These files are embedded into the Go binary at build time, so **you rebuild to change the corpus**.

## Format

Each markdown file should have H2 (`## `) headings. The chunker splits on H2, so each H2 section becomes one retrievable chunk. Long sections are sub-split at paragraph boundaries with overlap.

Good example:

```markdown
# Custom Domains

How to point your own domain at Acme.

## Requirements

- A domain you control.
- Access to your DNS provider.

## Steps

1. Open Settings → Domain.
2. Enter your domain.
3. Add a CNAME record at your DNS provider pointing at `cname.acme.app`.

## Troubleshooting

If verification fails, check that the CNAME is not behind Cloudflare proxy mode.
```

Bad example:

```markdown
# Everything about Acme

Acme is a thing that does things. You can configure it. There's a settings page.
You can also do integrations. Scheduling works. For help contact support.
```

The second won't retrieve well because there are no H2 headings and the content is vague.

## Rewrite tips

- **Name specific screens and fields.** "Open Settings → Domain → Add CNAME" beats "go to the settings page."
- **One topic per file.** A `contact-forms.md` beats a 50-feature mega-doc.
- **Prefer concrete over abstract.** "CNAME at `cname.acme.app`" is useful; "configure DNS appropriately" is not.
- **Keep each file 80–300 lines.** Shorter than that, split too fine; longer, the chunker will hit its overlap fallback.

## Apply your changes

```bash
# Add or edit files under docs/
$EDITOR docs/02-my-feature.md

# Rebuild the image
docker-compose build ragbot

# Restart
docker-compose up -d ragbot
```

On startup, the service hashes the `docs/` directory (SHA-256 over filename + content of every `.md`). If the hash matches the one stored in `ragbot_corpus_version`, it skips re-indexing (50ms no-op). If it differs, it truncates `ragbot_chunks` and re-embeds everything. You'll see this in the startup logs:

```
[ragbot] corpus hash changed, reindexing 9 docs
[ragbot] chunked into 58 pieces, embedding with nomic-embed-text
[ragbot] indexed 58 chunks at hash 206aee0af8d8
```

## Keeping vs replacing the seed docs

You have two options:

1. **Add alongside** — keep the existing `00-getting-started.md` etc. and add your own `10-acme-features.md`, `11-acme-pricing.md`. The bot becomes useful for both "how does go-rag-starter work" AND "how does Acme work."
2. **Replace completely** — delete everything in `docs/` and add only your own product docs. The bot is purely for your users.

Option 2 is the typical path once you're past the learning phase.

## Writing style

The single biggest lesson from building RAG bots: **the bot is only as good as your docs**. If your docs are vague, the bot's answers will be vague. Spend 80% of your effort writing crisp source docs; 20% is engineering.
