#!/usr/bin/env node

import { Command } from 'commander';
import { spawn } from 'child_process';
import fs from 'fs/promises';
import path from 'path';
import blessed from 'blessed';
import contrib from 'blessed-contrib'; // default CJS import gives full API incl. .table


const program = new Command();
program
    .option('-r, --report', 'Generate a brief report')
    .option(
        '-d, --directory <path>',
        'Specify the directory to scan',
        process.cwd()
    )
    .option('-a, --all', 'Show all repositories, including synced ones')
    .option('-t, --tui', 'Interactive TUI interface')
    .option('--depth <n>', 'Limit recursion depth when scanning repos (default: unlimited)', (v) => parseInt(v, 10))
    .helpOption('-h, --help', 'Display help for command')
    .addHelpText(
        'after',
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

const depthLimit = Number.isInteger(options.depth) ? options.depth : Infinity;

const cache = new Map();

const runCommand = (command, args, cwd) => {
    return new Promise((resolve, reject) => {
        const process = spawn(command, args, { cwd });
        let output = '';
        process.stdout.on('data', (data) => (output += data));
        process.stderr.on('data', (data) => (output += data));
        process.on('close', (code) => {
            if (code === 0) resolve(output.trim());
            else reject(new Error(`Command failed with code ${code}`));
        });
    });
};

const getGitStatus = async(repoPath) => {
    const cacheKey = `${repoPath}-status`;
    if (cache.has(cacheKey)) return cache.get(cacheKey);

    try {
        const [status, ahead, behind, branch, lastCommit] = await Promise.all([
            runCommand('git', ['-C', repoPath, 'status', '--porcelain']),
            runCommand('git', ['-C', repoPath, 'rev-list', '--count', '@{u}..HEAD']).catch(() => '0'),
            runCommand('git', ['-C', repoPath, 'rev-list', '--count', 'HEAD..@{u}']).catch(() => '0'),
            runCommand('git', ['-C', repoPath, 'rev-parse', '--abbrev-ref', 'HEAD']).catch(() => ''),
            runCommand('git', ['-C', repoPath, 'log', '-1', '--pretty=%h %cr %an']).catch(() => '')
        ]);

        let result;
        if (status === '' && ahead === '0' && behind === '0')
            result = { symbol: '✓', message: 'Up to date' };
        else if (ahead !== '0' && behind !== '0')
            result = { symbol: '↕', message: `Diverged (${ahead} ahead, ${behind} behind)` };
        else if (ahead !== '0')
            result = { symbol: '↑', message: `${ahead} commit(s) to push` };
        else if (behind !== '0')
            result = { symbol: '↓', message: `${behind} commit(s) to pull` };
        else
            result = { symbol: '✗', message: 'Uncommitted changes' };

        result.branch = branch.trim();
        result.lastCommit = lastCommit.trim();

        cache.set(cacheKey, result);
        return result;
    } catch (error) {
        const result = { symbol: '⚠', message: 'Error accessing repository' };
        cache.set(cacheKey, result);
        return result;
    }
};

const isGitRepo = async(dirPath) => {
    try {
        const gitDir = path.join(dirPath, '.git');
        const stats = await fs.stat(gitDir);
        return stats.isDirectory();
    } catch (error) {
        return false;
    }
};

// Recursively walk directories to find all Git repositories
const walkRepos = async(dir, list = [], depthLeft = Infinity) => {
    // If the current directory itself is a repo, record it even at depth limit
    if (await isGitRepo(dir)) {
        list.push(dir);
        return list;
    }
    if (depthLeft === 0) return list;

    let entries;
    try {
        entries = await fs.readdir(dir, { withFileTypes: true });
    } catch {
        // Ignore unreadable directories
        return list;
    }

    // Skip common heavy folders to improve speed
    const SKIP = new Set(['.git', 'node_modules', '.cache', '.venv']);

    await Promise.all(
        entries
        .filter(e => e.isDirectory() && !SKIP.has(e.name))
        .map(e => walkRepos(path.join(dir, e.name), list, depthLeft - 1))
    );

    return list;
};

// Get all Git repositories under the target directory, sorted by mtime (recent first)
const getGitRepos = async(dir, depth = Infinity) => {
    const repos = await walkRepos(dir, [], depth);
    const stats = await Promise.all(repos.map(r => fs.stat(r)));
    return repos.sort((a, b) => {
        const mA = stats[repos.indexOf(a)].mtimeMs;
        const mB = stats[repos.indexOf(b)].mtimeMs;
        return mB - mA;
    });
};

const generateReport = async() => {
    const gitRepos = await getGitRepos(options.directory, depthLimit);
    process.stdout.write('\x1B[?25l');
    let dotCount = 0;
    const totalReposText = `Found ${gitRepos.length} repositories, loading`;
    const dotTimer = setInterval(() => {
        dotCount = (dotCount % 6) + 1;
        const dots = '.'.repeat(dotCount);
        const padding = ' '.repeat(6 - dotCount);
        process.stdout.write(`\r${totalReposText}${dots}${padding}`);
    }, 150);

    const statuses = await Promise.all(gitRepos.map(async repo => ({ repo, ...(await getGitStatus(repo)) })));
    clearInterval(dotTimer);
    // On completion, show exactly 6 dots and newline
    process.stdout.write(`\r${totalReposText}......\n`);
    process.stdout.write('\x1B[?25h');
    const unsynced = statuses.filter(status => status.symbol !== '✓');

    //console.log("\nGit Repository Status Report");
    //console.log(`Total repositories: ${gitRepos.length}`);
    //console.log(`Repositories needing attention: ${unsynced.length}`);

    const reposToShow = options.all ? statuses : unsynced;

    if (options.tui) {
        await renderTUI(reposToShow, options.directory);
        return;
    }
    if (reposToShow.length > 0) {
        const chalk = (await
            import ('chalk')).default;
        reposToShow.forEach(({ repo, symbol, message }) => {
            const repoName = path.basename(repo);
            let line = `${symbol} ${repoName.padEnd(30)} ${message}`;
            switch (symbol) {
                case '✓':
                    line = chalk.green(line);
                    break;
                case '✗':
                case '⚠':
                    line = chalk.red(line);
                    break;
                case '↑':
                case '↓':
                case '↕':
                    line = chalk.yellow(line);
                    break;
            }
            console.log(line);
        });
    }
};

const renderTUI = async(repos, baseDir) => {
    const screen = blessed.screen({ smartCSR: true, title: 'git-status-dash' });
    const grid = new contrib.grid({ rows: 12, cols: 12, screen });

    // Select a valid table constructor for the installed blessed‑contrib version
    const TableCtor = contrib.listtable || contrib.table || contrib.Table || (contrib.widgets && contrib.widgets.table);
    const widgetType = TableCtor ? TableCtor : 'table'; // fallback to string type
    const table = grid.set(0, 0, 12, 12, widgetType, {
        keys: true,
        interactive: true,
        label: `${path.basename(baseDir)} Repositories`,
        columnSpacing: 2,
        columnWidth: [3, 32, 25]
    });

    let refreshTimer = null;
    let detailBox = null;
    let currentRepos = repos;

    const updateTable = async (reposData) => {
        const rows = reposData.map(({ symbol, repo, message }) => {
            const name = path.basename(repo);
            return [symbol, name, message];
        });

        // Colour-code the status rows for TUI
        const chalkT = (await
            import ('chalk')).default;
        const colouredRows = rows.map(r => {
            const symbol = r[0];
            let colouredSymbol = chalkT.green(symbol);
            let colouredRepo = chalkT.green(r[1]);
            let colouredStatus = chalkT.green(r[2]);
            if (symbol === '✗' || symbol === '⚠') {
                colouredSymbol = chalkT.red(symbol);
                colouredRepo = chalkT.red(r[1]);
                colouredStatus = chalkT.red(r[2]);
            } else if (symbol === '↑' || symbol === '↓' || symbol === '↕') {
                colouredSymbol = chalkT.yellow(symbol);
                colouredRepo = chalkT.yellow(r[1]);
                colouredStatus = chalkT.yellow(r[2]);
            }
            return [colouredSymbol, colouredRepo, colouredStatus];
        });
        table.setData({ headers: ['S', 'Repo', 'Status'], data: colouredRows });
        currentRepos = reposData;
    };

    // Initial table update
    await updateTable(repos);

    const refreshData = async () => {
        cache.clear(); // Clear cache to get fresh data
        const gitRepos = await getGitRepos(baseDir, depthLimit);
        const statuses = await Promise.all(gitRepos.map(async repo => ({ repo, ...(await getGitStatus(repo)) })));
        const reposToShow = options.all ? statuses : statuses.filter(status => status.symbol !== '✓');
        await updateTable(reposToShow);
        screen.render();
    };

    // Set up auto-refresh every 15 seconds
    refreshTimer = setInterval(refreshData, 15000);

    blessed.text({
        parent: screen,
        bottom: 0,
        right: 2,
        content: 'Q to quit | SPACE for details | Auto-refresh: 15s',
        style: { fg: 'grey' }
    });

    screen.key(['q', 'C-c'], () => {
        if (refreshTimer) clearInterval(refreshTimer);
        process.exit(0);
    });
    screen.key(['space'], () => table.rows.enterSelected());

    table.rows.on('select', (_, idx) => {
        if (detailBox) {
            detailBox.destroy();
            detailBox = null;
            screen.render();
            return;
        }
        const details = currentRepos[idx];
        detailBox = blessed.box({
            parent: screen,
            top: 'center',
            left: 'center',
            width: '60%',
            height: 'shrink',
            label: 'Details',
            border: 'line',
            tags: true,
            padding: { left: 1, right: 1 },
            content: `repo: {bold}${details.repo}{/bold}\nbranch: ${details.branch}\nlast: ${details.lastCommit}\nstatus: ${details.message}`
        });
        screen.render();
    });

    table.focus();
    screen.render();
};

generateReport().catch(console.error);

// for testing
export { getGitRepos, walkRepos, isGitRepo, renderTUI };