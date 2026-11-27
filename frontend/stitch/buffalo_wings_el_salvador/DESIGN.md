---
name: Buffalo Wings El Salvador
colors:
  surface: '#f9f9f9'
  surface-dim: '#dadada'
  surface-bright: '#f9f9f9'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f3f3f3'
  surface-container: '#eeeeee'
  surface-container-high: '#e8e8e8'
  surface-container-highest: '#e2e2e2'
  on-surface: '#1a1c1c'
  on-surface-variant: '#5a4138'
  inverse-surface: '#2f3131'
  inverse-on-surface: '#f1f1f1'
  outline: '#8e7166'
  outline-variant: '#e2bfb2'
  surface-tint: '#a63b00'
  primary: '#a63b00'
  on-primary: '#ffffff'
  primary-container: '#ff671d'
  on-primary-container: '#591c00'
  inverse-primary: '#ffb599'
  secondary: '#615d5e'
  on-secondary: '#ffffff'
  secondary-container: '#e4dedf'
  on-secondary-container: '#656263'
  tertiary: '#006591'
  on-tertiary: '#ffffff'
  tertiary-container: '#00a1e5'
  on-tertiary-container: '#00334d'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#ffdbce'
  primary-fixed-dim: '#ffb599'
  on-primary-fixed: '#370e00'
  on-primary-fixed-variant: '#7f2b00'
  secondary-fixed: '#e7e1e2'
  secondary-fixed-dim: '#cbc5c6'
  on-secondary-fixed: '#1d1b1c'
  on-secondary-fixed-variant: '#494647'
  tertiary-fixed: '#c9e6ff'
  tertiary-fixed-dim: '#89ceff'
  on-tertiary-fixed: '#001e2f'
  on-tertiary-fixed-variant: '#004c6e'
  background: '#f9f9f9'
  on-background: '#1a1c1c'
  surface-variant: '#e2e2e2'
  bw-orange: '#FF671D'
  bw-orange-deep: '#F56516'
  bw-yellow: '#FFB400'
  bw-lime: '#D2D023'
  bw-forest: '#063C00'
  bw-red: '#FD0102'
  bw-black: '#000000'
  bw-ink: '#1D120E'
  bw-charcoal: '#3D3A3B'
  bw-charcoal-dark: '#3B3A3B'
  bw-charcoal-light: '#4C4B4C'
  bw-white: '#FFFFFF'
  bg-paper: '#F1F1F1'
  bg-paper-dark: '#E0E0E0'
  bg-paper-mid: '#E4E4E2'
typography:
  display-hero:
    fontFamily: Anton, Impact, sans-serif
    fontSize: 230px
    fontWeight: '900'
    lineHeight: '0.85'
    letterSpacing: -0.02em
  display-hero-mobile:
    fontFamily: Anton, Impact, sans-serif
    fontSize: 96px
    fontWeight: '900'
    lineHeight: '0.85'
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Anton, Impact, sans-serif
    fontSize: 160px
    fontWeight: '900'
    lineHeight: '0.85'
    letterSpacing: -0.03em
  headline-lg-mobile:
    fontFamily: Anton, Impact, sans-serif
    fontSize: 72px
    fontWeight: '900'
    lineHeight: '0.9'
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Anton, Impact, sans-serif
    fontSize: 70px
    fontWeight: '800'
    lineHeight: '0.9'
    letterSpacing: '0'
  headline-sm:
    fontFamily: Bebas Neue, sans-serif
    fontSize: 36px
    fontWeight: '700'
    lineHeight: '1'
    letterSpacing: 0.01em
  section-title:
    fontFamily: Anton, sans-serif
    fontSize: 24px
    fontWeight: '800'
    lineHeight: '1'
    letterSpacing: 0.02em
  nav-link:
    fontFamily: Bebas Neue, sans-serif
    fontSize: 15px
    fontWeight: '400'
    lineHeight: '1'
    letterSpacing: -0.01em
  category-item:
    fontFamily: Bebas Neue, sans-serif
    fontSize: 13px
    fontWeight: '400'
    lineHeight: '1'
    letterSpacing: -0.01em
  body-md:
    fontFamily: Inter, system-ui, sans-serif
    fontSize: 13px
    fontWeight: '400'
    lineHeight: '1.2'
    letterSpacing: '0'
  label-sm:
    fontFamily: Bebas Neue, sans-serif
    fontSize: 10px
    fontWeight: '400'
    lineHeight: '1.05'
    letterSpacing: '0'
spacing:
  gutter: 16px
  margin: 24px
  space-xs: 4px
  space-sm: 8px
  space-md: 16px
  space-lg: 24px
  space-xl: 32px
---

## Brand & Style

This design system channels an energetic street-style fast-food aesthetic, combining bold editorial layouts with the tactile feel of printed newspaper posters and hand-glued sticker collages. The brand personality is loud, youth-oriented, vibrant, and unapologetically bold. 

The visual style merges High-Contrast / Bold principles with tactile paper textures, heavy drop shadows, and aggressive typographic rotations. The interface relies on a strict high-impact hierarchy, where chaotic promotional sticker placements are anchored by clean, rigid navigation bars. The emotional response is immediate hunger, excitement, and a sense of vibrant urban culture.

## Colors

The color palette is built on a high-contrast foundation of urban charcoal and gritty paper textures, ignited by fiery brand oranges and punchy promotional accent colors. 

- **Primary (`--bw-orange`)**: The core interactive and call-to-action color, driving category bars, chat buttons, and active states.
- **Secondary (`--bw-charcoal`)**: Deep, industrial gray utilized for structural bars, dark section backgrounds, and high-contrast framing.
- **Neutrals (`--bg-paper`)**: Light, grainy paper tones that serve as the primary canvas, providing a physical, printed-poster feel.
- **Named Accents**: A vibrant suite of promotional colors (`bw-yellow`, `bw-lime`, `bw-red`, `bw-forest`) reserved strictly for stickers, tags, and dynamic food flavor callouts.

## Typography

Typography is loud, compressed, and architectural. The system relies on heavy ultra-black display fonts (`Anton`, `Bebas Neue`) paired with clean, highly legible sans-serif utility text (`Inter`). 

Large display roles embrace massive scales, tight line-heights, and intentional multi-angle rotations (e.g., `-8deg`, `12deg`) to mimic pasted street posters. All navigation and promotional labels are set in uppercase condensed weights to maximize visual density and impact.

## Layout & Spacing

The layout model is built on a rigid, anchored structural grid paired with fluid, high-energy breaking elements. 

- **Structural Framing**: The top navbar (50px height) and category bar (42px height) provide fixed, reliable horizontal navigation across viewports.
- **Content Flow**: Page sections utilize fluid vertical stacks with consistent outer margins (24px) and tight internal component gaps (8px to 16px).
- **Responsive Adaptations**: On mobile devices, massive display typography scales down aggressively to prevent layout breaking, while sticky action buttons (like the chat FAB) remain pinned 24px from the bottom and 28px from the right.

## Elevation & Depth

Visual hierarchy is achieved through a "newspaper collage" depth philosophy: flat, high-contrast structural layers supplemented by hard, unapologetic drop shadows.

- **Stickers & Badges**: Feature hard offset shadows (`0 6px 0 rgba(0, 0, 0, 0.15)`) to simulate physical paper cutouts glued onto the surface.
- **Floating Elements**: Floating action buttons (FABs) utilize soft, diffused card shadows (`0 4px 12px rgba(0, 0, 0, 0.35)`).
- **Food Assets**: Cutout photography employs realistic drop shadows (`0 18px 18px rgba(0, 0, 0, 0.28)`) to lift key products off the paper background.
- **Layering**: Z-index architecture strictly separates the paper background (Layer 0), large display typography (Layer 1), and physical stickers/food cutouts (Layer 2), anchored by sticky navigation at Layer 50+.

## Shapes

The shape language is stark, geometric, and intentionally unrefined (Sharp level `0`). 

- **Stickers & Badges**: Rendered with sharp 0px rectangular corners to emulate physical paper clippings and postage labels.
- **Interactive Elements**: Floating action buttons and avatars use fully circular (`50%`) geometry to contrast against the rigid rectangular containers.
- **Dividers & Strokes**: Structural navbar dividers use clean 1px solid lines, while category icons employ hand-drawn, sketchy line-art strokes (1.5px width, unfilled) for an authentic street-food aesthetic.

## Components

- **Buttons**: Built with sharp corners or bold pill shapes. Primary CTAs use solid `--bw-orange` with uppercase condensed text, snapping instantly on hover with high-contrast state changes.
- **Chips & Stickers**: Rectangular promotional tags (`--bw-yellow`, `--bw-red`) set at dynamic rotations with hard paper drop shadows. Uppercase condensed bold text.
- **Category Bar**: A sticky horizontal strip (`--bw-charcoal` or `--bw-orange`) featuring hand-drawn 1.5px line-art icons paired with uppercase condensed labels, featuring smooth `translateY(-2px)` hover lifts.
- **Input Fields**: Utilitarian, high-contrast bounding boxes with clean borders and crisp, legible `Inter` typography for user data entry.
- **Cards**: Flat or dark-surface containers (`--bw-charcoal-dark`) holding menu items, leveraging realistic food cutouts with deep drop shadows that break the container boundaries.
- **Lists**: Clean vertical arrangements separated by 1px solid dividers (`--bw-charcoal-light`), optimizing scannability for fast-food menus.