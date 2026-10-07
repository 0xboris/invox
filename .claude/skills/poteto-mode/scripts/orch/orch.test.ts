import { afterEach, describe, expect, it } from "bun:test";
import {
  chmod,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  rm,
  writeFile,
} from "node:fs/promises";
import { realpathSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  NotFoundError,
  UserError,
  openStore,
  parseVerdict,
  type OpenStoreOptions,
  type Store,
} from "./store.ts";

const SCRIPT = join(import.meta.dir, "orch.ts");
const directories: string[] = [];
const handles: Store[] = [];

interface RunResult {
  readonly code: number;
  readonly stdout: string;
  readonly stderr: string;
}

async function makeDirectory(): Promise<string> {
  const directory = await mkdtemp(join(tmpdir(), "orch-test-"));
  directories.push(directory);
  return directory;
}

function useStore(
  directory: string,
  options?: OpenStoreOptions
): Store {
  const store = openStore(directory, options);
  handles.push(store);
  return store;
}

async function initializedStore(): Promise<{
  readonly directory: string;
  readonly store: Store;
}> {
  const directory = await makeDirectory();
  const store = useStore(directory);
  await store.init();
  return { directory, store };
}

function git({
  args,
  repo,
}: {
  args: readonly string[];
  repo: string;
}): string {
  const result = Bun.spawnSync(["git", "-C", repo, ...args]);
  if (result.exitCode !== 0) {
    throw new Error(
      `git ${args.join(" ")} failed: ${result.stderr.toString()}`
    );
  }
  return result.stdout.toString().trim();
}

async function makeGitStack(directory: string): Promise<{
  readonly repo: string;
  readonly mergedSha: string;
  readonly closedSha: string;
  readonly openSha: string;
}> {
  const repo = join(directory, "repo");
  await mkdir(repo);
  git({ repo, args: ["init", "--initial-branch=main"] });
  git({ repo, args: ["config", "user.name", "Orch Test"] });
  git({ repo, args: ["config", "user.email", "orch@example.com"] });
  await writeFile(join(repo, "main.txt"), "main\n");
  git({ repo, args: ["add", "."] });
  git({ repo, args: ["commit", "-m", "main"] });

  const branches = ["stack/merged", "stack/closed", "stack/open"];
  for (const [index, branch] of branches.entries()) {
    git({ repo, args: ["checkout", "-b", branch] });
    await writeFile(join(repo, `stack-${index}.txt`), `${branch}\n`);
    git({ repo, args: ["add", "."] });
    git({ repo, args: ["commit", "-m", branch] });
  }

  return {
    repo,
    mergedSha: git({ repo, args: ["rev-parse", "stack/merged"] }),
    closedSha: git({ repo, args: ["rev-parse", "stack/closed"] }),
    openSha: git({ repo, args: ["rev-parse", "stack/open"] }),
  };
}

async function withFakeGt<T>({
  directory,
  operation,
  output,
}: {
  directory: string;
  operation: (outputPath: string) => Promise<T>;
  output: string;
}): Promise<T> {
  const bin = join(directory, "bin");
  const outputPath = join(directory, "gt-output.txt");
  await mkdir(bin);
  await writeFile(outputPath, output);
  const gt = join(bin, "gt");
  await writeFile(
    gt,
    `#!/usr/bin/env bash
set -euo pipefail
if [ "$(pwd -P)" != "${realpathSync(join(directory, "repo"))}" ]; then
  printf 'gt ran outside the fixture repo: %s\\n' "$(pwd -P)" >&2
  exit 2
fi
case "$*" in
  "--no-interactive log short --stack --reverse")
    cat "${outputPath}"
    ;;
  "--no-interactive info stack/merged")
    printf 'stack/merged\\nPR #10 (Merged) merged change\\n'
    ;;
  "--no-interactive info stack/closed")
    printf 'stack/closed\\nPR #13 (Closed) closed change\\n'
    ;;
  "--no-interactive info stack/open")
    printf 'stack/open\\nPR #11 (Needs approvals) open change\\n'
    ;;
  *)
    printf 'unexpected gt arguments: %s\\n' "$*" >&2
    exit 2
    ;;
esac
`
  );
  await chmod(gt, 0o755);

  const originalPath = process.env.PATH;
  process.env.PATH = `${bin}:${originalPath ?? ""}`;
  try {
    return await operation(outputPath);
  } finally {
    if (originalPath === undefined) {
      delete process.env.PATH;
    } else {
      process.env.PATH = originalPath;
    }
  }
}

function runCli(
  args: readonly string[],
  env: Readonly<Record<string, string | undefined>> = process.env
): RunResult {
  const result = Bun.spawnSync([process.execPath, SCRIPT, ...args], { env });
  return {
    code: result.exitCode,
    stdout: result.stdout.toString(),
    stderr: result.stderr.toString(),
  };
}

afterEach(async () => {
  for (const store of handles.splice(0).reverse()) {
    await store.close();
  }
  for (const directory of directories.splice(0)) {
    await rm(directory, { recursive: true, force: true });
  }
});

describe("Store", () => {
  it("initializes an idempotent plain-file store and releases its lock", async () => {
    const directory = await makeDirectory();
    const store = useStore(directory);

    expect(await store.init()).toEqual({ store: directory });
    const firstUnits = await readFile(join(directory, "units.tsv"), "utf8");
    const firstLedger = await readFile(
      join(directory, "ledger.tsv"),
      "utf8"
    );

    expect(await store.init()).toEqual({ store: directory });
    expect(await readFile(join(directory, "units.tsv"), "utf8")).toBe(
      firstUnits
    );
    expect(await readFile(join(directory, "ledger.tsv"), "utf8")).toBe(
      firstLedger
    );
    expect((await readdir(directory)).sort()).toEqual([
      ".orch.lock",
      "frontier.json",
      "gates.md",
      "inbox",
      "ledger.tsv",
      "preferences.md",
      "units.tsv",
    ]);

    await store.close();
    expect(await readdir(directory)).not.toContain(".orch.lock");
  });

  it("composes unit add, set, get, list, and counts", async () => {
    const { store } = await initializedStore();

    expect(
      await store.units.add({
        id: "u1",
        track: "build",
        brief: "briefs/u1.md",
      })
    ).toMatchObject({ id: "u1", state: "pending" });
    expect(
      await store.units.add({ id: "=SUM(A1)", track: "+build" })
    ).toMatchObject({ id: "'=SUM(A1)", track: "'+build" });

    const updated = await store.units.set({
      id: "u1",
      state: "done",
      branch: "poteto/u1",
      pr: 184530,
      sha: "abc123",
    });
    expect(updated).toEqual({
      id: "u1",
      track: "build",
      state: "done",
      branch: "poteto/u1",
      pr: "184530",
      sha: "abc123",
      brief: "briefs/u1.md",
    });
    expect(await store.units.get("u1")).toEqual(updated);
    expect(
      await store.units.list({ state: "done", track: "build" })
    ).toEqual([updated]);
    expect(await store.units.counts()).toEqual({ done: 1, pending: 1 });
    await expect(
      store.units.add({ id: "u1", track: "build" })
    ).rejects.toThrow("unit u1 already exists");
    await expect(
      store.units.set({ id: "missing", state: "done" })
    ).rejects.toBeInstanceOf(NotFoundError);
  });

  it("records, replaces, checks, and summarizes typed ledger verdicts", async () => {
    const { store } = await initializedStore();

    try {
      await store.ledger.check({ pr: 184530, sha: "abc123" });
      throw new Error("expected ledger check to fail");
    } catch (error) {
      expect(error).toBeInstanceOf(NotFoundError);
      if (error instanceof NotFoundError) {
        expect(error.output).toEqual({
          compact: "NOT-VERIFIED",
          json: {
            pr: "184530",
            sha: "abc123",
            verdict: "NOT-VERIFIED",
          },
        });
      }
    }
    expect(() => parseVerdict("looks-good")).toThrow("verdict must be");

    const recorded = await store.ledger.record({
      pr: 184530,
      sha: "abc123",
      verdict: "unit-test-verified",
      evidence: "reports/verify.md",
      verifier: "sol",
    });
    expect(await store.ledger.check({ pr: 184530, sha: "abc123" })).toEqual(
      recorded
    );
    expect(await store.ledger.summary()).toEqual({
      "unit-test-verified": 1,
    });

    await store.ledger.record({
      pr: 184530,
      sha: "abc123",
      verdict: "live-ui-verified",
      evidence: "reports/live.md",
    });
    expect(await store.ledger.summary()).toEqual({
      "live-ui-verified": 1,
    });
  });

  it("pushes, peeks, and atomically drains inbox pointers", async () => {
    const { directory, store } = await initializedStore();

    const first = await store.inbox.push({
      agent: "worker-1",
      unit: "u1",
      status: "done",
      report: "reports/u1.md",
    });
    expect(first.pointer).toMatchObject({ unit: "u1", status: "done" });
    expect(first.filename).toEndWith(".tsv");
    await store.inbox.push({
      agent: "worker-2",
      unit: "u2",
      status: "failed",
    });

    expect(await store.inbox.count()).toBe(2);
    expect(await store.inbox.peek()).toHaveLength(2);
    expect(await store.inbox.count()).toBe(2);
    expect(await store.inbox.drain()).toHaveLength(2);
    expect(await store.inbox.count()).toBe(0);
    expect(await readdir(join(directory, "inbox"))).toEqual([]);
    expect(
      (await readdir(directory)).filter((name) =>
        name.startsWith(".inbox-drain-")
      )
    ).toEqual([]);
  });

  it("replaces a stale lock whose holder pid is dead", async () => {
    const { directory } = await initializedStore();
    const exited = Bun.spawn(["true"]);
    await exited.exited;
    await writeFile(join(directory, ".orch.lock"), `${exited.pid}\n`);

    const stale: string[] = [];
    const recovered = useStore(directory, {
      onStaleLock: (holder) => stale.push(holder),
    });
    expect(
      await recovered.units.add({ id: "u1", track: "build" })
    ).toMatchObject({ id: "u1" });
    expect(stale).toEqual([String(exited.pid)]);
    await recovered.close();
    expect(await readdir(directory)).not.toContain(".orch.lock");
  });

  it("blocks a writer and steals the pid lock only with force", async () => {
    const { directory, store } = await initializedStore();
    await store.close();
    await writeFile(join(directory, ".orch.lock"), `${process.pid}\n`);

    const blocked = useStore(directory);
    await expect(
      blocked.units.add({ id: "u1", track: "build" })
    ).rejects.toThrow(`store lock held by pid ${process.pid}`);

    const stolen: string[] = [];
    const forced = useStore(directory, {
      force: true,
      onLockStolen: (holder) => stolen.push(holder),
    });
    expect(
      await forced.units.add({ id: "u1", track: "build" })
    ).toMatchObject({ id: "u1" });
    expect(stolen).toEqual([String(process.pid)]);
    await forced.close();
    expect(await readdir(directory)).not.toContain(".orch.lock");
  });

  it("parks gates, stores standing orders, and renders status", async () => {
    const { directory, store } = await initializedStore();
    await store.units.add({ id: "u1", track: "build" });
    expect(
      await store.gates.park({
        id: "release",
        question: "Ship now?",
        options: "ship,wait",
        defaultAnswer: "wait",
      })
    ).toMatchObject({ kind: "open", id: "release" });
    expect(
      await store.standing.add({ line: "Never force push." })
    ).toEqual({ number: 1, line: "Never force push." });

    const first = await store.status.render();
    expect(first.changed).toBe("first render");
    expect(first.summary.openGateIds).toEqual(["release"]);
    expect(await readFile(join(directory, "status.md"), "utf8")).toContain(
      "| release | open | Ship now? |"
    );
    expect((await store.status.render()).changed).toBe("no derived changes");

    expect(
      await store.gates.resolve({ id: "release", answer: "ship" })
    ).toMatchObject({ kind: "resolved", answer: "ship" });
    expect((await store.status.render()).changed).toBe("open gates 1->0");
    expect(await store.gates.list()).toEqual([]);
    expect(await store.standing.show()).toEqual([
      { number: 1, line: "Never force push." },
    ]);
  });

  it("resolves the ordered Graphite frontier and validates an optional pin", async () => {
    const { directory, store } = await initializedStore();
    const stack = await makeGitStack(directory);
    const output = `◯ main
◯ stack/merged
◯ stack/closed
◉ stack/open (current)
`;

    await withFakeGt({
      directory,
      output,
      operation: async () => {
        expect(await store.frontier.set({ repo: stack.repo })).toEqual({
          generation: 1,
          prs: [
            {
              pr: 10,
              branches: "stack/merged",
              sha: stack.mergedSha,
              state: "MERGED",
            },
            {
              pr: 13,
              branches: "stack/closed",
              sha: stack.closedSha,
              state: "CLOSED",
            },
            {
              pr: 11,
              branches: "stack/open",
              sha: stack.openSha,
              state: "OPEN",
            },
          ],
          lowestUnmerged: 11,
        });
        expect(
          (
            await store.frontier.set({
              repo: stack.repo,
              prs: [10, 13, 11],
            })
          ).generation
        ).toBe(2);
        expect((await store.frontier.show()).generation).toBe(2);
        await expect(
          store.frontier.set({
            repo: stack.repo,
            prs: [10, 11, 12],
          })
        ).rejects.toThrow(
          "frontier pin mismatch: missing from gt: 12; extra in gt: 13"
        );
        await expect(
          store.frontier.set({
            repo: stack.repo,
            prs: [13, 10, 11],
          })
        ).rejects.toThrow(
          "frontier pin mismatch: order differs: expected 13,10,11; gt 10,13,11"
        );
        await expect(
          store.frontier.set({
            repo: stack.repo,
            prs: [10, 10],
          })
        ).rejects.toThrow("--prs must not contain duplicates");
      },
    });
  });

  it("rejects unparseable Graphite output loudly", async () => {
    const { directory, store } = await initializedStore();
    const stack = await makeGitStack(directory);

    await withFakeGt({
      directory,
      output: "◯ main\nthis line is not Graphite output\n",
      operation: async () => {
        await expect(
          store.frontier.set({ repo: stack.repo })
        ).rejects.toThrow(
          'gt log short output has an unparseable line 2: "this line is not Graphite output"'
        );
      },
    });
  });

  it("rejects malformed TSV, verdict, frontier, and inbox data", async () => {
    const { directory, store } = await initializedStore();

    await writeFile(join(directory, "units.tsv"), "wrong\n");
    await expect(store.units.list()).rejects.toThrow(
      "units.tsv has an invalid header"
    );
    await writeFile(
      join(directory, "units.tsv"),
      "id\ttrack\tstate\tbranch\tpr\tsha\tbrief\nshort\trow\n"
    );
    await expect(store.units.list()).rejects.toThrow(
      "units.tsv has a malformed row"
    );

    await writeFile(
      join(directory, "ledger.tsv"),
      "pr\tsha\tverdict\tevidence\tverifier\tts\n1\tsha\tinvalid\treport\tme\tnow\n"
    );
    await expect(store.ledger.summary()).rejects.toThrow(
      "ledger.tsv has invalid verdict invalid"
    );

    await writeFile(join(directory, "frontier.json"), '{"generation":"1"}\n');
    await expect(store.frontier.show()).rejects.toThrow(
      "frontier.json has an invalid shape"
    );

    await writeFile(join(directory, "inbox", "bad.tsv"), "too\tshort\n");
    await expect(store.inbox.peek()).rejects.toThrow(
      "inbox pointer bad.tsv is malformed"
    );
  });

  it("rejects operations after close", async () => {
    const { store } = await initializedStore();
    await store.close();
    await expect(store.units.list()).rejects.toThrow("store is closed");
    await expect(store.status.render()).rejects.toBeInstanceOf(UserError);
  });
});

describe("orch CLI", () => {
  it("prints commander help and rejects invalid parsing with exit 1", async () => {
    const help = runCli(["--help"]);
    expect(help.code).toBe(0);
    expect(help.stdout).toContain("Commands:");
    expect(help.stdout).toContain("unit");
    expect(help.stdout).toContain("ledger");

    const frontierHelp = runCli(["frontier", "set", "--help"]);
    expect(frontierHelp.code).toBe(0);
    expect(frontierHelp.stdout).toContain("--repo <dir>");
    expect(frontierHelp.stdout).toContain("--prs <n,...>");

    const directory = await makeDirectory();
    const invalid = runCli(["--store", directory, "unit", "add", "u1"]);
    expect(invalid.code).toBe(1);
    expect(invalid.stderr).toContain("required option '--track <track>'");
  });

  it("accepts ORCH_STORE and emits complete JSON", async () => {
    const directory = await makeDirectory();
    const env = { ...process.env, ORCH_STORE: directory };
    expect(runCli(["init"], env).code).toBe(0);

    const added = runCli(
      ["unit", "add", "u1", "--track", "build", "--json"],
      env
    );
    expect(added.code).toBe(0);
    expect(JSON.parse(added.stdout)).toEqual({
      id: "u1",
      track: "build",
      state: "pending",
      branch: "",
      pr: "",
      sha: "",
      brief: "",
    });
  });

  it("maps user and not-found outcomes to the preserved exit codes", async () => {
    const directory = await makeDirectory();
    expect(runCli(["--store", directory, "init"]).code).toBe(0);

    const missingRepo = runCli([
      "--store",
      directory,
      "frontier",
      "set",
    ]);
    expect(missingRepo.code).toBe(1);
    expect(missingRepo.stderr).toContain(
      "set --repo <dir> or ORCH_REPO"
    );

    const userError = runCli([
      "--store",
      directory,
      "unit",
      "add",
      "",
      "--track",
      "build",
    ]);
    expect(userError.code).toBe(1);
    expect(userError.stderr).toContain("unit id must not be empty");

    const missingUnit = runCli([
      "--store",
      directory,
      "unit",
      "get",
      "missing",
    ]);
    expect(missingUnit.code).toBe(2);
    expect(missingUnit.stderr).toContain("unit missing not found");

    const missingLedger = runCli([
      "--store",
      directory,
      "--json",
      "ledger",
      "check",
      "184530",
      "abc123",
    ]);
    expect(missingLedger.code).toBe(2);
    expect(JSON.parse(missingLedger.stdout)).toEqual({
      pr: "184530",
      sha: "abc123",
      verdict: "NOT-VERIFIED",
    });
    expect(missingLedger.stderr).toBe("");
  });
});

const SLUG = "acme/widgets";

function sha(seed: number): string {
  return seed.toString(16).padStart(40, "0");
}

function pull({
  number,
  head,
  base,
  merged = false,
  fork = false,
}: {
  number: number;
  head: string;
  base: string;
  merged?: boolean;
  fork?: boolean;
}): unknown {
  return {
    number,
    state: merged ? "closed" : "open",
    merged_at: merged ? "2026-10-01T00:00:00Z" : null,
    head: {
      ref: head,
      sha: sha(number),
      repo: { full_name: fork ? "someone/widgets" : SLUG },
    },
    base: { ref: base, repo: { full_name: SLUG } },
  };
}

function filler(number: number): unknown {
  return pull({ number, head: `other/${number}`, base: "main" });
}

function openPage(page: number): string {
  return `repos/${SLUG}/pulls?state=open&per_page=100&page=${page}`;
}

async function makeRestRepo({
  branch,
  directory,
  remote = `git@github.com:${SLUG}.git`,
}: {
  branch: string;
  directory: string;
  remote?: string;
}): Promise<string> {
  const repo = join(directory, "repo");
  await mkdir(repo);
  git({ repo, args: ["init", `--initial-branch=${branch}`] });
  git({ repo, args: ["remote", "add", "origin", remote] });
  return repo;
}

async function withFakeGh<T>({
  directory,
  operation,
  responses,
}: {
  directory: string;
  operation: () => Promise<T>;
  responses: Readonly<Record<string, unknown>>;
}): Promise<T> {
  const bin = join(directory, "gh-bin");
  await mkdir(bin);
  const cases: string[] = [];
  for (const [index, [path, body]] of Object.entries(responses).entries()) {
    const file = join(directory, `gh-response-${index}.json`);
    await writeFile(file, JSON.stringify(body));
    cases.push(`  "${path}") cat "${file}" ;;`);
  }
  const gh = join(bin, "gh");
  await writeFile(
    gh,
    `#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *graphql*|*--paginate*)
    printf 'fake gh refuses: %s\\n' "$*" >&2
    exit 1
    ;;
esac
if [ "$#" -ne 2 ] || [ "$1" != "api" ]; then
  printf 'fake gh only answers "api <path>": %s\\n' "$*" >&2
  exit 1
fi
case "$2" in
${cases.join("\n")}
  *)
    printf 'fake gh has no fixture for %s\\n' "$2" >&2
    exit 1
    ;;
esac
`
  );
  await chmod(gh, 0o755);

  const originalPath = process.env.PATH;
  process.env.PATH = `${bin}:${originalPath ?? ""}`;
  try {
    return await operation();
  } finally {
    if (originalPath === undefined) {
      delete process.env.PATH;
    } else {
      process.env.PATH = originalPath;
    }
  }
}

describe("REST frontier source", () => {
  it("discovers a linear stack from its middle branch, trunk-up with API head SHAs", async () => {
    const directory = await makeDirectory();
    const repo = await makeRestRepo({ directory, branch: "stack/b" });
    expect(runCli(["--store", directory, "init"]).code).toBe(0);

    await withFakeGh({
      directory,
      responses: {
        [openPage(1)]: [
          pull({ number: 23, head: "stack/c", base: "stack/b" }),
          pull({ number: 30, head: "main", base: "main", fork: true }),
          pull({ number: 21, head: "stack/a", base: "main" }),
          filler(31),
          pull({ number: 22, head: "stack/b", base: "stack/a" }),
        ],
      },
      operation: async () => {
        const result = runCli([
          "--store",
          directory,
          "frontier",
          "set",
          "--repo",
          repo,
          "--source",
          "rest",
        ]);
        expect(result).toEqual({
          code: 0,
          stdout:
            "generation=1 prs=" +
            "stack/a#21@0000000000000000000000000000000000000015:OPEN," +
            "stack/b#22@0000000000000000000000000000000000000016:OPEN," +
            "stack/c#23@0000000000000000000000000000000000000017:OPEN" +
            " lowest-unmerged=21\n",
          stderr: "",
        });
      },
    });
  });

  it("pages the open-PR list by hand until a short page", async () => {
    const { directory, store } = await initializedStore();
    const repo = await makeRestRepo({ directory, branch: "stack/a" });
    const fillers = Array.from({ length: 98 }, (_, index) =>
      filler(100 + index)
    );

    await withFakeGh({
      directory,
      responses: {
        [openPage(1)]: [
          pull({ number: 21, head: "stack/a", base: "main" }),
          pull({ number: 22, head: "stack/b", base: "stack/a" }),
          ...fillers,
        ],
        [openPage(2)]: [
          filler(300),
          pull({ number: 23, head: "stack/c", base: "stack/b" }),
        ],
      },
      operation: async () => {
        expect(
          await store.frontier.set({ repo, source: "rest" })
        ).toEqual({
          generation: 1,
          prs: [
            { pr: 21, branches: "stack/a", sha: sha(21), state: "OPEN" },
            { pr: 22, branches: "stack/b", sha: sha(22), state: "OPEN" },
            { pr: 23, branches: "stack/c", sha: sha(23), state: "OPEN" },
          ],
          lowestUnmerged: 21,
        });
      },
    });
  });

  it("fails loudly past the page cap", async () => {
    const { directory, store } = await initializedStore();
    const repo = await makeRestRepo({ directory, branch: "stack/a" });
    const responses: Record<string, unknown> = {};
    for (let page = 1; page <= 10; page += 1) {
      responses[openPage(page)] = Array.from({ length: 100 }, (_, index) =>
        filler(page * 1000 + index)
      );
    }

    await withFakeGh({
      directory,
      responses,
      operation: async () => {
        await expect(
          store.frontier.set({ repo, source: "rest" })
        ).rejects.toThrow(
          "gh api listed more than 1000 open pull requests; refusing to read further"
        );
      },
    });
  });

  it("refuses a forked stack, a branch without a PR, and a non-GitHub origin", async () => {
    const { directory, store } = await initializedStore();
    const repo = await makeRestRepo({ directory, branch: "stack/b" });
    const forked = {
      [openPage(1)]: [
        pull({ number: 21, head: "stack/a", base: "main" }),
        pull({ number: 22, head: "stack/b", base: "stack/a" }),
        pull({ number: 24, head: "stack/b2", base: "stack/a" }),
      ],
    };

    await withFakeGh({
      directory,
      responses: forked,
      operation: async () => {
        await expect(
          store.frontier.set({ repo, source: "rest" })
        ).rejects.toThrow(
          "stack forks at branch stack/a: open pull requests 22,24 all target it"
        );
        git({ repo, args: ["symbolic-ref", "HEAD", "refs/heads/feature/none"] });
        await expect(
          store.frontier.set({ repo, source: "rest" })
        ).rejects.toThrow(
          "branch feature/none has no open pull request; pin the stack with --prs"
        );
        git({
          repo,
          args: [
            "remote",
            "set-url",
            "origin",
            "https://gitlab.com/acme/widgets.git",
          ],
        });
        await expect(
          store.frontier.set({ repo, source: "rest" })
        ).rejects.toThrow(
          "origin is not a github.com remote: https://gitlab.com/acme/widgets.git"
        );
      },
    });
  });

  it("validates a --prs pin against the open chain and keeps merged members", async () => {
    const { directory, store } = await initializedStore();
    const repo = await makeRestRepo({ directory, branch: "main" });

    await withFakeGh({
      directory,
      responses: {
        [openPage(1)]: [
          pull({ number: 21, head: "stack/a", base: "stack/z" }),
          pull({ number: 22, head: "stack/b", base: "stack/a" }),
          filler(25),
        ],
        [`repos/${SLUG}/pulls/20`]: pull({
          number: 20,
          head: "stack/z",
          base: "main",
          merged: true,
        }),
        [`repos/${SLUG}/pulls/21`]: pull({
          number: 21,
          head: "stack/a",
          base: "stack/z",
        }),
        [`repos/${SLUG}/pulls/22`]: pull({
          number: 22,
          head: "stack/b",
          base: "stack/a",
        }),
        [`repos/${SLUG}/pulls/25`]: filler(25),
      },
      operation: async () => {
        expect(
          await store.frontier.set({
            repo,
            source: "rest",
            prs: [20, 21, 22],
          })
        ).toEqual({
          generation: 1,
          prs: [
            { pr: 20, branches: "stack/z", sha: sha(20), state: "MERGED" },
            { pr: 21, branches: "stack/a", sha: sha(21), state: "OPEN" },
            { pr: 22, branches: "stack/b", sha: sha(22), state: "OPEN" },
          ],
          lowestUnmerged: 21,
        });
        await expect(
          store.frontier.set({ repo, source: "rest", prs: [21, 25] })
        ).rejects.toThrow(
          "frontier pin mismatch: missing from rest: 25; extra in rest: 22"
        );
      },
    });
  });

  it("selects the source from ORCH_FRONTIER_SOURCE", async () => {
    const directory = await makeDirectory();
    const repo = await makeRestRepo({
      directory,
      branch: "stack/a",
      remote: `https://github.com/${SLUG}.git`,
    });
    expect(runCli(["--store", directory, "init"]).code).toBe(0);

    await withFakeGh({
      directory,
      responses: {
        [openPage(1)]: [pull({ number: 21, head: "stack/a", base: "main" })],
      },
      operation: async () => {
        const result = runCli(["--store", directory, "frontier", "set"], {
          ...process.env,
          ORCH_FRONTIER_SOURCE: "rest",
          ORCH_REPO: repo,
        });
        expect(result).toEqual({
          code: 0,
          stdout:
            "generation=1 prs=stack/a#21@0000000000000000000000000000000000000015:OPEN lowest-unmerged=21\n",
          stderr: "",
        });
      },
    });
  });
});
