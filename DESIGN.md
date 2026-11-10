# Design System Inspired by Home

> Auto-extracted from `https://zapier.com/app/home?conversationId=57e814d3-68a0-4f1d-a2d0-cda9039bf55a` on 2026-11-08

## 1. Visual Theme & Atmosphere

Friendly, approachable design with rounded shapes and generous whitespace.

The hero section leads with "What would you like to automate?" followed by "CopilotLoadingLoadingCopilot is AI and can make mistakes. Please double-check responses.".

**Key Characteristics:**
- Degular Display as the heading font
- Inter as the body font for all running text
- Heading weight 600, letter-spacing 1px
- Light/white background (#fffefb) as the primary canvas
- Primary accent `#ff4f00` used for CTAs and brand highlights
- 4 shadow level(s) detected — tinted shadows
- Rounded corners (4px+) creating a friendly, approachable feel
- Tags: light, rounded, colorful, monospace, sans-serif

## 2. Color Palette & Roles

### Primary
- **Primary Accent** (`#ff4f00`) · `--color-primary`: Brand color, CTA backgrounds, link text, interactive highlights.
- **Secondary Accent** (`#695be8`) · `--color-secondary`: Secondary brand, hover states, complementary highlights.
- **Background** (`#fffefb`) · `--color-bg`: Page background, primary canvas.
- **Background Secondary** (`#fdf5f2`) · `--color-bg-secondary`: Cards, surfaces, alternating sections.

### Text
- **Text Primary** (`#000000`) · `--color-text`: Headings and body text.
- **Text Secondary** (`#666666`) · `--color-text-secondary`: Muted text, captions, placeholders.

### Borders & Surfaces
- **Border** (`#fcfdfc`) · `--color-border`: Dividers, outlines, input borders.

### Full Extracted Palette

| # | Hex | CSS Variable | Role | Area | Contrast |
|---|---|---|---|---|---|
| 1 | `#fffefb` | `--palette-1` | block | large | text-dark |
| 2 | `#fdf5f2` | `--palette-2` | button | medium | text-dark |
| 3 | `#503ebd` | `--palette-3` | button | small | text-light |
| 4 | `#ff4f00` | `--palette-4` | button | small | text-light |
| 5 | `#695be8` | `--palette-5` | text-accent | small | text-light |
| 6 | `#156123` | `--palette-6` | badge | small | text-light |
| 7 | `#d5d7fc` | `--palette-7` | badge | small | text-dark |
| 8 | `#aee1fd` | `--palette-8` | badge | small | text-dark |
| 9 | `#035a7b` | `--palette-9` | badge | small | text-light |
| 10 | `#201515` | `--palette-10` | badge | small | text-light |
| 11 | `#ece9df` | `--palette-11` | badge | small | text-dark |

## 3. Typography Rules

- **Heading Font:** `Degular Display`, sans-serif
- **Body Font:** `Inter`, sans-serif

### Type Hierarchy

| Role | Font | Size | Weight | Line Height | Letter Spacing |
|---|---|---|---|---|---|
| H1 | Degular Display | 36px | 600 | 48px | 1px |
| H2 | Inter | 18px | 600 | 24px | normal |
| Body | Inter | 12px | 400 | 20px | normal |

### Type Scale

| Token | Size | Suggested Usage |
|---|---|---|
| Display | `36px` | headings |
| H1 | `22px` | headings |
| H2 | `18px` | headings |
| H3 | `16px` | headings |
| H4 | `14px` | headings |
| Body L | `13px` | body / supporting text |
| Body | `12px` | body / supporting text |
| Small | `10px` | body / supporting text |

## 4. Component Stylings

### Primary Button

```css
.btn-primary {
  background: transparent;
  color: #88827e;
  border-radius: 50px;
  padding: 0px 0px;
  font-size: 16px;
  font-weight: 400;
  border: none;
  cursor: pointer;
}
```

### Ghost Button

```css
.btn-ghost {
  background: transparent;
  color: #574e4c;
  border-radius: 0px;
  padding: 0px 0px;
  font-size: 10px;
  font-weight: 400;
  border: none;
  cursor: pointer;
}
```

### Ghost Button 2

```css
.btn-ghost-2 {
  background: transparent;
  color: #413735;
  border-radius: 4px;
  padding: 0px 0px;
  font-size: 14px;
  font-weight: 400;
  border: none;
  cursor: pointer;
}
```

### Filled Button

```css
.btn-filled {
  background: #fffefb;
  color: #413735;
  border-radius: 4px;
  padding: 0px 12px;
  font-size: 14px;
  font-weight: 500;
  border: 0.8px solid rgb(215, 211, 201);
  cursor: pointer;
}
```

### Ghost Button 3

```css
.btn-ghost-3 {
  background: transparent;
  color: #000000;
  border-radius: 50px;
  padding: 0px 0px;
  font-size: 16px;
  font-weight: 400;
  border: none;
  cursor: pointer;
}
```

### Filled Button 2

```css
.btn-filled-2 {
  background: #ff4f00;
  color: #fffefb;
  border-radius: 8px;
  padding: 0px 16px;
  font-size: 14px;
  font-weight: 600;
  border: 0.8px solid rgb(255, 79, 0);
  cursor: pointer;
}
```

### Card

```css
.card {
  background: #f7f6fd;
  border-radius: 12px;
  padding: 12px;
}
```

## 5. Layout Principles

- **Base spacing unit:** `12px` — use multiples (24px, 36px, 48px, etc.)

### Spacing Scale (extracted from real elements)

| Token | Value | Role |
|---|---|---|
| spacing-1 | `12px` | element |
| spacing-2 | `2px` | element |
| spacing-3 | `8px` | element |
| spacing-4 | `4px` | element |
| spacing-5 | `5px` | element |
| spacing-6 | `16px` | element |
| spacing-7 | `1px` | element |
| spacing-8 | `6px` | element |

### Border Radius Scale

| Token | Value | Element |
|---|---|---|
| radius-subtle | `4px` | subtle |
| radius-button | `8px` | button |
| radius-card | `50px` | card |
| radius-button | `6px` | button |
| radius-subtle | `5px` | subtle |
| radius-subtle | `2px` | subtle |

## 6. Depth & Elevation

| Level | Shadow | Usage |
|---|---|---|
| Low | `rgba(32, 21, 21, 0.08) 0px 1px 0px 0px` | Cards, subtle elevation |
| Low | `rgba(32, 21, 21, 0.08) 0px 1px 3px 0px` | Cards, subtle elevation |
| Low | `rgba(32, 21, 21, 0.12) 0px 2px 4px 0px` | Cards, subtle elevation |
| Deep | `rgba(0, 0, 0, 0.15) 0px 8px 32px 0px` | Hero sections, deep layers |

> **Note:** This site uses chromatic (color-tinted) shadows rather than pure black — this is a deliberate brand choice that adds warmth to elevation.

## 7. Do's and Don'ts

### Do
- Use `#fffefb` as the primary background color
- Use `Degular Display` for all headings and `Inter` for body text
- Use `#ff4f00` as the single dominant accent/CTA color
- Maintain `12px` as the base spacing unit — all gaps should be multiples
- Use rounded corners (`4px`+) consistently for all interactive elements
- Embrace bold color combinations — playful energy is the point
- Apply the shadow system for elevation — use the extracted shadow values
- Use weight 600 for headings to match the brand's typographic voice

### Don't
- Don't use colors outside the extracted palette without justification
- Don't substitute Degular Display/Inter with generic alternatives
- Don't use irregular spacing — stick to 12px grid
- Don't use dark/black backgrounds — this is a light-themed design
- Don't use sharp corners — they feel hostile in this rounded design language
- Don't use pure black (#000000) for text — use `#000000` instead
- Don't add decorative elements not present in the original design — no badges, ribbons, banners, or ornaments unless the source site uses them
- Don't invent UI patterns the source site doesn't have — if the original has no NEW badge, don't add one just because a red is in the palette

## 8. Responsive Behavior

| Breakpoint | Width | Notes |
|---|---|---|
| Mobile | < 640px | Single column, stack sections, reduce font sizes ~80% |
| Tablet | 640–1024px | 2-column where appropriate, maintain spacing ratios |
| Desktop | 1024–1440px | Full layout as designed |
| Wide | > 1440px | Max-width container, center content |

- Touch targets: minimum 44×44px on mobile
- Maintain 12px base unit across breakpoints — only scale multipliers

## 9. Agent Prompt Guide

### Quick Color Reference

```
Background:  #fffefb
Text:        #000000
Accent:      #ff4f00
Secondary:   #695be8
Border:      #fcfdfc
```

### Example Prompts

1. "Build a hero section with a `#fffefb` background, `Degular Display` heading in `#000000`, and a `#ff4f00` CTA button with 4px radius."
2. "Create a pricing card using background `#fdf5f2`, border `#fcfdfc`, `Inter` for text, and 36px padding."
3. "Design a navigation bar — `#fffefb` background, `#000000` links, `#ff4f00` for active state."
4. "Build a feature grid with 3 columns, 36px gap, each card using the card component style."
5. "Create a footer with `#000000` background, `#fffefb` text, and 24px padding."

### Iteration Guide

1. Start with layout structure (sections, grid, spacing)
2. Apply colors from the palette — background first, then text, then accents
3. Set typography — font families, sizes from the type scale, weights
4. Add components — buttons, cards, inputs using the specs above
5. Apply border-radius consistently across all elements
6. Add shadows for depth — use the extracted shadow values, not defaults
7. Check responsive behavior — test mobile and tablet layouts
8. Final pass — verify all colors match, spacing is consistent, fonts are correct
