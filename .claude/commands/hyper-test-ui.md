
Your job: thoroughly test a given web page, capture screenshots, evaluate look & feel, exercise every available functionality, and produce a Hyper human task with agent tasks (one per issue).


Use playwright MCP to perform the job. 

⸻

Inputs (validate strictly)
	•	page
	

If page_url is missing or unreachable, return an Error (see Error schema) with a clear suggestion_action.

⸻

Test Plan (execute deterministically)
	1.	Load & Readiness
	•	Verify page loads without console errors; wait for network idle.
	•	Capture initial full-page screenshot per device.
	2.	Navigation & IA
	•	Exercise all visible nav elements (header, menus, tabs, breadcrumbs, footer).
	•	Validate URL changes, active states, breadcrumbs consistency.
	3.	Interactive Elements
	•	Click all buttons/links; open/close modals, drawers, accordions, tooltips.
	•	Forms: fill valid and invalid data; verify validation, error copy, and focus states.
	•	Rich widgets: tables (sort, filter, paginate), dropdowns, date pickers, carousels, uploaders.
	4.	Look & Feel / Visual QA
	•	Layout: spacing, alignment, grid, responsive breakpoints.
	•	Typography: hierarchy, truncation/overflow, line-height.
	•	Color & contrast (WCAG AA targets), theme consistency (light/dark if applicable).
	•	Iconography and imagery clarity; empty states and loading skeletons.
	5.	A11y (Accessibility)
	•	Keyboard-only traversal, focus order, focus ring visibility.
	•	ARIA roles/labels for interactive controls.
	•	Alt text for images; semantic headings (H1…Hn).
	6.	Content & i18n
	•	Copy typos, grammar, tone consistency.
	•	Locale switches (if present), date/number formats; truncation in smaller viewports.
	7.	Performance (lightweight checks)
	•	Perceived load (first paint), spinner durations, image weight (rough).
	•	Lazy loading for images/lists; unnecessary reflows on interactions.
	8.	Resilience & Error States
	•	Network hiccups (if togglable): error banners, retry actions.
	•	Empty data and permission-denied states (if navigable).
	9.	Analytics & Telemetry (if visible)
	•	Existence of data-attributes or events on key actions (non-invasive confirmation).
	10.	Security UX

	•	Password fields masking, copy/paste policies, sensitive info in DOM or URL.
	•	Mixed-content warnings, insecure endpoints usage in network log.

⸻

Evidence Requirements
	•	Screenshots for: initial load, each modal/popup, each validation error, each viewport breakpoint, and any defect found.
	•	For each issue, attach:
	•	screenshots (1+),
	•	dom_snippet (small relevant HTML),
	•	console_log_excerpt (if error),
	•	steps_to_reproduce (deterministic),
	•	expected_result vs actual_result.

⸻
Prioritization & Reporting Rules
	•	Severity mapping guidance:
	•	Blocker: prevents primary user flow or security-meaningful issue.
	•	High: core feature degraded; major accessibility failure.
	•	Medium: noticeable defect; workaround exists.
	•	Low: cosmetic or minor copy/spacing.
	•	Order agent_tasks by: severity then impact then estimated_effort.
	•	Include a short rationale for prioritization in summary.

⸻

Operational Notes
	•	Be deterministic: same inputs → same outputs.
	•	Do not invent data; if an element is not present, mark as N/A and proceed.
	•	Keep screenshots readable; prefer full-width for full-page and tight crops for defects.
	•	If a flow branches, test happy path and at least one invalid path per form.

