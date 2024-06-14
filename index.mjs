#!/usr/bin/env node

import { Command } from "commander";
import blessed from "blessed";
import contrib from "blessed-contrib";
import chalk from "chalk";
import { execSync } from "child_process";
import fs from "fs";
import path from "path";

const program = new Command();
program.parse(process.argv);

const screen = blessed.screen();
const grid = new contrib.grid({ rows: 12, cols: 12, screen: screen });
const table = grid.set(0, 0, 12, 12, contrib.table, {
  keys: true,
  fg: "white",
  interactive: false,
  label: "Git Status Monitor",
  width: "100%",
  height: "100%",
  border: { type: "line", fg: "cyan" },
  columnSpacing: 2,
  columnWidth: [60, 40, 10],
});

const getGitStatus = (repoPath, cachedData) => {
  const repoKey = repoPath.replace(`${process.cwd()}/`, "");

  if (
    cachedData[repoKey] &&
    cachedData[repoKey].timestamp > Date.now() - 5000
  ) {
    return cachedData[repoKey].status;
  }

  try {
    const status = execSync(`cd ${repoPath} && git status --porcelain`, {
      encoding: "utf-8",
    });

    const ahead = execSync(
      `cd ${repoPath} && git rev-list --count @{u}..HEAD 2>/dev/null || echo "0"`,
      { encoding: "utf-8" }
    ).trim();

    const behind = execSync(
      `cd ${repoPath} && git rev-list --count HEAD..@{u} 2>/dev/null || echo "0"`,
      { encoding: "utf-8" }
    ).trim();

    let statusIcon;
    if (status.trim() === "" && ahead === "0" && behind === "0") {
      statusIcon = chalk.green("✓");
    } else if (ahead !== "0") {
      statusIcon = chalk.yellow(`↑ ${ahead} ahead`);
    } else if (behind !== "0") {
      statusIcon = chalk.yellow(`↓ ${behind} behind`);
    } else {
      statusIcon = chalk.red("✕");
    }

    cachedData[repoKey] = {
      status: statusIcon,
      timestamp: Date.now(),
    };

    return statusIcon;
  } catch (error) {
    // return chalk.red("✕");
    // return the error message AND the icon
    cachedData[repoKey] = {
      status: chalk.red("✕"),
      timestamp: Date.now(),
    };

    return `${chalk.red("✕")} ${error.message}`;
  }
};

const getGitRepos = (dir) => {
  const gitRepos = [];
  const items = fs.readdirSync(dir);

  items.forEach((item) => {
    const itemPath = path.join(dir, item);
    const stat = fs.statSync(itemPath);

    if (stat.isDirectory() && fs.existsSync(path.join(itemPath, ".git"))) {
      gitRepos.push({
        path: itemPath,
        lastModified: stat.mtimeMs,
      });
    }
  });

  // Sort the gitRepos array by lastModified in descending order
  gitRepos.sort((a, b) => b.lastModified - a.lastModified);

  // Return only the paths of the sorted gitRepos
  return gitRepos.map((repo) => repo.path);
};

const cacheFile = path.join(process.cwd(), ".status-monitor.json");
let cachedData = {};

if (fs.existsSync(cacheFile)) {
  cachedData = JSON.parse(fs.readFileSync(cacheFile, "utf-8"));
}

const updateTable = () => {
  const gitRepos = getGitRepos(process.cwd());
  const tableData = gitRepos.map((repo) => {
    const repoName = repo.replace(`${process.cwd()}/`, "");
    const status = getGitStatus(repo, cachedData);
    let formattedRepoName = repoName;

    if (status.includes("ahead")) {
      formattedRepoName = chalk.yellow(`↑ ${repoName}`);
    } else if (status.includes("behind")) {
      formattedRepoName = chalk.yellow(`↓ ${repoName}`);
    } else if (status.includes("✕")) {
      formattedRepoName = chalk.red(`✕ ${repoName}`);
    }

    return [formattedRepoName, status, ""];
  });

  table.setData({ headers: ["Repository", "Status", ""], data: tableData });
  screen.render();

  fs.writeFileSync(cacheFile, JSON.stringify(cachedData), "utf-8");
};

updateTable();
setInterval(updateTable, 1000);

screen.key(["escape", "q", "C-c"], () => process.exit(0));
