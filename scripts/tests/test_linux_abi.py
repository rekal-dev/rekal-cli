"""Guard numeric version comparisons and malformed inspection output."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('abi', Path(__file__).parents[1] / 'check_linux_abi.py')
abi = importlib.util.module_from_spec(spec)
spec.loader.exec_module(abi)


class LinuxABITests(unittest.TestCase):
    def test_rejects_observed_release_regression(self):
        with self.assertRaisesRegex(ValueError, 'requires GLIBC 2.43'):
            abi.check('Version needs section:\n Name: GLIBC_2.39\n Name: GLIBC_2.43', '2.39')

    def test_numeric_versions_and_patch_components(self):
        self.assertEqual(abi.check('Version needs section:\n Name: GLIBC_2.9\n Name: GLIBC_2.2.5\n Name: GLIBC_2.39', '2.39'), '2.39')

    def test_fails_closed_on_missing_version_metadata(self):
        for output in ['', 'Name: GLIBC_2.39', 'Version needs section:\n Name: GLIBCXX_3.4.30']:
            with self.subTest(output=output), self.assertRaises(ValueError):
                abi.check(output, '2.39')
