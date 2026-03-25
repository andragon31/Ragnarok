# Fenrir Protocol

You have access to Fenrir, an AI Governance & Memory Layer for project memory.

## Fenrir Persistent Memory — Protocol (MANDATORY)

### WHEN TO SAVE (MANDATORY after any of these):
1. Call **mem_save** IMMEDIATELY after:
- Bug fix completed
- Architecture or design decision made
- Non-obvious discovery or pattern established
- Configuration change or environment setup
- User preference or constraint learned

2. Use **topic_key** (e.g. "auth-logic", "db-schema", "api-design") to update evolving topics instead of creating new scattered observations.

### WHEN TO SEARCH:
- When asked about past work ("remember", "what did we do", "recordar", "qué hicimos")
- Before asking the user — check memory first with **mem_find** or **mem_context**.

### SESSION WORKFLOW:
1. **Start**: Call **mem_session_start** with your goal.
2. **Work**: Use **pkg_check** before adding dependencies.
3. **Capture**: Use **mem_save** to record progress.
4. **End**: Call **mem_session_end** with:
   - Goals achieved
   - Technical findings/discoveries
   - Decisons made
   - Next steps (what's pending)

## Fenrir Rules:
1. NEVER skip **mem_session_start**.
2. ALWAYS use **topic_key** for evolving topics.
3. If unsure about a package, use **pkg_license** and **pkg_audit**.
4. If you see a "Compaction" or "Context Reset" message, recover context with **mem_context**.
