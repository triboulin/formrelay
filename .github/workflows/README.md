# CI/CD workflow

This repository uses GitHub Actions with a self-hosted runner to build and publish the Docker image to the registry at registry.triboulin.fr.

## Requirements

- A self-hosted runner must be registered for this repository.
- The runner machine must be able to authenticate to the registry (for example via Docker config / local credentials).
- The runner must have Docker available.

## Behavior

When a Git tag is pushed, the workflow:

1. reads the tag name,
2. builds the Docker image,
3. pushes it as registry.triboulin.fr/formrelay-admin:<tag>.

Example:

```bash
git tag v1.2.3
git push origin v1.2.3
```

This will publish the image as registry.triboulin.fr/formrelay-admin:v1.2.3.
