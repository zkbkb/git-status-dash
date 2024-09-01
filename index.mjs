#!/usr/bin/env node

import { Command } from "commander";
import { spawn } from "child_process";
import fs from "fs/promises";
import path from "path";

const program = new Command();
program
  .option("-r, --report", "Generate a brief report")
  .option(
    "-d, --directory <path>",
    "Specify the directory to scan",
    process.cwd()
  )
  .option("-a, --all", "Show all repositories, including synced ones")
  .helpOption("-h, --help", "Display help for command")
  .addHelpText(
    "after",
    `
Examples:
  $ git-status-dash --report
  $ git-status-dash -d ~/projects -a

Status Information:
  ✓ Synced and up to date
  ↑ Ahead of remote (local commits to push)
  ↓ Behind remote (commits to pull)
  ↕ Diverged (need to merge or rebase)
  ✗ Uncommitted changes
  ⚠ Error accessing repository
  `
  )
  .parse(process.argv);

const options = program.opts();

const cache = new Map();

const runCommand = (command, args, cwd) => {
  return new Promise((resolve, reject) => {
    const process = spawn(command, args, { cwd });
    let output = "";
    process.stdout.on("data", (data) => (output += data));
    process.stderr.on("data", (data) => (output += data));
    process.on("close", (code) => {
      if (code === 0) resolve(output.trim());
      else reject(new Error(`Command failed with code ${code}`));
    });
  });
};

const getGitStatus = async (repoPath) => {
  const cacheKey = `${repoPath}-status`;
  if (cache.has(cacheKey)) return cache.get(cacheKey);

  try {
    const [status, ahead, behind] = await Promise.all([
      runCommand("git", ["-C", repoPath, "status", "--porcelain"]),
      runCommand("git", ["-C", repoPath, "rev-list", "--count", "@{u}..HEAD"]).catch(() => "0"),
      runCommand("git", ["-C", repoPath, "rev-list", "--count", "HEAD..@{u}"]).catch(() => "0")
    ]);

    let result;
    if (status === "" && ahead === "0" && behind === "0")
      result = { symbol: "✓", message: "Up to date" };
    else if (ahead !== "0" && behind !== "0")
      result = { symbol: "↕", message: `Diverged (${ahead} ahead, ${behind} behind)` };
    else if (ahead !== "0")
      result = { symbol: "↑", message: `${ahead} commit(s) to push` };
    else if (behind !== "0")
      result = { symbol: "↓", message: `${behind} commit(s) to pull` };
    else
      result = { symbol: "✗", message: "Uncommitted changes" };

    cache.set(cacheKey, result);
    return result;
  } catch (error) {
    const result = { symbol: "⚠", message: "Error accessing repository" };
    cache.set(cacheKey, result);
    return result;
  }
};

const isGitRepo = async (dirPath) => {
  try {
    const gitDir = path.join(dirPath, '.git');
    const stats = await fs.stat(gitDir);
    return stats.isDirectory();
  } catch (error) {
    return false;
  }
};

const getGitRepos = async (dir) => {
  const items = await fs.readdir(dir);
  const itemPaths = items.map(item => path.join(dir, item));
  const stats = await Promise.all(itemPaths.map(itemPath => fs.stat(itemPath)));
  
  const potentialRepos = itemPaths.filter((itemPath, index) => stats[index].isDirectory());
  const repoChecks = await Promise.all(potentialRepos.map(isGitRepo));
  const repos = potentialRepos.filter((_, index) => repoChecks[index]);

  return repos.sort((a, b) => stats[itemPaths.indexOf(b)].mtimeMs - stats[itemPaths.indexOf(a)].mtimeMs);
};

const generateReport = async () => {
  const gitRepos = await getGitRepos(options.directory);
  const statuses = await Promise.all(gitRepos.map(async repo => ({ repo, ...(await getGitStatus(repo)) })));
  const unsynced = statuses.filter(status => status.symbol !== "✓");

  //console.log("\nGit Repository Status Report");
  //console.log(`Total repositories: ${gitRepos.length}`);
  //console.log(`Repositories needing attention: ${unsynced.length}`);

  const reposToShow = options.all ? statuses : unsynced;

  if (reposToShow.length > 0) {
    //console.log("\nRepository Status:");
    const chalk = (await import("chalk")).default;
    reposToShow.forEach(({ repo, symbol, message }) => {
      const repoName = path.basename(repo);
      let line = `${symbol} ${repoName.padEnd(30)} ${message}`;
      switch (symbol) {
        case "✓":
          line = chalk.green(line);
          break;
        case "✗":
        case "⚠":
          line = chalk.red(line);
          break;
        case "↑":
        case "↓":
        case "↕":
          line = chalk.yellow(line);
          break;
      }
      console.log(line);
    });
  }
};

generateReport().catch(console.error);
