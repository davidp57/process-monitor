# Triage labels

One triage label per open issue, Matt Pocock's five roles used as GitHub labels as they are:

| Label | Meaning |
|---|---|
| `needs-triage` | Arrived from outside, not yet looked at |
| `needs-info` | Waiting for the reporter to answer a question |
| `ready-for-agent` | Specified: an agent can take it without asking anything |
| `ready-for-human` | Blocked on David: a decision to take, or a check that needs the real game or app running |
| `wontfix` | Decided against; the issue is closed as "not planned" with the reason in a comment |

`to-spec` and `to-tickets` create issues at `ready-for-agent`: they are specified by construction.

Work in progress needs no label: the open pull request that says `Closes #N` shows it.

A `ready-for-human` issue says in its body **what** is expected and **what each answer leads to**: a check that cannot come out negative proves nothing.
