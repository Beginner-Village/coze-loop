# @coze-arch/bot-flags

feature gating for bot studio

## Overview

This package is part of the YNET Studio monorepo and provides architecture functionality. It serves as a core component in the YNET ecosystem.

## Getting Started

### Installation

Add this package to your `package.json`:

```json
{
  "dependencies": {
    "@coze-arch/bot-flags": "workspace:*"
  }
}
```

Then run:

```bash
rush update
```

### Usage

```typescript
import { /* exported functions/components */ } from '@coze-arch/bot-flags';

// Example usage
// TODO: Add specific usage examples
```

## Features

- Core functionality for YNET Studio
- TypeScript support
- Modern ES modules

## API Reference

### Exports

- `type FEATURE_FLAGS, type FetchFeatureGatingFunction`
- `getFlags`
- `useFlags`
- `pullFeatureFlags`


For detailed API documentation, please refer to the TypeScript definitions.

## Development

This package is built with:

- TypeScript
- Modern JavaScript
- Vitest for testing
- ESLint for code quality

## Contributing

This package is part of the YNET Studio monorepo. Please follow the monorepo contribution guidelines.

## License

Apache-2.0
