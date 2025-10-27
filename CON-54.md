Implement 'conduit introspect resources' Command

🎯 Business Context & Purpose

Problem: Developers need a quick way to see all resources in their application. This is the entry point for introspection - the "what exists?" question.

Business Value: This is the most frequently used command (projected 80% usage). Fast, scannable output increases developer productivity.

User Impact: Developers can instantly see all resources without opening files or searching through code.

📋 Expected Behavior/Outcome

Command: conduit introspect resources

Default Output (scannable summary):

RESOURCES (12 total)

Core Resources:
User 8 fields 2 relationships 1 hook ✓ auth required
Post 15 fields 3 relationships 2 hooks ✓ cached
Comment 6 fields 2 relationships 1 hook ✓ nested

Administrative:
Category 4 fields 1 relationship - -
Tag 3 fields - - -

With --verbose: Show all fields, relationships, middleware per resource

With --format json: Machine-readable JSON for tooling

✅ Acceptance Criteria

Command conduit introspect resources implemented

Default output: Scannable table with key metrics

--verbose flag shows detailed information

--format json outputs structured JSON

--no-color flag disables color output

Categorizes resources (Core, Administrative, System)

Shows field count, relationship count, hook count

Indicates special flags (auth required, cached, nested)

Response time: <100ms

Help text with examples

Unit tests for output formatting

Integration test with real registry

Pragmatic Effort Estimate: 2 days

🔗 Dependencies & Constraints

Dependencies:

Requires CLI structure (CON-53)

Requires runtime registry (CON-52)

Code Reference: See IMPLEMENTATION-RUNTIME.md:803-862
