## Embed the widget on your site

The bot serves a widget script at `/widget.js`. Any HTML page can include it with a single `<script>` tag. No build step, no framework required.

## Basic embed

```html
<script src="http://localhost:8080/widget.js"></script>
```

That's it. A floating bubble appears in the bottom-right of every page where this script loads.

## Embed on a different domain

If your bot is at `https://bot.example.com` and your marketing site is at `https://www.example.com`, pass the API URL via `data-api`:

```html
<script src="https://bot.example.com/widget.js"
        data-api="https://bot.example.com"></script>
```

The widget sends queries to `{data-api}/api/query`. The bot sets permissive CORS headers (`Access-Control-Allow-Origin: *`) so cross-origin works out of the box.

## Customise title and subtitle

```html
<script src="https://bot.example.com/widget.js"
        data-api="https://bot.example.com"
        data-title="Ask Acme"
        data-subtitle="Answers from the Acme docs"></script>
```

## What the widget renders

- A 52×52 pixel floating bubble in the bottom-right corner, violet-to-pink gradient.
- Click to open a right-side slide-in panel (max 420px wide, full height).
- **Non-modal** — you can click around the host page while the chat is open. Only the X button or the bubble closes it.
- Mobile: the panel takes the full screen width below 420px viewport.

## Starter prompts

When the panel opens for the first time, it shows four clickable starter prompts. These are hardcoded in `web/widget.js`:

```javascript
const starterPrompts = [
  'How do I replace the corpus?',
  'How do I swap the LLM?',
  'How do I embed this on my site?',
  'What is pgvector?',
];
```

Edit them to match your product (e.g. "How do I set up payments?", "How do I reset my password?") and rebuild.

## Styling

All styles are scoped to classes prefixed with `ragbot-` and inlined in the widget file. They won't collide with your host site's CSS.

To change colours, edit the two occurrences of `linear-gradient(135deg, #8b5cf6, #ec4899)` in `web/widget.js`. The rest of the palette (dark background, muted text) is derived from these two and a few neutrals.

## React / Vue / Svelte hosts

The widget works the same. Put the `<script>` tag in your `public/index.html` or call it from your layout component:

```jsx
// Next.js
<Script src="https://bot.example.com/widget.js" strategy="lazyOnload" />
```

```vue
<!-- Nuxt: in nuxt.config.ts -->
app.head.script: [{ src: 'https://bot.example.com/widget.js', defer: true }]
```

## Production considerations

- **Rate limiting** — the bot has no built-in rate limiter. Put Cloudflare or your CDN in front if the widget is on a public page.
- **Content Security Policy** — if your site has a CSP, allow the bot's origin in `connect-src` and `script-src`.
- **Cookies** — the widget doesn't set any cookies. Conversations are identified by a UUID that lives in JavaScript memory only.
