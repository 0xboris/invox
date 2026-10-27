# GitHub through REST

Claude Code cloud sessions reach GitHub through an egress proxy that refuses every GraphQL call with a 403. `gh pr ...`, `gh issue ...` and `gh api graphql` are GraphQL-backed, so they fail there. Use `gh api` REST calls, or the GitHub MCP tools where a playbook names them. The proxy adds `/ccr/` routes for the operations REST lacks. github.com answers them with 404, so they work only inside a cloud session.

`gh api` fills `{owner}` and `{repo}` from the current checkout. `<n>` is the PR number and `<sha>` the head commit.

| Operation | Command |
| --- | --- |
| View a PR | `gh api repos/{owner}/{repo}/pulls/<n>` (state, `merged_at`, `mergeable`, `mergeable_state`, `draft`, `head.sha`, `auto_merge`) |
| Create a PR | `gh api -X POST repos/{owner}/{repo}/pulls -f head=<branch> -f base=<base> -f title=<title> -F body=@<body.md>` |
| Retarget a PR | `gh api -X PATCH repos/{owner}/{repo}/pulls/<n> -f base=<base>` |
| Mark ready | `gh api -X POST repos/{owner}/{repo}/pulls/<n>/ccr/ready_for_review` |
| Convert to draft | `gh api -X POST repos/{owner}/{repo}/pulls/<n>/ccr/convert_to_draft` |
| List PRs | `gh api 'repos/{owner}/{repo}/pulls?state=open&per_page=100&page=<p>'` |
| Checks | `gh api 'repos/{owner}/{repo}/commits/<sha>/check-runs?per_page=100&page=<p>'` and `gh api repos/{owner}/{repo}/commits/<sha>/status` |
| Reviews | `gh api 'repos/{owner}/{repo}/pulls/<n>/reviews?per_page=100&page=<p>'` |
| Review threads | `gh api repos/{owner}/{repo}/pulls/<n>/ccr/review_threads` (rows of `comment_ids`, `resolved`, `outdated`; the first id opens the thread) |
| Thread comments | `gh api 'repos/{owner}/{repo}/pulls/<n>/comments?per_page=100&page=<p>'` |
| Reply on a thread | `gh api -X POST repos/{owner}/{repo}/pulls/<n>/comments/<comment-id>/replies --input <payload.json>` |
| Resolve a thread | `gh api -X POST repos/{owner}/{repo}/pulls/<n>/ccr/comments/<opening-comment-id>/resolve` (or `/unresolve`) |
| Squash-merge | `gh api -X PUT repos/{owner}/{repo}/pulls/<n>/merge -f merge_method=squash -f sha=<sha>` |
| Arm auto-merge | `gh api -X PUT repos/{owner}/{repo}/pulls/<n>/ccr/auto_merge -f merge_method=squash` |
| Disarm auto-merge | `gh api -X DELETE repos/{owner}/{repo}/pulls/<n>/ccr/auto_merge` |
| Issue | `gh api repos/{owner}/{repo}/issues/<n>` and `gh api 'repos/{owner}/{repo}/issues/<n>/comments?per_page=100&page=<p>'` |

Page by hand. `gh api --paginate` follows `Link: rel="next"` URLs of the form `/repositories/<id>/...`, which the proxy refuses, so it fails once a list outgrows one page. Request `per_page=100&page=1`, `page=2`, and so on until a page returns fewer than 100 items.

Pass `sha=<head>` on every merge so GitHub refuses it if the head moved after you verified it. The merge and auto-merge rows still need the maintainer's go-ahead for that specific PR, as CLAUDE.md says.
