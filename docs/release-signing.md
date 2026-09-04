# Release signing

Windows release assets are Authenticode-signed through
[SignPath Foundation](https://signpath.org/). This identifies the publisher and
allows Microsoft SmartScreen to build reputation for subsequent releases. A new
certificate may still need time to establish reputation, so signing cannot
guarantee that SmartScreen immediately suppresses every warning.

The release workflow deliberately fails before publication when signing is not
configured or either signature is invalid. Local development builds remain
unsigned.

## One-time SignPath setup

1. [Apply to SignPath Foundation](https://signpath.org/apply) for this public
   repository.
2. Connect the approved SignPath project to
   `NolanMullins/wt-modern-8111` through the SignPath GitHub connector.
3. Create a release signing policy that accepts only builds from the `main`
   branch, then create two artifact configurations:
   - a ZIP artifact root containing one
     `wt-modern-windows-amd64.exe` portable PE file;
   - a ZIP artifact root containing one `wt-modern-setup.exe` installer PE
     file.

   GitHub Actions uploads artifacts as ZIP archives. Each configuration must
   therefore select the PE file inside the archive and apply Authenticode
   signing to it; a bare PE artifact root will not match the submitted
   artifact.
4. Configure these GitHub Actions repository variables:
   - `SIGNPATH_ORGANIZATION_ID`
   - `SIGNPATH_PROJECT_SLUG`
   - `SIGNPATH_SIGNING_POLICY_SLUG`
   - `SIGNPATH_PORTABLE_ARTIFACT_CONFIGURATION_SLUG`
   - `SIGNPATH_INSTALLER_ARTIFACT_CONFIGURATION_SLUG`
5. Configure `SIGNPATH_API_TOKEN` as a GitHub Actions repository secret.

The project, policy, and artifact configuration slugs must match the values
created during SignPath onboarding. The API token must be permitted to submit
requests for the release signing policy.

## Release order

The workflow:

1. builds and submits the portable executable to SignPath;
2. packages the signed portable executable inside the installer;
3. submits the installer to SignPath;
4. verifies both Authenticode signatures locally;
5. generates checksums from the signed files;
6. publishes the release.

Never restore an unsigned fallback. A signing outage should block a release
rather than publish binaries that Windows identifies as an unknown publisher.
