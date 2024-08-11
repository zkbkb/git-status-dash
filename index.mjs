#!/usr/bin/env node

import { Command } from "commander";
import chalk from "chalk";
import { execSync } from "child_process";
import fs from "fs";
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

const getGitStatus = (repoPath) => {
  try {
    const status = execSync(`git -C "${repoPath}" status --porcelain`, {
      encoding: "utf-8",
    });
    const ahead = execSync(
      `git -C "${repoPath}" rev-list --count @{u}..HEAD 2>/dev/null || echo "0"`,
      { encoding: "utf-8" }
    ).trim();
    const behind = execSync(
      `git -C "${repoPath}" rev-list --count HEAD..@{u} 2>/dev/null || echo "0"`,
      { encoding: "utf-8" }
    ).trim();

    if (status.trim() === "" && ahead === "0" && behind === "0")
      return { symbol: "✓", message: "Up to date" };
    if (ahead !== "0" && behind !== "0")
      return {
        symbol: "↕",
        message: `Diverged (${ahead} ahead, ${behind} behind)`,
      };
    if (ahead !== "0")
      return { symbol: "↑", message: `${ahead} commit(s) to push` };
    if (behind !== "0")
      return { symbol: "↓", message: `${behind} commit(s) to pull` };
    return { symbol: "✗", message: "Uncommitted changes" };
  } catch (error) {
    return { symbol: "⚠", message: "Error accessing repository" };
  }
};

const getGitRepos = (dir) => {
  return fs
    .readdirSync(dir)
    .map((item) => path.join(dir, item))
    .filter(
      (itemPath) =>
        fs.statSync(itemPath).isDirectory() &&
        fs.existsSync(path.join(itemPath, ".git"))
    )
    .sort((a, b) => fs.statSync(b).mtimeMs - fs.statSync(a).mtimeMs);
};

const generateReport = () => {
  const gitRepos = getGitRepos(options.directory);
  const statuses = gitRepos.map((repo) => ({ repo, ...getGitStatus(repo) }));
  const unsynced = statuses.filter((status) => status.symbol !== "✓");

  console.log(chalk.bold("\nGit Repository Status Report"));
  console.log(`Total repositories: ${gitRepos.length}`);
  console.log(`Repositories needing attention: ${unsynced.length}`);

  const reposToShow = options.all ? statuses : unsynced;

  if (reposToShow.length > 0) {
    console.log("\nRepository Status:");
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

generateReport();
