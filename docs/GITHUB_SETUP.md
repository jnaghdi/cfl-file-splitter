# Create the GitHub repository

The download is a prepared local source project. A remote repository cannot be
created without an authenticated GitHub account and permission to create it.
No GitHub account, owner or remote URL is assumed or embedded in the package.

## Recommended route on Windows

Install Git for Windows and the official GitHub CLI (`gh`) through approved
channels. Extract the package and open PowerShell **in `cfl-file-splitter`**.
Then:

```powershell
gh auth login --hostname github.com --git-protocol https --web
.\scripts\publish-github.ps1
```

Login is handled by GitHub CLI/browser; never paste a password or token into a
chat or source file. Review any credential-storage prompt from `gh` and follow
your organisation's policy. The script uses the active GitHub.com account.

The script verifies the source inventory, refuses an existing origin/repository,
initialises `main` when needed, stages only the inventoried source files, lists
those paths and shows the account/target. It requires typing `CREATE` before
creating **a private** repository and pushing. It does not publish a release,
make a public repository, overwrite another remote or force-push.

The first commit uses your existing Git author configuration. If none exists,
it derives a repository-local author and GitHub noreply email from the account
returned by `gh api user`; it does not alter global Git identity settings.

For an organisation where your account has repository-creation permission:

```powershell
.\scripts\publish-github.ps1 -Repository "YOUR-ORG/cfl-file-splitter"
```

A script execution-policy restriction should be handled through your normal
approved script-review process, not by disabling endpoint controls.

## Equivalent manual commands

Review all files before staging, especially any files you added yourself.
After authenticating and setting your normal Git author configuration:

```powershell
git init -b main
git add .
git diff --cached --stat
git commit -m "Initial CFL File Splitter 1.0.1 source repository"
gh repo create cfl-file-splitter --private --source . --remote origin --push --disable-wiki --description "Windows file splitter, verifier and joiner with SHA-256-verified self-identifying parts."
```

These commands are for a **new checkout without a remote**. Do not run them
blindly in an unrelated existing repository. `git add .` stages all unignored
files, so inspect the staged content carefully. The provided script is more
restrictive and uses SOURCE_SHA256SUMS.txt as its explicit file inventory.

## After the first push

Open the actual repository URL printed by the script. Check the Actions tab for
real results; no CI status is preclaimed. Configure branch protection/repository
rules, reviewers and private security reporting according to your account's
available features. Consider requiring the Linux and Windows checks before
merging. Keep the project private pending licensing review.

The source tree contains no EXEs by design. Successful Windows CI builds upload
a ZIP artifact. Tagging the exact application version creates a draft release:

```powershell
git tag -a v1.0.1 -m "CFL File Splitter 1.0.1"
git push origin v1.0.1
```

Do this only after checking CI and following `RELEASE_CHECKLIST.md`. The release
workflow tests again and creates a draft; the owner reviews and publishes it
manually. Authentication permissions may need to allow committing workflow files
and running Actions. Organisation policy can limit private repository creation
or workflow execution; use an authorised account and do not bypass those rules.

## A partial failure during creation or push

The script stops on failure and never deletes a remote to retry. A failed push
can leave a local commit and/or an empty private repository. Inspect
`git remote -v`, `git status` and the repository in GitHub before taking further
action. Once the correct origin and authentication have been confirmed, a normal
`git push -u origin main` can complete the upload. Do not force-push or repeatedly
create repositories to conceal a permissions or network error.

## Official references

Checked when preparing this repository, 2026-09-18:
- https://cli.github.com/manual/gh_auth_login
- https://cli.github.com/manual/gh_repo_create
- https://cli.github.com/manual/gh_release_create
- https://docs.github.com/en/actions/tutorials/build-and-test-code/go
