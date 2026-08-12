import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { basename, join, resolve } from "node:path";
import { spawnSync } from "node:child_process";

const checkFiles = process.argv.slice(2);
if (!checkFiles.length) {
  console.error("usage: node scripts/run-checks.mjs <check.ts> [...check.ts]");
  process.exit(2);
}

const tempRoot = mkdtempSync(join(tmpdir(), "caipu-admin-checks-"));
try {
  for (const [index, checkFile] of checkFiles.entries()) {
    const input = resolve(checkFile);
    const output = join(tempRoot, `${index}-${basename(checkFile, ".ts")}.mjs`);
    const build = spawnSync("esbuild", [
      input,
      "--bundle",
      "--platform=node",
      "--format=esm",
      "--alias:@=./src",
      `--outfile=${output}`,
    ], { stdio: "inherit" });
    if (build.error) throw build.error;
    if (build.status !== 0) throw new Error(`esbuild failed for ${checkFile} with status ${build.status ?? 1}`);

    const run = spawnSync(process.execPath, [output], { stdio: "inherit" });
    if (run.error) throw run.error;
    if (run.status !== 0) throw new Error(`check failed for ${checkFile} with status ${run.status ?? 1}`);
  }
} finally {
  rmSync(tempRoot, { recursive: true, force: true });
}
