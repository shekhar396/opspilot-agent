# Release Process

This repository is prepared for tagged releases; this document does not imply that a release already exists.

## Prepare a release

1. Ensure the `main` branch and working tree are clean.
2. Pull the latest reviewed changes.
3. Run `make check`.
4. Review version-facing changes and release notes.
5. Choose a semantic version such as `v0.1.0`.
6. Create and push an annotated tag:

```bash
git tag -a v0.1.0 -m "OpsPilot Agent v0.1.0"
git push origin v0.1.0
```

The tag triggers the release workflow. It uses the tag as `Version`, the GitHub commit SHA as `Commit`, and one generated UTC timestamp as `Date` for both Linux architectures. The workflow runs formatting verification, tests, race tests, and vet; packages the binaries; verifies checksums and amd64 metadata; and creates a GitHub Release with generated notes.

## Verify the release

1. Confirm the GitHub Release has both `.tar.gz` archives and `checksums.txt`.
2. Download all artifacts into an empty directory.
3. Run `sha256sum -c checksums.txt`.
4. Extract the archive for the test machine.
5. Run `./opspilot-agent version` from its versioned archive directory and compare the tag, commit, and build date.
6. Validate the packaged example configuration.
7. Test installation, service operation, upgrade, non-purge uninstall, reinstall, and purge on a clean supported Linux VM using a controlled HTTPS endpoint.
8. Publish or edit the generated release notes as needed.

Release builds are static (`CGO_ENABLED=0`) and use `-trimpath`. They are not claimed to be bit-for-bit reproducible. Releases are not currently signed and do not include SBOMs or provenance attestations.

## Prereleases

Use semantic prerelease tags such as:

```text
v0.2.0-rc.1
```

Tags containing `-rc`, `-beta`, or `-alpha` are marked as prereleases by the workflow.

## Rollback and corrections

If a release is incorrect, remove the GitHub Release when necessary, but do not silently replace artifacts that may already have been distributed. Prefer a corrected patch release. Avoid deleting and recreating public tags. A tag push is an intentional publishing action and must occur only after review.
