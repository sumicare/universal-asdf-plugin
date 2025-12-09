# `argo-rollouts` asdf plugin

This is the `argo-rollouts` integration for the [universal-asdf-plugin](../../) project.

[Argo Rollouts](https://github.com/argoproj/argo-rollouts) is a Kubernetes controller that enables progressive delivery strategies such as blue-green, canary, and traffic-shifting deployments. It integrates with ingress controllers and service meshes to gradually roll out changes, monitor metrics, and automatically roll back when issues are detected.

## Usage

```bash
# List available versions
universal-asdf-plugin list-all argo-rollouts

# Install latest version
universal-asdf-plugin install argo-rollouts latest
```

## License

By using this project, you agree to the Sumicare OSS [Terms of Use](../../OSS_TERMS.md).

Licensed under the [Apache License, Version 2.0](../../LICENSE).