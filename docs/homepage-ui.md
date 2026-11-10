# Homepage design

The public homepage at `/` uses black (#000), orange (#ff4f00), and white (#fff), with a matching favicon and browser theme. Its layout takes visual direction from the [Zapier homepage](https://zapier.com/) while keeping TORQ branding, original illustrations, and copy specific to this CRM.

The page includes an editorial hero, a product preview, a three-step introduction, agent/webhook/workflow links, a FAQ, and a closing workspace invitation. Illustrations are local inline SVG, so there are no new image downloads or dependencies. The heading and body typography use the local Arial/Helvetica/system font stack.

The sample pipeline, contacts, and tasks are clearly labelled as illustrative. Task checkboxes change only local preview state. Workspace links use existing authenticated routes. Login and workspace access follow the existing authentication flow.

Motion includes staggered entrance reveals, tab transitions, subtle floating artwork, and a moving connector line. The hero's pause/play control stops its ongoing animation. Reduced-motion preferences disable animations and update immediately when the system preference changes. The mobile menu closes on navigation and Escape; product tabs support arrow keys, Home, and End; FAQ controls expose their expanded state. There is a skip link and visible keyboard focus styling.

## Verification

- Production frontend build and targeted lint pass.
- Browser screenshots reviewed on desktop and mobile, including the hero, product, process, connections, FAQ, and footer.
- Headless browser interaction checks pass for product tabs, keyboard navigation, sample task completion, FAQ expansion, mobile navigation and Escape, animation pause, and reduced motion.
- Responsive checks at 320, 390, 768, 1024, and 1440 pixels found no page-level horizontal overflow. The pipeline and contact table scroll within their preview on narrow screens.
- No browser runtime exceptions were recorded during these checks.

Development frontend remains available at http://localhost:5173/.

## Sign-in page

The `/login` page now shares the homepage's black, orange, and white palette, sans-serif typography, thin borders, and original inline SVG illustration. The illustration uses a short entrance sequence; reduced-motion settings disable it. Narrow screens show the form directly.

Email/password authentication and the existing protected-route redirect are preserved. Fields retain password-manager autocomplete, explicit labels, required-field validation, password visibility control, pending-state disabling, and an error message that receives focus. The browser check simulated a rejected login to verify pending/error behavior without changing an account. Responsive and visual checks passed at desktop and mobile sizes; production build and targeted lint passed.

Homepage copy was revised to describe CRM functions directly, with repeated star decorations and promotional slogans removed.
