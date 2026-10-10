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
        for name in (
            "GITHUB_ACTIONS",
            "TGO_LOCAL_VALIDATION_LOCK_HELD",
            "TGO_LOCAL_VALIDATION_VERIFY_CGROUP",
            "TGO_LOCAL_VALIDATION_CGROUP_ROOT",
            "TGO_LOCAL_VALIDATION_CGROUP_FILE",
            "TGO_LOCAL_VALIDATION_MEMINFO",
            "TGO_LOCAL_VALIDATION_MIN_AVAILABLE_KB",
            "TGO_LOCAL_VALIDATION_SYSTEMD_RUN",
        ):
            environment.pop(name, None)
        return environment

    def resource_environment(
        self, directory: Path, available_kb: int = 6291456
    ) -> dict[str, str]:
        meminfo = directory / "meminfo"
        meminfo.write_text(f"MemAvailable: {available_kb} kB\n")
        cgroup_root = directory / "cgroup"
        control_dir = cgroup_root / "test.scope"
        control_dir.mkdir(parents=True, exist_ok=True)
        (control_dir / "memory.max").write_text("4294967296\n")
        (control_dir / "memory.swap.max").write_text("0\n")
        (control_dir / "memory.oom.group").write_text("1\n")
        cgroup_file = directory / "self-cgroup"
        cgroup_file.write_text("0::/test.scope\n")
        systemd_run = directory / "systemd-run"
        systemd_run.write_text(
            "#!/bin/sh\n"
            'test -z "${SYSTEMD_RUN_ARGS:-}" || printf "%s\\n" "$@" > "$SYSTEMD_RUN_ARGS"\n'
            'test -z "${SYSTEMD_RUN_FAILURE:-}" || exit "$SYSTEMD_RUN_FAILURE"\n'
            'while test "$1" != --; do shift; done\n'
            "shift\n"
            "exec env TGO_LOCAL_VALIDATION_LOCK_HELD=1 "
            'TGO_LOCAL_VALIDATION_VERIFY_CGROUP=1 "$@"\n'
        )
        systemd_run.chmod(0o755)
        environment = self.clean_environment()
        environment["TGO_LOCAL_VALIDATION_MEMINFO"] = str(meminfo)
        environment["TGO_LOCAL_VALIDATION_SYSTEMD_RUN"] = str(systemd_run)
        environment["TGO_LOCAL_VALIDATION_CGROUP_ROOT"] = str(cgroup_root)
        environment["TGO_LOCAL_VALIDATION_CGROUP_FILE"] = str(cgroup_file)
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
                env=self.resource_environment(directory),
            )
            deadline = time.monotonic() + 1
            while not events.exists() and time.monotonic() < deadline:
                time.sleep(0.01)
            self.assertTrue(events.exists())
            second = subprocess.Popen(
                [LOCK_SCRIPT, "sh", "-c", command, "sh", "second", events],
                cwd=worktree,
                env=self.resource_environment(directory),
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
                env=self.resource_environment(Path(temporary)),
                timeout=2,
            )
            self.assertEqual(result.read_text(), "complete\n")

    def test_rejects_insufficient_available_memory(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            result = subprocess.run(
                [LOCK_SCRIPT, "sh", "-c", "exit 0"],
                cwd=repository,
                env=self.resource_environment(directory, 6291455),
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn("needs 6291456 kB available", result.stderr)

    def test_applies_memory_limit_at_admission_boundary(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            arguments = directory / "arguments"
            environment = self.resource_environment(directory)
            environment["SYSTEMD_RUN_ARGS"] = str(arguments)
            subprocess.run(
                [LOCK_SCRIPT, "sh", "-c", 'test "$TGO_LOCAL_VALIDATION_LOCK_HELD" = 1'],
                cwd=repository,
                env=environment,
                check=True,
            )
            self.assertIn("--property=MemoryMax=4G", arguments.read_text().splitlines())
            self.assertIn("--property=MemorySwapMax=0", arguments.read_text().splitlines())
            self.assertIn("--property=OOMPolicy=kill", arguments.read_text().splitlines())
            self.assertNotIn(
                "--property=MemoryOOMGroup=yes", arguments.read_text().splitlines()
            )

    def test_reports_cgroup_start_failure(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            environment = self.resource_environment(directory)
            environment["SYSTEMD_RUN_FAILURE"] = "125"
            result = subprocess.run(
                [LOCK_SCRIPT, "sh", "-c", "exit 0"],
                cwd=repository,
                env=environment,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 125)
            self.assertIn("status 125 inside the 4G process-tree memory limit", result.stderr)

    def test_rejects_missing_cgroup_runner(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            environment = self.resource_environment(directory)
            environment["TGO_LOCAL_VALIDATION_SYSTEMD_RUN"] = str(directory / "missing")
            result = subprocess.run(
                [LOCK_SCRIPT, "sh", "-c", "exit 0"],
                cwd=repository,
                env=environment,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn("needs systemd-run", result.stderr)

    def test_preserves_memory_kill_status(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            environment = self.resource_environment(directory)
            environment["SYSTEMD_RUN_FAILURE"] = "137"
            result = subprocess.run(
                [LOCK_SCRIPT, "sh", "-c", "exit 0"],
                cwd=repository,
                env=environment,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 137)
            self.assertIn("status 137 inside the 4G process-tree memory limit", result.stderr)
            self.assertIn("an out-of-memory kill is possible", result.stderr)

    def test_rejects_missing_effective_cgroup_limit(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            started = directory / "started"
            environment = self.resource_environment(directory)
            (directory / "cgroup/test.scope/memory.max").unlink()
            result = subprocess.run(
                [LOCK_SCRIPT, "sh", "-c", f"touch {started}"],
                cwd=repository,
                env=environment,
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 1)
            self.assertFalse(started.exists())
            self.assertIn("does not enforce the requested memory limits", result.stderr)

    def test_rejects_wrong_effective_cgroup_limit(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            started = directory / "started"
            environment = self.resource_environment(directory)
            control_dir = directory / "cgroup/test.scope"
            limits = {
                "memory.max": ("4294967296\n", "max\n"),
                "memory.swap.max": ("0\n", "max\n"),
                "memory.oom.group": ("1\n", "0\n"),
            }
            for name, (valid, invalid) in limits.items():
                with self.subTest(name=name):
                    path = control_dir / name
                    path.write_text(invalid)
                    result = subprocess.run(
                        [LOCK_SCRIPT, "sh", "-c", f"touch {started}"],
                        cwd=repository,
                        env=environment,
                        text=True,
                        capture_output=True,
                    )
                    self.assertEqual(result.returncode, 1)
                    self.assertFalse(started.exists())
                    self.assertIn(
                        "does not enforce the requested memory limits", result.stderr
                    )
                    path.write_text(valid)

    def test_preserves_command_failure_status(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            repository = self.make_repository(directory)
            result = subprocess.run(
                [LOCK_SCRIPT, "sh", "-c", "exit 23"],
                cwd=repository,
                env=self.resource_environment(directory),
                text=True,
                capture_output=True,
            )
            self.assertEqual(result.returncode, 23)
            self.assertIn("status 23 inside the 4G process-tree memory limit", result.stderr)

    def test_hosted_ci_bypasses_local_resource_controls(self) -> None:
        environment = self.clean_environment()
        environment["GITHUB_ACTIONS"] = "true"
        environment["TGO_LOCAL_VALIDATION_MEMINFO"] = "/missing"
        environment["TGO_LOCAL_VALIDATION_SYSTEMD_RUN"] = "/missing"
        subprocess.run(
            [LOCK_SCRIPT, "sh", "-c", 'test "$TGO_LOCAL_VALIDATION_LOCK_HELD" = 1'],
            cwd=ROOT,
            env=environment,
            check=True,
        )


if __name__ == "__main__":
    unittest.main()
