# Contributing to BlueKing Auth

The BlueKing team embraces an open attitude and welcomes developers who share the same vision to contribute to the project. Before getting started, please carefully read the following guidelines.

## Code License

[MIT LICENSE](../LICENSE.txt) is the open-source license for BlueKing Auth. Any code contributions will be protected by this license, so please confirm whether you accept this agreement before contributing code.

## Contributing Features and Enhancements

If you want to contribute features and enhancements to the `blueking-auth` project, please follow these steps:

- Check existing [Issues](https://github.com/TencentBlueKing/blueking-auth/issues) to see if there are any issues related to the desired feature. If so, discuss it in that issue.
- If there is no relevant issue, you can create a new issue to describe your feature request. The BlueKing team will periodically check and participate in the discussion.
- If the team agrees on the feature, you need to provide further details, such as design, implementation details, test cases, etc., in the issue.
- Following the [BlueKing Development Specification](https://bk.tencent.com/docs/document/7.0/250/46218), complete the feature coding, add corresponding unit tests, and update the documentation.
- If it is your first time contributing to a BlueKing project, you will also need to sign the [Tencent Contributor License Agreement](https://bk-cla.bktencent.com/TencentBlueKing/blueking-auth).
- Submit a [Pull Request](https://github.com/TencentBlueKing/blueking-auth/pulls) to the main branch and link it to the corresponding issue. The PR should include code, documentation, and unit tests.
- The BlueKing team will promptly review the PR, and upon approval, merge it into the main branch.

> Note: To ensure code quality, for large features and enhancements, without affecting the existing service functions, the BlueKing team recommends splitting requirements as much as possible and submitting multiple PRs for review. This practice helps shorten the review time.

## Getting Started

To contribute code, it is recommended to first set up your local development environment by referring to the following document:

- [Local Development Deployment Guide](DEVELOP_GUIDE.md)

## GIT Commit Convention

The BlueKing team recommends using **short and accurate** commit messages to describe the changes you made. The format is as follows:

```bash
git commit -m 'tag: concise summary of the commit'
```

Example:

```shell
git commit -m 'fix: fix abnormal display issue in deployment status page process creation time'
```

### Tag Descriptions

| Tag      | Description   |
|----------|---------------|
| feat     | New feature/development     |
| fix      | Fix an existing bug         |
| docs     | Add/modify documentation    |
| style    | Modify comments, format according to code standards, etc. |
| refactor | Code refactoring, architectural adjustments |
| perf     | Optimize configuration, parameters, logic, or functionality |
| test     | Add/modify unit test cases  |
| chore    | Adjust build scripts, tasks, etc. |

## Pull Request

If you are already working on an issue, have a reasonable solution, and have made positive progress, it is recommended to reply to that issue. This informs the BlueKing team, other developers, and users that you are interested in and actively working on the issue, preventing duplication of effort and avoiding waste of resources.

We welcome everyone to contribute code to build the BlueKing API Gateway, and we are happy to discuss solutions with everyone. We look forward to receiving your PRs.

For fixing issues, the BlueKing team hopes that a single PR covers all relevant content, including but not limited to code, fixed documents, and usage instructions.

> Note: Please ensure that the PR title follows the Git Commit Convention. During development, control the number of commits in a single PR to avoid unnecessary repeated submissions.

## Issues

The BlueKing team uses [Issues](https://github.com/TencentBlueKing/blueking-auth/issues) for tracking bugs and features.

When submitting a bug report, please check if there is an existing or similar issue to ensure there is no duplication.

If confirming it is a new bug, please include the following information when submitting an issue:

- Information about your operating system, language version, etc.
- Current version information you are using, such as version, commit id.
- Relevant module log output when the problem occurs (be careful not to include sensitive information).
- Accurate steps to reproduce the issue. Providing a reproduction script/tool will be more useful than a lengthy description.

Please assist in translating these guidelines.