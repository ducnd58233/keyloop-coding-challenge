#!/usr/bin/env python3
"""Contract tests for block_direct_main.decide. No network, no git."""

from __future__ import annotations

import unittest

from block_direct_main import decide, extract_command


class DecideTests(unittest.TestCase):
    def test_allows_feature_push(self) -> None:
        self.assertIsNone(
            decide("git push -u origin HEAD", "feature/doc-viewer-task-5")
        )
        self.assertIsNone(
            decide(
                "git push origin feature/doc-viewer-task-4-aggregation",
                "feature/doc-viewer-task-4-aggregation",
            )
        )

    def test_allows_gh_pr_merge(self) -> None:
        self.assertIsNone(decide("gh pr merge 1 --merge", "main"))
        self.assertIsNone(
            decide(
                "git push -u origin HEAD && gh pr create --base main --head feature/x",
                "feature/x",
            )
        )

    def test_allows_read_only_git(self) -> None:
        self.assertIsNone(decide("git status", "main"))
        self.assertIsNone(decide("git fetch origin && git pull origin main", "main"))
        self.assertIsNone(decide("git checkout main", "feature/x"))
        self.assertIsNone(decide("git merge origin/main", "feature/x"))
        self.assertIsNone(decide("git stash push -m wip", "main"))

    def test_allows_force_on_feature(self) -> None:
        self.assertIsNone(
            decide("git push --force-with-lease origin feature/x", "feature/x")
        )
        self.assertIsNone(decide("git push origin main:backup-main", "main"))

    def test_denies_direct_main(self) -> None:
        for command in (
            "git push origin main",
            "git push origin master",
            "git push origin HEAD:main",
            "git push origin +main",
            "git push --force origin main",
            "git push --force-with-lease origin main",
            "git push -f origin main",
            "git push origin :main",
            "git push --all",
            "git push --mirror",
        ):
            with self.subTest(command=command):
                self.assertIsNotNone(decide(command, "feature/x"))

    def test_denies_implicit_push_while_on_main(self) -> None:
        self.assertIsNotNone(decide("git push", "main"))
        self.assertIsNotNone(decide("git push origin", "main"))
        self.assertIsNotNone(decide("git push origin HEAD", "main"))
        self.assertIsNotNone(decide("git push -u origin HEAD", "master"))

    def test_denies_push_in_compound_command(self) -> None:
        self.assertIsNotNone(
            decide("make lint && git push origin main && gh pr merge 1 --merge", "feature/x")
        )

    def test_extracts_shell_and_tool_payloads(self) -> None:
        self.assertEqual(
            extract_command({"command": "git push origin main"}),
            "git push origin main",
        )
        self.assertEqual(
            extract_command({"tool_input": {"command": "git status"}}),
            "git status",
        )


if __name__ == "__main__":
    unittest.main()
