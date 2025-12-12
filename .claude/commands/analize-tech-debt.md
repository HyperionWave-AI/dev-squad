Goals:
	1.	Find Dead or Unused Code:
	•	Identify functions, classes, or modules that are no longer used or referenced.
	•	Check for deprecated APIs or stale configurations.
	•	Flag unmaintained dependencies or legacy patterns (e.g., old frameworks or utility libraries).
	2.	Assess Code Structure and Maintainability:
	•	Evaluate adherence to SOLID principles, layering, and separation of concerns.
	•	Detect large or complex modules that violate single-responsibility.
	•	Check for circular dependencies or tight coupling between components.
	3.	Check for Code Duplication and Redundancy:
	•	Locate repeated code blocks, especially in service logic, utilities, and database access layers.
	•	Suggest opportunities to refactor repeated patterns into shared modules or helpers.
	4.	Analyze Tests and Reliability:
	•	Determine areas lacking test coverage.
	•	Identify brittle or redundant test cases.
	5.	Recommend Improvements:
	•	Propose clear and actionable steps to reduce technical debt.
	•	Suggest structural or architectural refactors.
	•	Prioritize issues based on impact, complexity, and risk.

Inputs:
Provide:
	•	Ask what service to cover
	•	technical-knoweledge collection in hyper
    •	adr collection in hyper
    
Output:
	•	Hyper human task with multiple agent tasks (1 per tech debt issue) 