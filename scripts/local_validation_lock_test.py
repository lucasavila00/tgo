import os
from pathlib import Path
import subprocess
import tempfile
import time
import unittest


ROOT = Path(__file__).resolve().parents[1]
LOCK_SCRIPT = ROOT / "scripts" / "with-local-validation-lock.sh"


class LocalValidationLockTest(unittest.TestCase):
    def clean_environment(self) -> dict[str, str]:
        environment = os.environ.copy()
        environment.pop("GITHUB_ACTIONS", None)
        environment.pop("TGO_LOCAL_VALIDATION_LOCK_HELD", None)
        return environment

    def make_repository(self, directory: Path) -> Path:
        repository = directory / "repository"
        subprocess.run(["git", "init", "-q", repository], check=True)
        (repository / "tracked").write_text("tracked\n")
        subprocess.run(["git", "-C", repository, "add", "tracked"], check=True)
        subprocess.run(
            [
                "git",
                "-C",
                repository,
                "-c",
                "user.name=TGo Test",
                "-c",
                "user.email=test@example.com",
                "commit",
                "-qm",
                "Initial test commit",
            ],
            check=True,
        )
        return repository

    def test_excludes_another_worktree(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            worktree = directory / "worktree"
            subprocess.run(
                ["git", "-C", repository, "worktree", "add", "-q", worktree],
                check=True,
            )
            events = directory / "events"
            command = (
                f'echo "$1 start" >> "$2"; sleep 0.3; '
                f'echo "$1 end" >> "$2"'
            )
            first = subprocess.Popen(
                [LOCK_SCRIPT, "sh", "-c", command, "sh", "first", events],
                cwd=repository,
                env=self.clean_environment(),
            )
            deadline = time.monotonic() + 1
            while not events.exists() and time.monotonic() < deadline:
                time.sleep(0.01)
            self.assertTrue(events.exists())
            second = subprocess.Popen(
                [LOCK_SCRIPT, "sh", "-c", command, "sh", "second", events],
                cwd=worktree,
                env=self.clean_environment(),
            )
            self.assertEqual(first.wait(timeout=2), 0)
            self.assertEqual(second.wait(timeout=2), 0)
            self.assertEqual(
                events.read_text().splitlines(),
                ["first start", "first end", "second start", "second end"],
            )

    def test_nested_make_does_not_take_the_lock_again(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repository = self.make_repository(Path(temporary))
            result = repository / "result"
            makefile = repository / "Makefile"
            makefile.write_text(
                "outer:\n"
                f"\t+@{LOCK_SCRIPT} $(MAKE) inner\n"
                "inner:\n"
                f"\t@{LOCK_SCRIPT} sh -c '"
                'test "$$TGO_LOCAL_VALIDATION_LOCK_HELD" = 1 && '
                f"echo complete > {result}'\n"
            )

            subprocess.run(
                ["make", "outer"],
                cwd=repository,
                check=True,
                env=self.clean_environment(),
                timeout=2,
            )
            self.assertEqual(result.read_text(), "complete\n")


if __name__ == "__main__":
    unittest.main()
