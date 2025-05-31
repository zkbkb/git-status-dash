# Git Status Monitor

<img width="1002" alt="Screenshot 2024-06-14 at 1 29 22 AM" src="https://github.com/ejfox/git-status-monitor/assets/530073/67b94585-f78e-4789-986b-25439b8ccce1">

Tired of constantly jumping between terminal tabs to check the status of your git repositories? Surprised to find a repo is way behind after making a few changes? Well, jump no more! Introducing the Git Status Monitor – your one-stop shop for keeping tabs on all your git repos in real-time.

## Getting Started 🏁

To get this party started, `npx git-status-dash` watches whatever directory it is run in.

You can also install it globally with `npm install -g git-status-dash` and run it from anywhere with `git-status-dash`.

```bash
# Opens in tui mode (thanks @zkbkb)
npx git-status-dash -t

# Open another directory
npx git-status-dash -d ~/code/

# Get help
npx git-status-dash --help
```

### Running by hand

Or, if you'd like to clone the repo, first make sure you have Node.js installed on your system. Then, follow these steps:

1. Clone this repository or download the `git-status-monitor.js` file.
2. Open a terminal and navigate to the directory containing the script.
3. Run `npm install` to install the required dependencies.
4. Make the script executable with `chmod +x git-status-monitor.js`.
5. Run the script with `./git-status-monitor.js`.

## What's it telling me? 🤔

The table displays the following information for each repository:

- Repository name (relative to the scanned directory)
- Status icon and details:
  - ✓ (green): The repository is in sync with the remote.
  - ↑ (yellow): The local branch is ahead of the remote by the specified number of commits.
  - ↓ (yellow): The local branch is behind the remote by the specified number of commits.
  - ✕ (red): There are uncommitted changes or the repository is not a valid git repo.

The repositories are sorted by the most recently modified ones at the top, so you can quickly see which repos need your attention.

## Happy Monitoring! 😄

Now go forth and conquer your git workflow with the power of the Git Status Monitor. May your commits be clean and your branches always in sync!

If you have any questions, suggestions, PRs are welcome. 
