# Lessons Learned

> **WHAT**: Operator field notes — patterns that bit us, what we wish we'd known earlier, post-incident insights. One file per topic; named `<topic>.md`.
> **AUTHORITY**: 📚 Operator notes. Not canon; canon lives in [../PRINCIPLES.md](../PRINCIPLES.md). When a lesson hardens into a rule, promote it to PRINCIPLES.

## Index

(Empty.) When you finish an incident or hit a "wish I knew" moment, drop a note here.

## When to write a lesson

- A bug class that took > 2 hours to find — write the diagnostic path so next time someone catches it faster.
- A non-obvious config setting that, once flipped, fixed a class of issues.
- A confusing repo / deploy boundary that tripped you up (e.g., manifests-in-openova-private-not-here).
- A non-obvious EAS / Expo / Xcode behaviour.

## Template

```markdown
# <Topic>

| Date | YYYY-MM-DD |
| Author | <handle> |
| Surface | mobile / backend / data / ci-cd / infra |
| Related issues | #N, #M |

## Symptom

<What was visible to the operator?>

## What I assumed (and why it was wrong)

<The diagnostic dead-ends.>

## What it actually was

<Root cause.>

## How to spot it next time

<Concrete tells — log line, error shape, etc.>

## Fix shape

<What the right fix looked like; reference the PR.>
```
