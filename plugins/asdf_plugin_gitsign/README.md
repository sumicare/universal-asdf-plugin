# `gitsign` asdf plugin

This is the `gitsign` integration for the [universal-asdf-plugin](../../) project.

[gitsign](https://github.com/sigstore/gitsign) is a tool for signing Git commits using keyless signatures. It leverages Sigstore infrastructure to issue ephemeral certificates and record signatures in transparency logs, improving the provenance of your Git history.

## Usage

```bash
# List available versions
universal-asdf-plugin list-all gitsign

# Install latest version
universal-asdf-plugin install gitsign latest
```

## License

By using this project, you agree to the Sumicare OSS [Terms of Use](../../OSS_TERMS.md).

Licensed under the [Apache License, Version 2.0](../../LICENSE).